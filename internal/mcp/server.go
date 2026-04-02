package mcp

import (
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

func (s *Server) ProcessStream(reader io.Reader, writer io.Writer) error {
	decoder := json.NewDecoder(reader)
	fw := protocol.NewFlushWriter(writer)
	encoder := json.NewEncoder(fw)

	for {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			resp := &Response{
				JSONRPC: "2.0",
				Error: &protocol.JSONRPCError{
					Code:    -32700,
					Message: "Parse error",
				},
			}
			encoder.Encode(resp)
			fw.Flush()
			continue
		}

		if req.ID == nil && req.Method != "initialize" {
			s.handler.Handle(&req)
			continue
		}

		resp := s.handler.Handle(&req)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
		fw.Flush()
	}
}
