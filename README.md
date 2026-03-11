# Claudette

An MCP server for Claude Code, built in Go.

## Prerequisites

- Go 1.23+
- CGO enabled (required for SQLite support)

## Installation

```bash
make install
```

This runs `go install ./cmd/claudette`, which places the `claudette` binary in your `$GOPATH/bin`. Make sure `$GOPATH/bin` is in your `PATH`.

## MCP Configuration

The project includes a `.mcp.json` that configures Claude Code to use the installed binary:

```json
{
  "mcpServers": {
    "claudette": {
      "command": "claudette",
      "args": ["serve"]
    }
  }
}
```

After running `make install`, restart Claude Code or run `/mcp` to reconnect.

## Development

```bash
make build    # Build to ./bin/claudette
make test     # Run tests
make fmt      # Format code
make vet      # Run go vet
make tidy     # Run go mod tidy
make clean    # Remove build artifacts
```
