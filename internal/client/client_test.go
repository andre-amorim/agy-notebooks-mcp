package client

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/andre-amorim/agy-notebooks-mcp/internal/auth"
)

func TestClientCacheAndFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	origConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origConfigHome)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	token := &auth.TokenConfig{
		AccessToken: "mock-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	cli, err := NewClient(token)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test ListNotebooks fallback/fixture
	notebooks, err := cli.ListNotebooks(ctx)
	if err != nil {
		t.Fatalf("expected notebooks, got error: %v", err)
	}
	if len(notebooks) == 0 {
		t.Fatalf("expected at least 1 notebook")
	}
	if notebooks[0].ID != "nb-welcome-001" {
		t.Errorf("expected ID nb-welcome-001, got %s", notebooks[0].ID)
	}

	// Test ReadNotebookSource
	src, err := cli.ReadNotebookSource(ctx, "nb-welcome-001", "")
	if err != nil {
		t.Fatalf("failed to read notebook source: %v", err)
	}
	if src.Title == "" {
		t.Errorf("expected non-empty title")
	}

	// Test Query Filtering
	filtered, err := cli.ReadNotebookSource(ctx, "nb-welcome-001", "Grounded Synthesis")
	if err != nil {
		t.Fatalf("failed to filter source: %v", err)
	}
	if !testing.Short() && len(filtered.Content) == 0 {
		t.Errorf("expected filtered content")
	}

	// Test Empty ID error
	_, err = cli.ReadNotebookSource(ctx, "", "")
	if err == nil {
		t.Errorf("expected error for empty notebook ID")
	}
}
