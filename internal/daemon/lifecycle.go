package daemon

import (
	"fmt"
	"net"
	"path/filepath"
	"time"
)

type LifecycleManager struct {
	lockFile   *LockFile
	pidFile    *PIDFile
	socketPath string
}

func NewLifecycleManager(baseDir, socketPath string) *LifecycleManager {
	return &LifecycleManager{
		lockFile:   NewLockFile(filepath.Join(baseDir, "daemon.lock")),
		pidFile:    NewPIDFile(filepath.Join(baseDir, "daemon.pid")),
		socketPath: socketPath,
	}
}

func (lm *LifecycleManager) AcquireStartupLock() error {
	return lm.lockFile.Acquire()
}

func (lm *LifecycleManager) AcquireInstanceLock() error {
	if err := lm.lockFile.Acquire(); err != nil {
		return fmt.Errorf("failed to acquire instance lock: %w", err)
	}
	return nil
}

func (lm *LifecycleManager) ValidateNoOtherInstance() error {
	return lm.AcquireInstanceLock()
}

func (lm *LifecycleManager) isSocketResponsive() bool {
	conn, err := net.DialTimeout("unix", lm.socketPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (lm *LifecycleManager) RegisterRunningDaemon() error {
	return lm.pidFile.Write()
}

func (lm *LifecycleManager) Cleanup() {
	lm.pidFile.Remove()
	lm.lockFile.Release()
}

func (lm *LifecycleManager) LockFile() *LockFile {
	return lm.lockFile
}

func (lm *LifecycleManager) PIDFile() *PIDFile {
	return lm.pidFile
}
