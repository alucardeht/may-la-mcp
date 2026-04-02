package protocol

import (
	"bufio"
	"io"
)

type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id,omitempty"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type FlushWriter struct {
	w   io.Writer
	buf *bufio.Writer
}

func NewFlushWriter(w io.Writer) *FlushWriter {
	return &FlushWriter{w: w, buf: bufio.NewWriter(w)}
}

func (fw *FlushWriter) Write(p []byte) (int, error) {
	return fw.buf.Write(p)
}

func (fw *FlushWriter) Flush() error {
	return fw.buf.Flush()
}
