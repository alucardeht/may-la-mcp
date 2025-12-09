package mcp

import "github.com/maylamcp/mayla/pkg/protocol"

type Request = protocol.JSONRPCRequest
type Response = protocol.JSONRPCResponse
type Tool = protocol.Tool
type ToolCall = protocol.ToolCall

type InitializeRequest struct {
	ClientInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type InitializeResponse struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    interface{}  `json:"capabilities"`
	ServerInfo      interface{}  `json:"serverInfo"`
}

type ListToolsRequest struct {
}

type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type CallToolResponse struct {
	Content []interface{} `json:"content"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
