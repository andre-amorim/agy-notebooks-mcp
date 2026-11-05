package client

import "time"

// Notebook represents high-level metadata for a Gemini Notebook (NotebookLM).
type Notebook struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"displayName"`
	LastModified time.Time `json:"lastModified"`
	SourceCount  int       `json:"sourceCount,omitempty"`
	Summary      string    `json:"summary,omitempty"`
}

// NotebookSource represents a specific text source, document, or research note in a notebook.
type NotebookSource struct {
	ID         string    `json:"id"`
	NotebookID string    `json:"notebookId"`
	Title      string    `json:"title"`
	Type       string    `json:"type"` // e.g., "doc", "pdf", "note", "url"
	Content    string    `json:"content"`
	WordCount  int       `json:"wordCount,omitempty"`
	CreatedAt  time.Time `json:"createdAt,omitempty"`
}

// QueryResponse represents a synthesized response grounded in notebook sources.
type QueryResponse struct {
	NotebookID string   `json:"notebookId"`
	Query      string   `json:"query"`
	Answer     string   `json:"answer"`
	Citations  []string `json:"citations,omitempty"`
}
