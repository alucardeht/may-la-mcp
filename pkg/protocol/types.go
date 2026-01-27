package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
)

type JSONRPCRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      interface{}       `json:"id,omitempty"`
	Method  string            `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type BatchRequest []JSONRPCRequest

type BatchResponse []JSONRPCResponse

type RequestOrBatch struct {
	Single *JSONRPCRequest
	Batch  BatchRequest
}

func (r *RequestOrBatch) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '[' {
		return json.Unmarshal(data, &r.Batch)
	}
	r.Single = new(JSONRPCRequest)
	return json.Unmarshal(data, r.Single)
}

func (r *RequestOrBatch) IsBatch() bool {
	return len(r.Batch) > 0
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ToolCall struct {
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type HealthResponse struct {
	Status string `json:"status"`
	Uptime int64  `json:"uptime"`
}

type Flusher interface {
	Flush() error
}

type FlushWriter struct {
	w         io.Writer
	bufWriter *bufio.Writer
}

func NewFlushWriter(w io.Writer) *FlushWriter {
	return &FlushWriter{
		w:         w,
		bufWriter: bufio.NewWriter(w),
	}
}

func (fw *FlushWriter) Write(p []byte) (n int, err error) {
	return fw.bufWriter.Write(p)
}

func (fw *FlushWriter) Flush() error {
	return fw.bufWriter.Flush()
}
