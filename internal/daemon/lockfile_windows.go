//go:build windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	// LOCKFILE_EXCLUSIVE_LOCK requests an exclusive lock
	lockfileExclusiveLock = 0x00000002
	// LOCKFILE_FAIL_IMMEDIATELY returns immediately if lock cannot be acquired
	lockfileFailImmediately = 0x00000001
)

// platformLock acquires an exclusive non-blocking lock on the file using LockFileEx
func (l *LockFile) platformLock(f *os.File) error {
	var ol syscall.Overlapped
	handle := syscall.Handle(f.Fd())

	r1, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1, 0,
		uintptr(unsafe.Pointer(&ol)),
	)
	if r1 == 0 {
		// Error code 33 (ERROR_LOCK_VIOLATION) means lock is held by another process
		if err == syscall.Errno(33) {
			return ErrLockHeld
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

// platformUnlock releases the lock on the file using UnlockFileEx
func (l *LockFile) platformUnlock(f *os.File) {
	var ol syscall.Overlapped
	handle := syscall.Handle(f.Fd())

	procUnlockFileEx.Call(
		uintptr(handle),
		0,
		1, 0,
		uintptr(unsafe.Pointer(&ol)),
	)
}
