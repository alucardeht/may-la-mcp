package daemon

import (
	"fmt"
	"os"
	"syscall"
)

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

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			return fmt.Errorf("daemon already running (lock held)")
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	l.file = f
	return nil
}

func (l *LockFile) Release() error {
	if l.file == nil {
		return nil
	}

	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)

	err := l.file.Close()
	l.file = nil

	os.Remove(l.path)

	return err
}

func (l *LockFile) Abandon() error {
	if l.file == nil {
		return nil
	}

	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)

	err := l.file.Close()
	l.file = nil

	return err
}

func (l *LockFile) IsLocked() bool {
	return l.file != nil
}
