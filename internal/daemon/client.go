package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

type Client struct {
	conn net.Conn
}

func NewClient(conn net.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

func (c *Client) SendRequest(req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	data = append(data, '\n')

	if _, err := c.conn.Write(data); err != nil {
		return nil, err
	}

	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	decoder := json.NewDecoder(c.conn)
	var resp protocol.JSONRPCResponse
	if err := decoder.Decode(&resp); err != nil {
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
