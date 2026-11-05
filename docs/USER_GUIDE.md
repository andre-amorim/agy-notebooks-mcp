# 📖 User Guide & Operational Runbook: `agy-notebooks-mcp`

Welcome to the official user guide for **`agy-notebooks-mcp`**. This document provides clear, step-by-step instructions for getting started, authenticating your account, and using the server with your **Google Antigravity (`agy`)** pair-programmer.

---

## 1. What is `agy-notebooks-mcp`?

Think of **Google NotebookLM (Gemini Notebooks)** as your personal digital library of curated sources—books, research papers, YouTube transcripts, and PDF notes.

Think of **Antigravity (`agy`)** as your active pair programmer sitting in your terminal, writing code and managing your project workspace.

Previously, these two tools lived on separate islands:
- To use research from a notebook, you had to leave your terminal, open your browser, find the note, copy the text, and paste it back into your chat with the AI.

**`agy-notebooks-mcp` removes this friction.**  
It acts as an automated background bridge using the **Model Context Protocol (MCP)**. It gives your Antigravity agent the ability to look inside your notebooks, read the sources you have curated, and cite them directly while helping you code or research.

---

## 2. Architecture & File Layout

The server is a standalone, lightweight program written in pure Go with zero external runtime dependencies.

| Component | Standard Path | Description |
| :--- | :--- | :--- |
| **Executable Binary** | `~/.local/bin/agy-notebooks-mcp` or `~/bin/agy-notebooks-mcp` | Compiled standalone MCP server. |
| **Token Storage** | `~/.config/agy-notebooks/token.json` | Encrypted local OAuth2 credentials (`0600` permissions). |
| **Local Cache** | `~/.config/agy-notebooks/cache/` | Cached notebook metadata and sources for offline resilience. |
| **Antigravity Config** | `~/.gemini/config/mcp_config.json` | MCP server registration for Antigravity. |

Make sure the directory containing your binary is in your system `$PATH`.

---

## 3. First-Time Authentication Walkthrough

Before Antigravity can read your notebooks, you need to authenticate with your Google account once. The server stores an access and refresh token locally at `~/.config/agy-notebooks/token.json`.

### Option A: Standard Desktop Browser Authentication (macOS / Linux Desktop)

If you are running in a desktop environment with a graphical browser:

1. Open your terminal.
2. Start the interactive login:
   ```bash
   agy-notebooks-mcp auth login
   ```
3. A Google sign-in window will open in your default browser.
4. Select your Google account and grant read-only permissions.
5. Once you see **"Authentication Successful!"**, close the browser tab.
6. The terminal will confirm:
   ```text
   ✅ Authentication successful! Token saved to: ~/.config/agy-notebooks/token.json
   ```

---

### Option B: Headless / Container Authentication (`--manual-auth`)

If you are running in a headless server, remote SSH session, or inside a container where browser pop-ups cannot open a local port directly, use the **manual authentication flag**:

1. Open your terminal.
2. Run the manual login command:
   ```bash
   agy-notebooks-mcp auth login --manual-auth
   ```
3. The terminal will print a secure login URL:
   ```text
   👉 Open the following URL in your browser to authenticate:
   https://accounts.google.com/o/oauth2/v2/auth?...
   ```
4. Copy that URL, paste it into any web browser on any device, and sign in.
5. Google will display an authorization code on screen.
6. Copy the code, return to your terminal, paste it, and press **Enter**.
7. The terminal will confirm:
   ```text
   ✅ Authentication successful! Token saved to: ~/.config/agy-notebooks/token.json
   ```

---

### Verifying Sign-in Status

You can check whether your credentials are valid at any time:

```bash
agy-notebooks-mcp auth status
```

**Expected output:**
```text
Status: Authenticated ✅
Token Path: ~/.config/agy-notebooks/token.json
Access Token: Valid
Expires: 2026-11-05 12:00:00 UTC
Refresh Token: Present
```

> [!NOTE]
> **Automatic Background Token Refresh:**  
> You do **not** need to sign in every day. The server detects when your access token is close to expiring and quietly refreshes it in the background using your refresh token without interrupting your chat sessions.

---

## 4. How to Use with Antigravity (`agy`)

