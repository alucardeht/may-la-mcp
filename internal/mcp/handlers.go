package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
	"github.com/alucardeht/may-la-mcp/pkg/version"
)

type Handler struct {
	registry *tools.Registry
}

func NewHandler(registry *tools.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) Handle(req *Request) *Response {
	resp := &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = h.handleInitialize(req)
	case "notifications/initialized":
		return resp
	case "ping":
		resp.Result = map[string]interface{}{}
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

func (h *Handler) handleInitialize(req *Request) interface{} {
	clientVersion := ""
	if params, ok := req.Params["protocolVersion"]; ok {
		if v, ok := params.(string); ok {
			clientVersion = v
		}
	}

	negotiated := version.ProtocolVersion
	for _, v := range version.SupportedProtocolVersions {
		if clientVersion == v {
			negotiated = v
			break
		}
	}

	return map[string]interface{}{
		"protocolVersion": negotiated,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "May-la MCP Server",
			"version": version.Version,
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

		toolData := map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"inputSchema": schema,
		}

		if annotations := t.Annotations(); annotations != nil {
			toolData["annotations"] = annotations
		}

		toolsData[i] = toolData
	}

	return map[string]interface{}{
		"tools": toolsData,
	}
}

func (h *Handler) handleCallTool(req *Request) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool panic: %v", r)
			slog.Error("tool panic", "panic", r, "stack", string(debug.Stack()))
		}
	}()

	paramsData, err := json.Marshal(req.Params)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	var callReq struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(paramsData, &callReq); err != nil {
		return nil, fmt.Errorf("invalid tool call: %w", err)
	}

	if callReq.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	value, err := h.registry.Execute(ctx, callReq.Name, callReq.Arguments)
	if err != nil {
		return nil, err
	}

	text, ok := value.(string)
	if !ok {
		jsonBytes, _ := json.Marshal(value)
		text = string(jsonBytes)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text,
			},
		},
	}, nil
}
