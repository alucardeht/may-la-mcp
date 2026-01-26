# May-la MCP

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)
[![MCP Protocol](https://img.shields.io/badge/MCP-2025--11--25-3178c6?style=flat-square)](https://spec.modelcontextprotocol.io)
[![Performance](https://img.shields.io/badge/Cold%20Start-%3C%2050ms-brightgreen?style=flat-square)](docs/performance.md)

**A high-performance MCP (Model Context Protocol) server written in Go**, engineered as a faster, more efficient alternative to SERENA MCP. Deliver powerful code navigation, file manipulation, and semantic search capabilities with lightning-fast response times.

## 🚀 Why May-la?

| Feature | May-la | SERENA |
|---------|--------|--------|
| **Cold Start** | < 50ms | 2-5s |
| **Tool Overhead** | < 10ms | ~200ms |
| **Memory Footprint** | < 50MB | 200-500MB |
| **Language** | Go (compiled) | Python (interpreted) |
| **Startup Latency** | Negligible | Multi-second |

May-la is purpose-built for Claude-Claude operations where response time directly impacts user experience. Every millisecond counts.

## 📋 Features

### 20 Production-Ready Tools Across 5 Categories

#### 📁 File Operations (7 tools)
- **`read`** — Read files with intelligent chunking and progress tracking
- **`write`** — Write files with atomic operations and safety checks
- **`edit`** — Edit files using search/replace with regex support
- **`create`** — Create new files with directory structure validation
- **`delete`** — Remove files and directories safely
- **`move`** — Move and rename files
- **`list`** — List directory contents with filtering and sorting

#### 🔍 Search & Navigation (4 tools)
- **`search`** — Full-text search powered by ripgrep with context
- **`find`** — Find files by pattern (glob/regex)
- **`symbols`** — Extract code symbols with semantic intelligence (LSP → Index → Regex fallback)
- **`references`** — Find symbol references across codebase with LSP support

#### 💾 Memory System (6 tools)
- **`memory_write`** — Save long-term memory with auto-versioning
- **`memory_read`** — Retrieve memories by name and version
- **`memory_list`** — List all stored memories with metadata
- **`memory_search`** — Semantic search over memories using FTS5
- **`memory_delete`** — Remove memories with safety checks
- **`memory_update`** — Update existing memory content, category, or tags with partial updates and append mode

#### 📄 Documentation (2 tools)
- **`doc_write`** — Write project documentation files with automatic directory creation
- **`doc_read`** — Read project documentation files

#### 🏥 System (1 tool)
- **`health`** — Check daemon status and version

### 🏷️ Tool Annotations

All tools include MCP annotations for smarter client integration:

| Annotation | Description |
|------------|-------------|
| `readOnlyHint` | Tool only reads data, no side effects |
| `destructiveHint` | Tool can delete or permanently modify data |
| `idempotentHint` | Tool can be safely retried with same result |
| `openWorldHint` | Tool may return evolving/dynamic results |

## 🧠 Semantic Code Intelligence

May-la provides intelligent code understanding through a 3-tier semantic analysis system:

### Architecture

```
Query → Index (SQLite FTS5) → LSP Server → Regex Fallback
              ↓                    ↓              ↓
         Cached symbols      Language        Pattern-based
         (sub-ms lookup)     Analysis        extraction
```

### Supported Language Servers

| Language | LSP Server | Status | Extensions |
|----------|-----------|--------|------------|
| Go | gopls | ✅ Enabled | `.go` |
| TypeScript | typescript-language-server | ✅ Enabled | `.ts`, `.tsx` |
| JavaScript | typescript-language-server | ✅ Enabled | `.js`, `.jsx`, `.mjs` |
| Python | pylsp | ✅ Enabled | `.py` |
| Rust | rust-analyzer | ✅ Enabled | `.rs` |
| C/C++ | clangd | ✅ Enabled | `.c`, `.cpp`, `.h` |
| Java | jdtls | ⚠️ Disabled | `.java` |

### SQLite FTS5 Index

- Automatic symbol indexing with full-text search
- Sub-millisecond lookups for cached results
- Background incremental updates via file watching

### Encoding Support (30+)

Automatic encoding detection and normalization:
- **Unicode:** UTF-8, UTF-16 LE/BE (with BOM support)
- **Asian:** Shift-JIS, EUC-JP, GBK, GB18030, Big5, EUC-KR
- **Latin:** ISO-8859-1 through 16, Windows-1250 through 1258
- **Cyrillic:** KOI8-R, KOI8-U

## 🛠 Installation

May-la works with any MCP-compatible IDE. Choose your IDE below:

### For Claude Code

**One-line installation:**

**macOS / Linux:**
```bash
claude mcp add may-la -s user -- bash -c 'SCRIPT=$(mktemp); curl -sL https://raw.githubusercontent.com/alucardeht/may-la-mcp/main/scripts/mayla-launcher.sh > "$SCRIPT"; bash "$SCRIPT"; rm "$SCRIPT"'
```

**Windows (PowerShell):**
```powershell
claude mcp add may-la -s user -- powershell -ExecutionPolicy Bypass -Command "$script = [System.IO.Path]::GetTempFileName(); irm https://raw.githubusercontent.com/alucardeht/may-la-mcp/main/scripts/mayla-launcher.ps1 -OutFile $script; & $script; Remove-Item $script"
```

After installation:
1. Restart Claude Code: `/quit` then restart
2. Verify installation (see Validation section below)

### For Cursor

**Step 1: Install binaries**

Run the same installation command as Claude Code above (it downloads the binaries to `~/.mayla/`).

**Step 2: Configure Cursor**

Add to your Cursor settings (`~/.cursor/mcp.json` or via Settings → MCP):

> **Important**: The tilde (`~`) does not expand in JSON. Use the absolute path to your home directory instead.

**macOS:**
```json
{
  "mcpServers": {
    "may-la": {
      "command": "/Users/YOUR_USERNAME/.mayla/mayla",
      "args": []
    }
  }
}
```

**Linux:**
```json
{
  "mcpServers": {
    "may-la": {
      "command": "/home/YOUR_USERNAME/.mayla/mayla",
      "args": []
    }
  }
}
```

**Windows:**
```json
{
  "mcpServers": {
    "may-la": {
      "command": "C:\\Users\\YOUR_USERNAME\\.mayla\\mayla.exe",
      "args": []
    }
  }
}
```

Replace `YOUR_USERNAME` with your actual username.

**Step 3: Restart Cursor**

Restart Cursor to load the MCP server.

### For Gemini CLI

**One-liner installation:**

**macOS / Linux:**
```bash
gemini mcp add may-la -s user -- bash -c 'SCRIPT=$(mktemp); curl -sL https://raw.githubusercontent.com/alucardeht/may-la-mcp/main/scripts/mayla-launcher.sh > "$SCRIPT"; bash "$SCRIPT"; rm "$SCRIPT"'
```

**Verify installation:**
```bash
gemini mcp list
```

You should see `may-la` in the list of configured MCP servers.

> **Note**: The launcher script automatically downloads the binaries to `~/.mayla/` if they don't exist, and keeps them updated.

### What Happens During Installation

1. Launcher script downloads from GitHub
2. Detects your platform (OS + architecture)
3. Downloads **both** pre-compiled binaries for your platform:
   - `mayla` (CLI client) - ~6-7MB
   - `mayla-daemon` (background server) - ~6-8MB
4. Stores in `~/.mayla/` directory (or `%USERPROFILE%\.mayla\` on Windows)
5. For macOS: Removes quarantine attributes to prevent Gatekeeper blocks
6. Auto-updates when new versions are released

### Supported Platforms

| OS | Architecture | Status | Binary Size | Notes |
|----|--------------|--------|-------------|-------|
| **macOS** | Apple Silicon (arm64) | ✅ Full CGO | ~6-7 MB | SQLite FTS5 enabled |
| **macOS** | Intel (amd64) | ✅ Full CGO | ~6-7 MB | SQLite FTS5 enabled |
| **Linux** | amd64 | ✅ Full CGO | ~6-7 MB | SQLite FTS5 enabled |
| **Windows** | amd64 | ✅ Full CGO | ~6-7 MB | SQLite FTS5 enabled |

> **Note**: ARM64 builds for Linux/Windows are not provided due to CGO cross-compilation complexity. Native compilation on those platforms would require specialized toolchains.

### Build from Source (Optional)

If you prefer to build yourself:

```bash
git clone https://github.com/alucardeht/may-la-mcp.git
cd may-la-mcp
make build-all
```

Requirements: Go 1.22+

## ✅ Validation

**How to verify May-la is working correctly:**

### Quick Test

In Claude Code or Cursor, try any of these MCP tools:

```
Use may-la to list files in current directory
```

or

```
Use may-la to read README.md
```

### Detailed Verification

**Step 1: Check binaries are installed**
```bash
ls -lh ~/.mayla/
```

Expected output:
```
-rwxr-xr-x  mayla          (~6-7 MB)
-rwxr-xr-x  mayla-daemon   (~6-8 MB)
```

**Step 2: Test MCP connection**

In Claude Code:
```
/mcp list
```

You should see `may-la` in the list of available servers.

### Common Issues

**"Failed to connect to daemon"**
- Daemon may not have started properly
- Check: `ps aux | grep mayla-daemon`
- Solution: Restart your IDE

**"Command not found: mayla"**
- Binaries not in expected location
- Check: `ls -la ~/.mayla/`
- Solution: Re-run installation command

**macOS: "Cannot be opened because the developer cannot be verified"**
- Quarantine attributes not removed properly
- Solution: `xattr -d com.apple.quarantine ~/.mayla/mayla ~/.mayla/mayla-daemon`

**Windows: "Access Denied"**
- Antivirus blocking execution
- Solution: Add `%USERPROFILE%\.mayla\` to antivirus exclusions

## 📖 Quick Start

### 1. Start the Daemon

```bash
mayla-daemon --socket /tmp/mayla.sock &
```

The daemon listens on a Unix socket for JSON-RPC 2.0 requests.

### 2. Make Tool Calls

All communication happens via standard MCP protocol. Claude handles this automatically once registered.

```
Tool: read
Input:
  path: "/path/to/file.go"
  max_lines: 100
```

### 3. Example: File Navigation

```
Tool: list
Input:
  path: "."
  pattern: "*.go"
  recursive: true
```

```
Tool: search
Input:
  query: "func handleRequest"
  path: "."
```

```
Tool: symbols
Input:
  path: "internal/tools/files/reader.go"
  language: "go"
```

## 🏗 Project Structure

```
may-la-mcp/
├── cmd/
│   ├── mayla/                 # MCP stdio adapter
│   └── mayla-daemon/          # Background daemon (Unix socket)
├── internal/
│   ├── config/                # Configuration management
│   ├── daemon/                # Socket server, JSON-RPC handling
│   ├── index/                 # SQLite FTS5 symbol indexing
│   │   ├── store.go           # Database operations
│   │   ├── worker.go          # Background indexer
│   │   └── encoder.go         # Encoding detection (30+ encodings)
│   ├── logger/                # Structured logging (slog)
│   ├── lsp/                   # Language Server Protocol
│   │   ├── manager.go         # LSP lifecycle management
│   │   ├── client.go          # JSON-RPC client
│   │   └── config.go          # LSP configurations
│   ├── mcp/                   # MCP protocol handler
│   ├── router/                # Query routing (Index → LSP → Regex)
│   ├── tools/
│   │   ├── files/             # File operations
│   │   ├── search/            # Search & navigation
│   │   └── memory/            # Memory system
│   ├── types/                 # Shared type definitions
│   └── watcher/               # File system watcher (fsnotify)
├── tests/                     # E2E tests
├── Makefile                   # Build automation
└── README.md
```

## 🔧 Development

### Build Commands

```bash
# Build everything
make build

# Build daemon only
make daemon

# Build CLI tool
make cli

# Run unit tests
make test

# Run E2E tests
make test-e2e

# Clean build artifacts
make clean

# Install locally
make install
```

### Configuration

Create `~/.mayla/config.yaml`:

```yaml
socket: /tmp/mayla.sock
memory_dir: ~/.mayla/memories
log_level: info
max_chunk_size: 10000
search:
  exclude_patterns:
    - "*.git"
    - "node_modules"
    - "__pycache__"
```

## 📊 Performance Characteristics

### Benchmarks

| Operation | Latency | Memory |
|-----------|---------|--------|
| Cold Start | < 50ms | - |
| `read` (1MB file) | ~15ms | +2MB |
| `search` (1000 files) | ~80ms | +5MB |
| `symbols` (large file) | ~25ms | +3MB |
| `memory_search` (1000 items) | ~40ms | +8MB |

### Resource Limits

- **Memory**: < 50MB base, < 100MB under load
- **File Size**: No hard limit, streaming with intelligent chunking
- **Search Scope**: Respects `.gitignore` and exclusion patterns

## 🔌 Protocol Details

May-la implements the [Model Context Protocol](https://spec.modelcontextprotocol.io) (2025-11-25) with:
- **JSON-RPC 2.0 messaging** over Unix sockets
- **JSON-RPC 2.0 notifications** — One-way messages (no response required)
- **Tool annotations** — Semantic hints for client optimization

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tool/read",
  "params": {
    "path": "/path/to/file.go",
    "max_lines": 100
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "contents": "...",
    "line_count": 45,
    "total_lines": 150
  }
}
```

### Error Handling

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32603,
    "message": "File not found",
    "data": {
      "path": "/nonexistent/file.go"
    }
  }
}
```

## 🎯 Use Cases

### 1. Code Navigation for Claude
Provide Claude with fast, comprehensive code understanding:
```
1. Find relevant files with `find`
2. Extract symbols with `symbols`
3. Search implementation with `search`
4. Read context with `read`
```

### 2. Persistent Memory
Maintain long-term context across conversations:
```
1. Save insights with `memory_write`
2. Retrieve context with `memory_read`
3. Search semantically with `memory_search`
4. Manage with `memory_list`
```

## 📝 License

Apache License 2.0 — See [LICENSE](LICENSE) for details.

## 🤝 Contributing

Contributions welcome! Please:
1. Check existing issues and PRs
2. Test thoroughly with `make test-e2e`
3. Keep performance targets in mind (< 50ms cold start)
4. Document new tools in this README

## 📚 Additional Resources

- [MCP Specification](https://spec.modelcontextprotocol.io) — Protocol documentation
- [Go 1.22 Release Notes](https://golang.org/doc/go1.22) — Language features
- [ripgrep Documentation](https://github.com/BurntSushi/ripgrep) — Search engine
- [Tree-sitter](https://tree-sitter.github.io) — Symbol extraction

## 🐛 Troubleshooting

### Installation Issues

**Binaries not downloading**
- Check internet connection
- Verify GitHub is accessible: `curl -I https://github.com`
- Check firewall/proxy settings

**Permission denied on Linux/macOS**
- Ensure binaries are executable: `chmod +x ~/.mayla/*`
- Check directory permissions: `ls -ld ~/.mayla`

### Runtime Issues

**"Socket not found" error**
- Daemon not running: Start manually `~/.mayla/mayla-daemon &`
- Socket path issue: Check `~/.mayla/config.yaml` for correct socket path
- On restart, old socket may exist: `rm ~/.mayla/daemon.sock`

**High memory usage**
- Normal: May-la uses <50MB base, <100MB under load
- If >200MB: Check for memory leaks, report issue with `ps aux | grep mayla`

**SQLite/FTS5 errors**
- Verify CGO is enabled: `strings ~/.mayla/mayla | grep sqlite`
- Should show SQLite symbols if CGO compiled correctly
- If not: Re-download with installation command (may be old version)

### Platform-Specific

**macOS: Quarantine issues**
```bash
# Remove quarantine from both binaries
xattr -d com.apple.quarantine ~/.mayla/mayla
xattr -d com.apple.quarantine ~/.mayla/mayla-daemon
```

**Linux: Missing dependencies**
- CGO requires standard C libraries
- Install: `sudo apt-get install libc6-dev` (Debian/Ubuntu)
- Install: `sudo yum install glibc-devel` (RHEL/CentOS)

**Windows: Antivirus false positives**
- Add exclusion: `%USERPROFILE%\.mayla\`
- Binaries are signed and safe (check GitHub Actions build logs)

### Getting Help

If problems persist:
1. Check [existing issues](https://github.com/alucardeht/may-la-mcp/issues)
2. Include in your report:
   - OS and architecture (`uname -a` on macOS/Linux, `systeminfo` on Windows)
   - Error messages from IDE console
3. Open new issue with details

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/alucardeht/may-la-mcp/issues)
- **Discussions**: [GitHub Discussions](https://github.com/alucardeht/may-la-mcp/discussions)
- **Documentation**: [docs/](docs/) directory

---

**Built with ⚡ for speed, 🎯 for precision, 💾 for persistence.**