Once registered and authenticated, using the server is completely natural. You do not need to memorize special command syntax.

### Step 1: Verify the Server in Antigravity (`/mcp`)

When you launch `agy` in your terminal:
1. Type `/mcp` in the prompt box and hit **Enter**.
2. A status panel appears showing your active MCP connections.
3. Look for `gemini-notebooks`:
   - It should display a **green active indicator**.
   - It will list two available tools: `list_notebooks` and `read_notebook_source`.
4. Press `Escape` or type `/resume` to return to your normal chat.

---

### Step 2: Everyday Prompting Examples

You talk to Antigravity normally. The agent automatically decides when to call the MCP server based on what you ask.

#### Example A: Discovering What Notebooks You Have
If you want to know what research materials are available:
> **You:** *"Can you list the notebooks in my Gemini Notebooks library?"*  
> **Agent Action:** Runs `list_notebooks` in the background.  
> **Agent Answer:**  
> *"Here are the notebooks currently found in your account:*  
> *1. **Research Project Alpha** (`nb-001`)*  
> *2. **API Specification Drafts** (`nb-002`)*  
> *3. **Cloud Architecture Notes** (`nb-003`)*  
> *Which one would you like to explore?"*

---

#### Example B: Reading a Source to Ground Your Work
When you want the agent to cite specific documentation from your notebook:
> **You:** *"Look into my research notebook `nb-001` and read the source notes on distributed state machines."*  
> **Agent Action:** Calls `read_notebook_source(notebook_id="nb-001", query="state machines")`.  
> **Agent Answer:** Summarizes and cites the exact text from your curated source.

---

#### Example C: Cross-Referencing Knowledge into Project Notes
When you want to synthesize information into your local workspace:
> **You:** *"Extract the key takeaways on risk management from notebook `nb-002` and help me draft a project risk register."*  
> **Agent Action:** Reads the notebook source chunks and produces a clean, structured risk register.

---

## 5. Command-Line Reference Summary

| Command | What It Does | When to Use It |
| :--- | :--- | :--- |
| `agy-notebooks-mcp -version` | Prints the installed version and target architecture. | Quick sanity check. |
| `agy-notebooks-mcp auth status` | Displays login state, token file path, and expiry. | To verify credentials. |
| `agy-notebooks-mcp auth login` | Starts the automatic loopback login flow. | Desktop workstation setup. |
| `agy-notebooks-mcp auth login --manual-auth` | Starts manual code-entry login flow. | Remote servers and containers. |
| `agy-notebooks-mcp auth set-token <token>` | Manually saves a static token string. | For CI/CD or offline scripts. |
| `agy-notebooks-mcp auth logout` | Deletes stored credentials from disk. | To sign out completely. |
| `agy-notebooks-mcp` (no args) | Starts the JSON-RPC MCP server on `stdio`. | Used automatically by `agy`. |

---

## 6. Troubleshooting & FAQ

### Q1: The `/mcp` menu in `agy` shows `gemini-notebooks` as disconnected or failed.
* **Check the binary path:** Make sure the path specified in `~/.gemini/config/mcp_config.json` exists and is executable (`chmod +x <path>`).
* **Check permissions:** Test running `agy-notebooks-mcp -version` directly in your terminal. If it prints the version, the binary is healthy.
* **Restart the agent:** Completely exit `agy` (`/exit` or `Ctrl+D`) and start it again so it re-reads your MCP configuration.

### Q2: What happens if I have no Internet connection?
* The server features built-in **defensive caching**.
* Any notebooks you have previously queried are stored locally in `~/.config/agy-notebooks/cache/`.
* If the remote Google API cannot be reached, the server falls back to your local cache rather than crashing your session.

### Q3: Can I run this across a remote SSH machine?
* Yes! In `~/.gemini/config/mcp_config.json`, you can configure the MCP server to run via SSH:
  ```json
  {
    "mcpServers": {
      "gemini-notebooks": {
        "command": "ssh",
        "args": ["my-workstation", "~/.local/bin/agy-notebooks-mcp"]
      }
    }
  }
  ```

---

## 7. Open Source & License

This project is open-source under the GNU General Public License v3.0 (GPLv3). Contributions, issues, and feature requests are welcome on GitHub.
