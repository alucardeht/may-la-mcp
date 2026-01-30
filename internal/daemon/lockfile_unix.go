//go:build unix

package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// platformLock acquires an exclusive non-blocking lock on the file using flock
func (l *LockFile) platformLock(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			return ErrLockHeld
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

// platformUnlock releases the lock on the file
func (l *LockFile) platformUnlock(f *os.File) {
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
