package mcp

import (
	"bufio"
	"encoding/json"
	"io"

	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

type Server struct {
	registry *tools.Registry
	handler  *Handler
}

func NewServer(registry *tools.Registry) *Server {
	return &Server{
		registry: registry,
		handler:  NewHandler(registry),
	}
}

func (s *Server) HandleRequest(req *Request) *Response {
	return s.handler.Handle(req)
}

func (s *Server) ProcessStream(reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	encoder := json.NewEncoder(writer)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := &Response{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &protocol.JSONRPCError{
					Code:    -32700,
					Message: "Parse error",
				},
			}
			encoder.Encode(resp)
			continue
		}

		resp := s.HandleRequest(&req)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (s *Server) Registry() *tools.Registry {
	return s.registry
}
