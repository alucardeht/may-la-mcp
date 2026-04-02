# May-la MCP

A lean, fast MCP server for code search and navigation. Written in Go, installed via npm.

**2.7 MB binary. 5 tools. Instant startup. Zero dependencies.**

## What it does

LLMs waste too many tool calls searching for code. May-la fixes that — each tool returns maximum useful context per call, so the model understands your codebase in 2-3 calls instead of 7+.

## Tools

### `info` — Project overview in one call
Tech stack detection, directory tree, file distribution, entry points. One call and the model knows where everything is.

### `read` — Smart file reading
Read files with automatic structural summary (functions, types, exports). Supports batch reading (multiple files in one call), line ranges, and reading a specific symbol by name.

### `search` — Search with function-level context
Search by term or regex. Instead of returning just the matching line, returns the **entire function** containing the match. Eliminates the grep-then-read loop.

### `find` — File discovery with metadata
Find files by glob pattern. Results grouped by directory with file sizes. Respects `.gitignore`.

### `symbols` — Code structure extraction
Extract functions, classes, types, interfaces, methods from files or directories. Supports Go, TypeScript, JavaScript, Python, Java, Rust, and C/C++. No LSP required — uses fast regex extraction.

## Installation

### Step 1: Install

```bash
npm install -g @alucardeht/may-la-mcp
```

### Step 2: Add to your client

**Claude Code:**
```bash
claude mcp add may-la -- may-la-mcp
```

**Cursor** (`~/.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "may-la": {
      "command": "may-la-mcp"
    }
  }
}
```

**Gemini CLI:**
```bash
gemini mcp add may-la -- may-la-mcp
```

**Other clients:** point to the command `may-la-mcp`.

### Supported Platforms

| OS | Architecture |
|----|-------------|
| macOS | arm64, amd64 |
| Linux | amd64 |
| Windows | amd64 |

### Build from source

```bash
git clone https://github.com/alucardeht/may-la-mcp.git
cd may-la-mcp
make build
# Binary at bin/mayla
```

Requires Go 1.22+. No CGO needed.

## Verify

```bash
# Check binary is installed
ls -lh ~/.mayla/mayla

# Test MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1.0"}}}' | may-la-mcp
```

In Claude Code: `/mcp` should show `may-la` in the list.

## Architecture

```
MCP Client (Claude, Cursor, etc.)
    ↓ stdio
may-la-mcp (Node wrapper)
    ↓ spawn
mayla (Go binary, 2.7 MB)
    ↓
5 read-only tools
```

No daemon. No database. No LSP servers. The binary starts, handles requests over stdio, and exits when the client disconnects.

### Symbol extraction

Regex-based extraction for 7 languages — no language servers needed:

| Language | Symbols detected |
|----------|-----------------|
| Go | functions, methods, types, interfaces, consts |
| TypeScript/JavaScript | functions, classes, interfaces, types, methods, arrow functions |
| Python | functions, classes, methods |
| Java | classes, interfaces, methods |
| Rust | functions, methods, structs, traits, enums, consts |
| C/C++ | functions, structs, unions, enums, defines |

### .gitignore aware

All tools respect `.gitignore` and skip common junk directories (`node_modules`, `vendor`, `__pycache__`, `dist`, `.git`, etc.).

## Project Structure

```
may-la-mcp/
├── cmd/mayla/             # Entry point (stdio MCP server)
├── internal/
│   ├── mcp/               # MCP protocol (JSON-RPC)
│   ├── tools/             # The 5 tools
│   ├── lang/              # Regex symbol extractors (7 languages)
│   ├── gitignore/         # .gitignore parser
│   └── encoding/          # File encoding (UTF-8, UTF-16)
├── pkg/
│   ├── protocol/          # JSON-RPC types
│   └── version/           # Version info
├── npm/                   # npm package wrapper
├── scripts/               # Installation scripts
└── Makefile
```

## Development

```bash
make build      # Build binary
make test       # Run tests
make install    # Install to /usr/local/bin
make fmt        # Format code
make lint       # Run linter
```

## Troubleshooting

**`may-la-mcp` not found after npm install**
- npm global bin dir may not be in PATH
- Check: `npm bin -g`
- Fix: add that directory to your PATH

**macOS: "Cannot be opened because the developer cannot be verified"**
```bash
xattr -d com.apple.quarantine ~/.mayla/mayla
```

**Binary not downloaded during install**
- Check internet connection
- Try: `npm install -g @alucardeht/may-la-mcp` again
- Manual download: [GitHub Releases](https://github.com/alucardeht/may-la-mcp/releases)

## License

Apache License 2.0
