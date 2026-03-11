# Claudette — User Guide

**Version:** v1.0.0

## Installation

```bash
make install             # builds and installs the claudette binary
claudette install        # creates .mcp.json for Claude Code
claudette build          # parse your codebase
```

Restart Claude Code or run `/mcp` to pick up the MCP server.

### Prerequisites

- Go 1.23+
- CGO enabled (required for tree-sitter and SQLite)
- `$GOPATH/bin` in your `PATH`

## Core Workflow

### 1. Build the graph (first time only)
```
/claudette:build-graph
```
Parses all git-tracked files in your codebase.

### 2. Review changes (daily use)
```
/claudette:review-delta
```
Reviews only files changed since last commit + everything impacted. Significantly fewer tokens than a full review.

### 3. Review a PR
```
/claudette:review-pr
```
Comprehensive structural review of a branch diff with blast-radius analysis.

### 4. Watch mode (optional)
```bash
claudette watch
```
Auto-updates the graph on every file save. Zero manual work.

### 5. Visualize the graph (optional)
```bash
claudette visualize
open .claudette/graph.html
```
Interactive D3.js force-directed graph. Starts collapsed (File nodes only) — click a file to expand its children. Use the search bar to filter, and click legend edge types to toggle visibility.

### 6. Semantic search (optional)
Use `embed_graph` MCP tool to compute vectors. `semantic_search_nodes` automatically uses vector similarity when available, falls back to keyword matching.

## Supported Languages

| Language | Extensions |
|----------|------------|
| Python | `.py` |
| JavaScript | `.js`, `.jsx` |
| TypeScript | `.ts`, `.tsx` |
| Go | `.go` |

More languages can be added by extending `internal/parser/languages.go`.

## What Gets Indexed

- **Nodes**: Files, Classes, Functions/Methods, Types, Tests
- **Edges**: CALLS, IMPORTS_FROM, INHERITS, IMPLEMENTS, CONTAINS, TESTED_BY, DEPENDS_ON

See [schema.md](schema.md) for full details.

## Ignore Patterns

By default, these paths are excluded from indexing:

```
.claudette/**      node_modules/**    .git/**
__pycache__/**     *.pyc              .venv/**
venv/**            dist/**            build/**
.next/**           target/**          *.min.js
*.min.css          *.map              *.lock
package-lock.json  yarn.lock          *.db
*.sqlite           *.db-journal       *.db-wal
```

To add custom patterns, create a `.claudetteignore` file in your repo root (same syntax as `.gitignore`):

```
generated/**
vendor/**
*.generated.ts
```
