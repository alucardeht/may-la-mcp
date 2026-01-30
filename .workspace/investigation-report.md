# May-la MCP Disconnection Investigation Report

## Executive Summary

**Root Cause:** Multiple simultaneous daemon instances are running concurrently due to a missing instance reuse/singleton pattern. Each time Claude Code invokes may-la, a new instance (with unique ID) is created, spawning a new daemon process with separate database files.

**Severity:** HIGH - Multiple daemons cause:
- Resource exhaustion (2x+ memory, CPU overhead)
- Database synchronization issues
- Unpredictable MCP disconnections
- WAL file conflicts when multiple instances write simultaneously

---

## Current State: 2 Concurrent Instances Running

### Instance 1 (OLDER - 23:37:13)
```
Process:     PID 26579 (mayla-daemon)
Parent:      PID 26569 (mayla launcher)
Instance ID: 26569-1769740633943133000-2df607687a47e449
Status:      RUNNING (2 minutes 4 seconds)
Database:    /Users/juliolena/.mayla/instances/26569-.../
  - index.db:     4.0K
  - index.db-wal: 216K (moderate activity)
```

### Instance 2 (NEWER - 23:37:29)
```
Process:     PID 27505 (mayla-daemon)  
Parent:      PID 27461 (mayla launcher)
Instance ID: 27461-1769740649544250000-49d8e4f3cdcd374b
Status:      RUNNING (1 minute 48 seconds)
Database:    /Users/juliolena/.mayla/instances/27461-.../
  - index.db:      200K
  - index.db-wal:  4.0M (HEAVY ACTIVITY - 10x larger)
```

---

## Root Cause Analysis

### 1. Instance Generation Is Non-Deterministic

**File:** `cmd/mayla/main.go:93`
```go
func generateInstanceID() string {
    return fmt.Sprintf("%d-%d-%x", 
        os.Getpid(),           // Launcher PID
        time.Now().UnixNano(), // Current timestamp
        rand.Int63())          // Random value
}
```

**Problem:** Each `mayla` invocation:
- Gets a new launcher process (new PID)
- Has a unique timestamp (nanosecond precision)
- Includes random bytes
- **Result:** Every invocation = NEW instance directory

### 2. No Instance Reuse/Singleton Pattern

**File:** `internal/config/config.go:98-109`
```go
func LoadConfigWithInstance(instanceID string) (*Config, error) {
    // ... creates new instance directory ...
    if err := os.MkdirAll(instanceDir, 0700); err != nil {
        return nil, fmt.Errorf("failed to create instance directory: %w", err)
    }
    // Unconditionally creates daemon socket and databases in instanceDir
}
```

**Problem:**
- No check for existing daemon
- No reuse of existing instance
- Each invocation = new socket + new database + new daemon

### 3. Lock File NOT Preventing Multiple Instances

**File:** `internal/daemon/lifecycle.go:35-37`
```go
func (lm *LifecycleManager) ValidateNoOtherInstance() error {
    return lm.AcquireInstanceLock()
}
```

**Problem:** Each instance has its OWN lock file in its own directory:
```
Instance 1: /Users/juliolena/.mayla/instances/26569-.../daemon.lock ✓
Instance 2: /Users/juliolena/.mayla/instances/27461-.../daemon.lock ✓ (DIFFERENT FILE!)
```

Lock files don't conflict because they're in different directories. There's NO global lock across instances.

### 4. What Triggers New Instance Creation?

The user mentioned: "saved only one memory with an instance open and it exploded"

**Hypothesis:** The `claude-memory` hook in `/Users/juliolena/.claude/settings.json` triggers:

```json
{
  "hooks": {
    "PreCompact": [
      {
        "matcher": "*",
        "hooks": [{
          "type": "command",
          "command": "claude-memory save --pre-compact"
        }]
      }
    ],
    "SessionEnd": [
      {
        "command": "claude-memory save --session-end"
      }
    ]
  }
}
```

Each call to `claude-memory` → likely invokes `mayla` → new launcher → new instance ID → new daemon.

---

## Evidence of the Problem

### Database File Growth Disparity
```
Instance 1 (older):
  index.db-wal: 216 KB (light activity over 2 min)

Instance 2 (newer):  
  index.db-wal: 4.0 MB (10x larger in 1.8 min!)
```

**Why?** Instance 2 is getting ALL the write operations because:
1. It was created last (newest mayla launcher)
2. Claude Code connects to its socket
3. Instance 1 is abandoned but still running
4. Memory writes go to Instance 2, not Instance 1

### Lock File Status
Both instances have locks, but locks don't prevent concurrent instances:
```bash
/Users/juliolena/.mayla/instances/26569-*/daemon.lock  ✓ HELD
/Users/juliolena/.mayla/instances/27461-*/daemon.lock  ✓ HELD (SEPARATE FILE!)
```

