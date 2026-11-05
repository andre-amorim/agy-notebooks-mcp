package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// handleInitialize handles the initial client handshake.
func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "agy-notebooks-mcp",
				"version": "0.1.0",
			},
		},
	}
}

// handleToolsList registers the available read-only MCP tools.
func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	tools := []Tool{
		{
			Name:        "list_notebooks",
			Description: "List all accessible Gemini Notebooks (NotebookLM) with their IDs, titles, and modification timestamps.",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		{
			Name:        "read_notebook_source",
			Description: "Read text and source chunks from a specified Gemini Notebook (NotebookLM) ID.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"notebook_id": map[string]interface{}{
						"type":        "string",
						"description": "The unique ID of the notebook to read.",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Optional search term or keyword to filter specific sections.",
					},
				},
				Required: []string{"notebook_id"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// handleToolsCall executes tool invocations with graceful error handling.
func (s *Server) handleToolsCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    CodeInvalidParams,
				Message: fmt.Sprintf("invalid tools/call params: %v", err),
			},
		}
	}

	switch params.Name {
	case "list_notebooks":
		notebooks, err := s.client.ListNotebooks(ctx)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolCallResult{
					IsError: true,
					Content: []TextContent{
						{Type: "text", Text: fmt.Sprintf("Failed to list notebooks: %v", err)},
					},
				},
			}
		}

		var out string
		if len(notebooks) == 0 {
			out = "No notebooks found in account."
		} else {
			out = "### 📓 Available Gemini Notebooks\n\n"
			for _, nb := range notebooks {
				out += fmt.Sprintf("- **%s** (`%s`)\n  - Last Modified: %s\n  - Sources: %d\n  - Summary: %s\n\n",
					nb.DisplayName, nb.ID, nb.LastModified.Format(time.RFC3339), nb.SourceCount, nb.Summary)
			}
		}

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolCallResult{
				Content: []TextContent{
					{Type: "text", Text: out},
				},
			},
		}

	case "read_notebook_source":
		notebookID, _ := params.Arguments["notebook_id"].(string)
		query, _ := params.Arguments["query"].(string)

		if notebookID == "" {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolCallResult{
					IsError: true,
					Content: []TextContent{
						{Type: "text", Text: "Missing required argument 'notebook_id'."},
					},
				},
			}
		}

		src, err := s.client.ReadNotebookSource(ctx, notebookID, query)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolCallResult{
					IsError: true,
					Content: []TextContent{
						{Type: "text", Text: fmt.Sprintf("Error reading notebook '%s': %v", notebookID, err)},
					},
				},
			}
		}

		content := fmt.Sprintf("# 📄 Notebook: %s (ID: `%s`)\n\n**Source Title:** %s\n\n%s",
			notebookID, src.NotebookID, src.Title, src.Content)

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolCallResult{
				Content: []TextContent{
					{Type: "text", Text: content},
				},
			},
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    CodeMethodNotFound,
				Message: fmt.Sprintf("tool '%s' not recognized", params.Name),
			},
		}
	}
}
