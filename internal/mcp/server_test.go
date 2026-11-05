package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/andre-amorim/agy-notebooks-mcp/internal/auth"
	"github.com/andre-amorim/agy-notebooks-mcp/internal/client"
)

func TestMCPServerLifecycle(t *testing.T) {
	token := &auth.TokenConfig{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	cli, err := client.NewClient(token)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	requests := []string{
		// 1. Initialize
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		// 2. Initialized notification
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		// 3. List tools
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		// 4. Call list_notebooks
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_notebooks","arguments":{}}}`,
		// 5. Call read_notebook_source
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"read_notebook_source","arguments":{"notebook_id":"nb-welcome-001"}}}`,
		// 6. Unknown method
		`{"jsonrpc":"2.0","id":5,"method":"unknown/method"}`,
	}

	in := bytes.NewBufferString(strings.Join(requests, "\n") + "\n")
	out := &bytes.Buffer{}

	server := NewServer(cli, in, out)
	ctx := context.Background()

	if err := server.Serve(ctx); err != nil {
		t.Fatalf("server.Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 5 { // 5 responses for 5 requests with IDs (the notification has no response)
		t.Fatalf("expected 5 responses, got %d. Output:\n%s", len(lines), out.String())
	}

	// Verify initialize response
	var initResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("failed to parse init response: %v", err)
	}
	if initResp.Error != nil {
		t.Errorf("init returned error: %v", initResp.Error)
	}

	// Verify tools/list response
	var toolsResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &toolsResp); err != nil {
		t.Fatalf("failed to parse tools/list response: %v", err)
	}
	resMap := toolsResp.Result.(map[string]interface{})
	toolsList := resMap["tools"].([]interface{})
	if len(toolsList) < 2 {
		t.Errorf("expected at least 2 tools, got %d", len(toolsList))
	}

	// Verify tools/call list_notebooks
	var callResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[2]), &callResp); err != nil {
		t.Fatalf("failed to parse tools/call response: %v", err)
	}
	if callResp.Error != nil {
		t.Errorf("tools/call list_notebooks returned error: %v", callResp.Error)
	}

	// Verify unknown method response
	var unknownResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[4]), &unknownResp); err != nil {
		t.Fatalf("failed to parse unknown method response: %v", err)
	}
	if unknownResp.Error == nil {
		t.Errorf("expected error for unknown method, got nil")
	}
}