---

## Why Disconnections Happen

1. **Memory accumulation:** Instance 2 gets 4MB WAL file (still growing)
2. **Multiple databases indexing:** Both daemons rebuild indexes independently
3. **Socket conflicts:** Unclear which instance socket Claude Code connects to
4. **Resource exhaustion:** 2 daemons + 2 index rebuild processes = CPU/memory spike
5. **One daemon stalls:** Another takes over inconsistently → disconnections

---

## Architecture Diagram

```
Claude Code Session
    ↓
    └─ Hook: claude-memory save
         ↓
    ┌────────────────────────────────┐
    │ FIRST INVOCATION (23:37:13)     │
    │ mayla launcher (PID 26569)      │
    │   → instanceID: 26569-xxx-xxx   │
    │   → daemon (PID 26579)          │
    │   → socket: ~/.mayla/instances/26569-xxx/daemon.sock
    │   → db: ~/.mayla/instances/26569-xxx/index.db
    └────────────────────────────────┘
                ↓
         (daemon running, awaiting commands)
         
    ┌────────────────────────────────┐
    │ SECOND INVOCATION (23:37:29)    │
    │ mayla launcher (PID 27461)      │ ← NEW PID!
    │   → instanceID: 27461-yyy-yyy   │ ← NEW ID!
    │   → daemon (PID 27505)          │ ← NEW DAEMON!
    │   → socket: ~/.mayla/instances/27461-yyy/daemon.sock
    │   → db: ~/.mayla/instances/27461-yyy/index.db
    └────────────────────────────────┘
                ↓
    NOW TWO DAEMONS RUNNING!
    - Instance 1: Idle, no new commands
    - Instance 2: Active (gets all requests from Claude Code)
    
    RESULT: Resource waste + potential conflicts
```

---

## Impact Assessment

### Immediate Problems
1. ✅ **Confirmed:** 2 daemons running concurrently
2. ✅ **Confirmed:** Each has separate database file
3. ✅ **Confirmed:** WAL file in Instance 2 is 4MB (growing)
4. ✅ **Confirmed:** No global singleton pattern

### Why Disconnection Occurs
- Multiple processes writing to separate databases
- File watching triggers on both instances
- LSP servers potentially conflicting
- Memory pressure from multiple instances
- Socket becomes unresponsive when one daemon stalls

---

## Fix Required: Instance Singleton Pattern

### Option 1: Reuse Existing Instance (Recommended)
```
Before creating new launcher/daemon:
1. Check ~/.mayla/instances/ directory
2. If running daemon exists with healthy socket:
   - Reuse its socket
   - Connect to existing daemon
   - Skip launcher/daemon startup
3. If no daemon or socket unresponsive:
   - Create new instance
   - Cleanup old stale instances
```

### Option 2: Global Lock + Single Instance
```
Create global lock: ~/.mayla/daemon.lock (NOT per-instance)
- Only ONE mayla launcher can start daemon at a time
- All subsequent calls connect to existing daemon socket
- Instance ID becomes deterministic/static
```

### Option 3: Health Check + Auto-Restart
```
1. Always try to connect to ~/.mayla/primary-instance/daemon.sock
2. If unresponsive:
   - Kill stale daemon
   - Cleanup old instances
   - Start fresh single daemon
3. Reuse same instance directory every time
```

---

## Recommended Actions

### Immediate (High Priority)
1. Kill all but one mayla-daemon process
2. Remove old instance directories to free disk
3. Configure settings.json hooks to avoid multiple invocations

### Short-term (Medium Priority)  
1. Implement instance reuse/singleton pattern
2. Add health checks to detect stale daemons
3. Implement automatic cleanup of abandoned instances
4. Make instance ID deterministic

### Long-term (Design)
1. Separate "launcher" and "daemon" concerns
2. Single long-lived daemon process per workspace
3. Claude Code should detect and reuse existing daemon
4. Instance management system for cleanup policies

---

## Quick Fix for User

```bash
# Stop all mayla processes
pkill -f mayla-daemon
pkill -f "mayla " 

# Remove old instance directories (keep only newest if you want)
rm -rf ~/.mayla/instances/

# Restart Claude Code - will create fresh single instance
```

Then update settings.json hooks to be less aggressive about calling claude-memory.

---

## Files to Review for Implementation

1. **`cmd/mayla/main.go`** - Launcher logic (instance generation)
2. **`internal/config/config.go`** - LoadConfigWithInstance() 
3. **`internal/daemon/lifecycle.go`** - Lock management
4. **`cmd/mayla-daemon/main.go`** - Daemon startup
5. **Recent commits:** Look for instance management changes
