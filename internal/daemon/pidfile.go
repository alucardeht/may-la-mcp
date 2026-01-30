package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type PIDFile struct {
	path string
}

func NewPIDFile(path string) *PIDFile {
	return &PIDFile{path: path}
}

func (p *PIDFile) Write() error {
	pid := os.Getpid()

	f, err := os.OpenFile(p.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			info, lerr := os.Lstat(p.path)
			if lerr == nil && info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("PID file is a symlink (security risk)")
			}
			os.Remove(p.path)
			f, err = os.OpenFile(p.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("failed to create PID file: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create PID file: %w", err)
		}
	}
	defer f.Close()

	_, err = f.WriteString(strconv.Itoa(pid))
	return err
}

func (p *PIDFile) Read() (int, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return 0, nil
	}

	pid, err := strconv.Atoi(content)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID: %d (must be positive)", pid)
	}

	return pid, nil
}

func (p *PIDFile) IsProcessAlive() bool {
	pid, err := p.Read()
	if err != nil || pid == 0 {
		return false
	}

	return processExists(pid)
}

func (p *PIDFile) Remove() error {
	if info, err := os.Lstat(p.path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to remove PID file: is a symlink")
		}
	}
	return os.Remove(p.path)
}

func (p *PIDFile) Path() string {
	return p.path
}
