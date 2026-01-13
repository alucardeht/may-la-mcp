package mcp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

type Handler struct {
	registry  *tools.Registry
	startTime time.Time
}

func NewHandler(registry *tools.Registry) *Handler {
	return &Handler{
		registry:  registry,
		startTime: time.Now(),
	}
}

func (h *Handler) Handle(req *Request) *Response {
	if req.ID == nil {
		return nil
	}

	resp := &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = h.handleInitialize()
	case "tools/list":
		resp.Result = h.handleListTools()
	case "tools/call":
		result, err := h.handleCallTool(req)
		if err != nil {
			resp.Error = &protocol.JSONRPCError{
				Code:    -32603,
				Message: err.Error(),
			}
		} else {
			resp.Result = result
		}
	default:
		resp.Error = &protocol.JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	return resp
}

func (h *Handler) handleInitialize() interface{} {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "May-la MCP Server",
			"version": "0.1.0",
		},
	}
}

func (h *Handler) handleListTools() interface{} {
	toolsList := h.registry.List()
	toolsData := make([]map[string]interface{}, len(toolsList))

	for i, t := range toolsList {
		var schema interface{}
		if err := json.Unmarshal(t.Schema(), &schema); err != nil {
			schema = json.RawMessage(t.Schema())
		}

		toolsData[i] = map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"inputSchema": schema,
		}
	}

	return map[string]interface{}{
		"tools": toolsData,
	}
}

func (h *Handler) handleCallTool(req *Request) (interface{}, error) {
	callReq := struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}{}

	paramsData, err := json.Marshal(req.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	if err := json.Unmarshal(paramsData, &callReq); err != nil {
		return nil, fmt.Errorf("failed to parse tool call request: %w", err)
	}

	if callReq.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	result, err := h.registry.Execute(callReq.Name, callReq.Arguments)
	if err != nil {
		return nil, err
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	}, nil
}
