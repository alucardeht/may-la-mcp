package daemon

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

type SocketListener struct {
	path     string
	listener net.Listener
}

func NewSocketListener(socketPath string) *SocketListener {
	return &SocketListener{
		path: socketPath,
	}
}

func (sl *SocketListener) Start() error {
	dir := filepath.Dir(sl.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if err := os.Remove(sl.path); err != nil && !os.IsNotExist(err) {
		return err
	}

	listener, err := net.Listen("unix", sl.path)
	if err != nil {
		return err
	}

	sl.listener = listener
	return os.Chmod(sl.path, 0700)
}

func (sl *SocketListener) Accept() (net.Conn, error) {
	if sl.listener == nil {
		return nil, fmt.Errorf("listener not started")
	}
	return sl.listener.Accept()
}

func (sl *SocketListener) Close() error {
	if sl.listener == nil {
		return nil
	}
	return sl.listener.Close()
}

type SocketConnector struct {
	path string
}

func NewSocketConnector(socketPath string) *SocketConnector {
	return &SocketConnector{
		path: socketPath,
	}
}

func (sc *SocketConnector) Connect() (net.Conn, error) {
	return net.Dial("unix", sc.path)
}
