package mcp

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/logger"
	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
	"github.com/alucardeht/may-la-mcp/pkg/version"
)

var log = logger.ForComponent("mcp")

type Handler struct {
	registry  *tools.Registry
	startTime time.Time
	initialized bool
	clientInfo ClientInfo
}

type ClientInfo struct {
	Name    string
	Version string
}

func NewHandler(registry *tools.Registry) *Handler {
	return &Handler{
		registry:    registry,
		startTime:   time.Now(),
		initialized: false,
		clientInfo:  ClientInfo{},
	}
}

func (h *Handler) Handle(req *Request) *Response {
	resp := &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		result, err := h.handleInitialize(req)
		if err != nil {
			resp.Error = &protocol.JSONRPCError{
				Code:    -32603,
				Message: err.Error(),
			}
		} else {
			resp.Result = result
		}
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
	case "notifications/initialized":
		h.handleInitializedNotification(req)
		resp.Result = map[string]interface{}{}
	default:
		resp.Error = &protocol.JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	return resp
}

func (h *Handler) handleInitialize(req *Request) (interface{}, error) {
	initReq := struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}{}

	paramsData, err := json.Marshal(req.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	if err := json.Unmarshal(paramsData, &initReq); err != nil {
		return nil, fmt.Errorf("failed to parse initialize request: %w", err)
	}

	h.clientInfo.Name = initReq.ClientInfo.Name
	h.clientInfo.Version = initReq.ClientInfo.Version

	negotiatedVersion := negotiateProtocolVersion(initReq.ProtocolVersion)

	return map[string]interface{}{
		"protocolVersion": negotiatedVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "May-la MCP Server",
			"version": version.Version,
		},
	}, nil
}

func negotiateProtocolVersion(clientVersion string) string {
	for _, v := range version.SupportedProtocolVersions {
		if clientVersion == v {
			return v
		}
	}

	return version.ProtocolVersion
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

		if annotated, ok := t.(tools.AnnotatedTool); ok {
			if title := annotated.Title(); title != "" {
				toolData["title"] = title
			}
			if annotations := annotated.Annotations(); annotations != nil {
				toolData["annotations"] = annotations
			}
		}

		toolsData[i] = toolData
	}

	return map[string]interface{}{
		"tools": toolsData,
	}
}

func (h *Handler) handleInitializedNotification(req *Request) {
	h.initialized = true
}

func (h *Handler) handleCallTool(req *Request) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool execution panicked: %v", r)
			log.Error("tool panic recovered",
				"panic", r,
				"stack", string(debug.Stack()))
		}
	}()

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

	result, err = h.registry.ExecuteWithTimeout(callReq.Name, callReq.Arguments, 4*time.Minute)
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
