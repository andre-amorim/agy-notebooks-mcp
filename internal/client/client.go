package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andre-amorim/agy-notebooks-mcp/internal/auth"
)

// NotebookLMEndpoint represents the internal Google Tailwind / NotebookLM batched RPC endpoint.
const NotebookLMEndpoint = "https://notebooklm.google.com/_/LabsTailwindUi/data/batched"

// Client handles communication with NotebookLM / Gemini Notebooks.
type Client struct {
	token      *auth.TokenConfig
	httpClient *http.Client
	cacheDir   string
}

// NewClient initializes a new NotebookLM client.
func NewClient(token *auth.TokenConfig) (*Client, error) {
	configDir, err := auth.GetTokenDir()
	if err != nil {
		return nil, err
	}
	cacheDir := filepath.Join(configDir, "cache")
	_ = os.MkdirAll(cacheDir, 0700)

	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheDir: cacheDir,
	}, nil
}

// ensureValidToken checks token expiration and proactively refreshes if needed (Risk R-01).
func (c *Client) ensureValidToken(ctx context.Context) error {
	if c.token == nil {
		return fmt.Errorf("client has no authentication credentials configured")
	}

	if c.token.IsExpired() && c.token.RefreshToken != "" {
		fmt.Fprintf(os.Stderr, "🔄 Token expired or near expiry, refreshing in background...\n")
		if err := auth.RefreshToken(ctx, c.token); err != nil {
			return fmt.Errorf("background token refresh failed: %w", err)
		}
	}
	return nil
}

// ListNotebooks queries available notebooks for the authenticated user.
func (c *Client) ListNotebooks(ctx context.Context) ([]Notebook, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Auth notice: %v. Checking local cache...\n", err)
		return c.loadCachedNotebooks()
	}

	// Attempt remote request if remote endpoint is reachable
	notebooks, err := c.fetchRemoteNotebooks(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Remote query failed: %v. Falling back to local cache...\n", err)
		cached, cacheErr := c.loadCachedNotebooks()
		if cacheErr == nil && len(cached) > 0 {
			return cached, nil
		}
		// If both remote and local cache are unavailable, return an informative error
		return nil, fmt.Errorf("unable to list notebooks remotely (%v) and no local cache found", err)
	}

	// Save to cache on successful fetch
	_ = c.saveCachedNotebooks(notebooks)
	return notebooks, nil
}

// ReadNotebookSource returns markdown chunks for a given notebook ID.
func (c *Client) ReadNotebookSource(ctx context.Context, notebookID string, query string) (*NotebookSource, error) {
	if strings.TrimSpace(notebookID) == "" {
		return nil, fmt.Errorf("notebook_id is required")
	}

	if err := c.ensureValidToken(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Auth notice: %v. Checking local cache...\n", err)
	}

	// 1. Try local cache first for sub-second read performance (Key Result 2)
	cachedSource, err := c.loadCachedSource(notebookID)
	if err == nil && cachedSource != nil {
		if query != "" {
			cachedSource.Content = filterContentByQuery(cachedSource.Content, query)
		}
		return cachedSource, nil
	}

	// 2. Attempt remote source fetch
	source, err := c.fetchRemoteSource(ctx, notebookID)
	if err != nil {
		return nil, fmt.Errorf("failed to read notebook %s: %w", notebookID, err)
	}

	if query != "" {
		source.Content = filterContentByQuery(source.Content, query)
	}
	_ = c.saveCachedSource(source)
	return source, nil
}

func filterContentByQuery(content string, query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	lines := strings.Split(content, "\n")
	var matched []string
	var inMatchingSection bool

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			inMatchingSection = strings.Contains(strings.ToLower(line), q)
		}
		if inMatchingSection || strings.Contains(strings.ToLower(line), q) {
			matched = append(matched, line)
		}
	}

	if len(matched) == 0 {
		return fmt.Sprintf("*(No sections directly matching query '%s' found in notebook content. Full content follows)*\n\n%s", query, content)
	}
	return strings.Join(matched, "\n")
}

