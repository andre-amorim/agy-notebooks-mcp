# 📓 agy-notebooks-mcp

[![Go Reference](https://pkg.go.dev/badge/github.com/andre-amorim/agy-notebooks-mcp.svg)](https://pkg.go.dev/github.com/andre-amorim/agy-notebooks-mcp)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A lightweight, sovereign, pure **Go Model Context Protocol (MCP) server** that provides read-only inspection tools for **Google Antigravity (`agy`)** agents to discover, search, and ground reasoning directly in user-curated **Gemini Notebooks / NotebookLM** sources.

---

## 🎯 Motivation & Architecture

The local development workflow pairs `agy` with an Obsidian OKF vault for long-term project memory, while Gemini Notebooks (NotebookLM) serves as the primary external learning engine. Currently, these systems are siloed, requiring manual copy-pasting to ground agent sessions in curated notebook sources.

`agy-notebooks-mcp` bridges this gap over standard `stdio` JSON-RPC 2.0:

```
┌────────────────────────────────┐
│   Google Antigravity (`agy`)   │
└───────────────┬────────────────┘
                │ JSON-RPC 2.0 (stdio)
                ▼
┌────────────────────────────────────────────────────────┐
│           `agy-notebooks-mcp` (Go Subprocess)          │
│  • Sub-second response times (<2.5s)                   │
│  • Proactive OAuth2 token auto-refresh (zero restarts) │
│  • Defensive parsing & local caching fallback          │
└───────┬────────────────────────────────────────┬───────┘
        │ Google OAuth2 / Loopback               │ RPC
        ▼                                        ▼
┌────────────────────────────────┐   ┌───────────────────────────┐
│   Local Token Cache (~/.config)│   │  Gemini Notebooks Backend │
└────────────────────────────────┘   └───────────────────────────┘
```

---

## ⚡ Quick Start

### 1. Build and Install

#### Option A: Nix Flake (Recommended for Reproducibility)
```bash
nix build .#
./result/bin/agy-notebooks-mcp -version
```

#### Option B: Standard Go Toolchain
```bash
go build -o bin/agy-notebooks-mcp ./cmd/agy-notebooks-mcp
```

### 2. Authenticate

Run the interactive OAuth2 login command:
```bash
./bin/agy-notebooks-mcp auth login
```
*Note for ChromeOS / Crostini / Headless users:* If running inside a container where the loopback browser listener cannot be opened directly, use the manual auth flag:
```bash
./bin/agy-notebooks-mcp auth login --manual-auth
```
Or manually configure an existing token:
```bash
./bin/agy-notebooks-mcp auth set-token <YOUR_TOKEN>
# or set environment variable:
export NOTEBOOKLM_TOKEN="..."
```

Check your authentication status anytime:
```bash
./bin/agy-notebooks-mcp auth status
```

---

## 🔌 Antigravity (`agy`) Integration

Add the server to your Antigravity MCP configuration (`settings.json`):

```json
{
  "mcpServers": {
    "gemini-notebooks": {
      "command": "/absolute/path/to/agy-notebooks-mcp",
      "args": []
    }
  }
}
```

### Exposed MCP Tools

| Tool | Description | Arguments |
| :--- | :--- | :--- |
| `list_notebooks` | Discovers and lists all accessible Gemini Notebooks with IDs and metadata. | *None* |
| `read_notebook_source` | Reads source chunks and notes for a specific notebook. | `notebook_id` (string, required), `query` (string, optional) |

---

## 🛡️ Scope & Boundaries

* **In Scope:** Read-only queries, Go `stdio` transport, local token caching with auto-refresh, defensive fallbacks.
* **Out of Scope:** Write/edit operations, audio podcast generation, multi-user/multi-tenant credential vaults.

---

## 📄 License

GPLv3 © 2026 Andre Amorim.
