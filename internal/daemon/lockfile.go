package daemon

import (
	"fmt"
	"os"
)

// ErrLockHeld is returned when attempting to acquire a lock that is already held
var ErrLockHeld = fmt.Errorf("daemon already running (lock held)")

type LockFile struct {
	path string
	file *os.File
}

func NewLockFile(path string) *LockFile {
	return &LockFile{path: path}
}

func (l *LockFile) Acquire() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}

	err = l.platformLock(f)
	if err != nil {
		f.Close()
		return err
	}

	l.file = f
	return nil
}

func (l *LockFile) Release() error {
	if l.file == nil {
		return nil
	}

	l.platformUnlock(l.file)

	err := l.file.Close()
	l.file = nil

	os.Remove(l.path)

	return err
}

func (l *LockFile) Abandon() error {
	if l.file == nil {
		return nil
	}

	l.platformUnlock(l.file)

	err := l.file.Close()
	l.file = nil

	return err
}

func (l *LockFile) IsLocked() bool {
	return l.file != nil
}