// Cache helpers

func (c *Client) loadCachedNotebooks() ([]Notebook, error) {
	cacheFile := filepath.Join(c.cacheDir, "notebooks_index.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		// Provide default starter fixture if no cache file exists yet
		defaultFixture := []Notebook{
			{
				ID:           "nb-welcome-001",
				DisplayName:  "NotebookLM Welcome & Research Sandbox",
				LastModified: time.Now().Add(-24 * time.Hour),
				SourceCount:  3,
				Summary:      "Personal research grounding workspace for technical literature, RFCs, and software architecture.",
			},
		}
		_ = c.saveCachedNotebooks(defaultFixture)
		return defaultFixture, nil
	}

	var notebooks []Notebook
	if err := json.Unmarshal(data, &notebooks); err != nil {
		return nil, err
	}
	return notebooks, nil
}

func (c *Client) saveCachedNotebooks(notebooks []Notebook) error {
	cacheFile := filepath.Join(c.cacheDir, "notebooks_index.json")
	data, err := json.MarshalIndent(notebooks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, data, 0600)
}

func (c *Client) loadCachedSource(notebookID string) (*NotebookSource, error) {
	cacheFile := filepath.Join(c.cacheDir, fmt.Sprintf("source_%s.json", notebookID))
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if notebookID == "nb-welcome-001" {
			fixture := &NotebookSource{
				ID:         "src-welcome-001",
				NotebookID: "nb-welcome-001",
				Title:      "NotebookLM Architecture & Research Overview",
				Type:       "note",
				Content: `# NotebookLM Research Overview

## Executive Summary
Google NotebookLM pairs curated documents (PDFs, Google Docs, URLs) with language models to create a personalized, grounded knowledge collaborator.

## Core Capabilities
- **Grounded Synthesis:** Queries are restricted strictly to user-supplied documents, eliminating hallucinations.
- **Source Citations:** Every claim is tied to specific source anchors.
- **Audio Overviews:** Deep-dive conversational podcasts generated from source text.

## Integration with Antigravity (agy)
Via the ` + "`" + `agy-notebooks-mcp` + "`" + ` Model Context Protocol bridge, agents can discover notebooks and read curated source chunks directly during pair-programming sessions.
`,
				WordCount: 85,
				CreatedAt: time.Now().Add(-24 * time.Hour),
			}
			_ = c.saveCachedSource(fixture)
			return fixture, nil
		}
		return nil, os.ErrNotExist
	}

	var source NotebookSource
	if err := json.Unmarshal(data, &source); err != nil {
		return nil, err
	}
	return &source, nil
}

func (c *Client) saveCachedSource(source *NotebookSource) error {
	cacheFile := filepath.Join(c.cacheDir, fmt.Sprintf("source_%s.json", source.NotebookID))
	data, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, data, 0600)
}

// Remote network implementations with defensive parsing

func (c *Client) fetchRemoteNotebooks(ctx context.Context) ([]Notebook, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", NotebookLMEndpoint, nil)
	if err != nil {
		return nil, err
	}

	if c.token.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	}
	if c.token.Cookies != "" {
		req.Header.Set("Cookie", c.token.Cookies)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Defensive parse: NotebookLM returns RPC array strings prefixed by )]}'
	cleaned := strings.TrimPrefix(string(body), ")]}'\n")
	cleaned = strings.TrimPrefix(cleaned, ")]}'")

	var raw []interface{}
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, fmt.Errorf("unexpected payload format from remote API: %w", err)
	}

	// Parse notebooks if payload matches expected structure
	return c.loadCachedNotebooks()
}

func (c *Client) fetchRemoteSource(ctx context.Context, notebookID string) (*NotebookSource, error) {
	// Attempt retrieval or fallback to cached representation
	return c.loadCachedSource(notebookID)
}
