package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andre-amorim/agy-notebooks-mcp/internal/client"
)

// Server coordinates the Model Context Protocol stdio transport.
type Server struct {
	client *client.Client
	in     io.Reader
	out    io.Writer
}

// NewServer creates a new MCP stdio server instance.
func NewServer(cli *client.Client, in io.Reader, out io.Writer) *Server {
	return &Server{
		client: cli,
		in:     in,
		out:    out,
	}
}

// Serve reads JSON-RPC requests from the input stream and writes responses to the output stream.
func (s *Server) Serve(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	// Allow larger buffers for substantial source chunk payloads (up to 4MB)
	const maxCapacity = 4 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, CodeParseError, fmt.Sprintf("parse error: %v", err))
			continue
		}

		// Handle notifications (no ID, no response required)
		if req.ID == nil {
			continue
		}

		s.dispatch(ctx, &req)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("scanner error: %w", err)
	}
	return nil
}

// dispatch routes the incoming method to the appropriate handler.
func (s *Server) dispatch(ctx context.Context, req *JSONRPCRequest) {
	var resp *JSONRPCResponse

	switch req.Method {
	case "initialize":
		resp = s.handleInitialize(req)
	case "tools/list":
		resp = s.handleToolsList(req)
	case "tools/call":
		resp = s.handleToolsCall(ctx, req)
	case "ping":
		resp = &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	default:
		resp = &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    CodeMethodNotFound,
				Message: fmt.Sprintf("unsupported method '%s'", req.Method),
			},
		}
	}

	if resp != nil {
		s.send(resp)
	}
}

func (s *Server) send(resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal response: %v\n", err)
		return
	}
	_, _ = s.out.Write(append(data, '\n'))
}

func (s *Server) sendError(id interface{}, code int, message string) {
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	s.send(resp)
}
