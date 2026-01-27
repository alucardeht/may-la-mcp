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

func (s *Server) HandleRequest(req *Request) *Response {
	return s.handler.Handle(req)
}

func (s *Server) HandleBatch(batch []Request) []*Response {
	responses := make([]*Response, 0, len(batch))
	for _, req := range batch {
		resp := s.HandleRequest(&req)
		if req.ID != nil {
			responses = append(responses, resp)
		}
	}
	return responses
}

func (s *Server) ProcessStream(reader io.Reader, writer io.Writer) error {
	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(writer)

	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				return nil
			}
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

		if len(raw) == 0 {
			continue
		}

		if raw[0] == '[' {
			var batch []Request
			if err := json.Unmarshal(raw, &batch); err != nil {
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

			responses := s.HandleBatch(batch)
			if err := encoder.Encode(responses); err != nil {
				return err
			}
		} else {
			var req Request
			if err := json.Unmarshal(raw, &req); err != nil {
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
	}
}

func (s *Server) Registry() *tools.Registry {
	return s.registry
}
