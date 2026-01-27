package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

type Client struct {
	conn    net.Conn
	writer  *bufio.Writer
	encoder *json.Encoder
	decoder *json.Decoder
	mu      sync.Mutex
}

func NewClient(conn net.Conn) *Client {
	writer := bufio.NewWriter(conn)
	return &Client{
		conn:    conn,
		writer:  writer,
		encoder: json.NewEncoder(writer),
		decoder: json.NewDecoder(conn),
	}
}

func (c *Client) SendRequest(req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.encoder.Encode(req); err != nil {
		return nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, err
	}

	var resp protocol.JSONRPCResponse
	if err := c.decoder.Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) Call(method string, params map[string]interface{}) (interface{}, error) {
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", resp.Error.Message)
	}

	return resp.Result, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
