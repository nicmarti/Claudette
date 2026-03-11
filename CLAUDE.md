# CLAUDE.md — Claudette

## Project Overview

Claudette is an MCP server and CLI tool that builds a persistent, incremental knowledge graph of codebases using Tree-sitter. It gives Claude Code structural context (functions, classes, imports, call graphs) so it reads only what matters. Written in Go, single binary, no runtime dependencies.

Based on [code-review-graph](https://github.com/tirth8205/code-review-graph) by Tirth Kanani, rewritten in Go by Nicolas Martignole.

## Build & Run

```bash
CGO_ENABLED=1 go build -o bin/claudette ./cmd/claudette   # build
CGO_ENABLED=1 go install ./cmd/claudette                   # install to $GOPATH/bin
make build                                                  # same as above via Makefile
make install                                                # go install
make test                                                   # run tests (CGO_ENABLED=1)
```

CGO is **required** — tree-sitter and SQLite are C libraries.

## Project Structure

```
cmd/claudette/main.go         CLI entry point (cobra). All subcommands defined here.
internal/
  server/server.go            MCP server (stdio transport, mcp-go library)
  tools/tools.go              MCP tool handler implementations
  tools/helpers.go            Shared helpers for tools
  graph/store.go              SQLite-backed graph storage (WAL mode)
  graph/models.go             Node, Edge, Stats types
  graph/bfs.go                BFS impact/blast-radius analysis
  parser/parser.go            Tree-sitter AST parsing
  parser/languages.go         Language definitions (extensions, node type mappings)
  parser/testpatterns.go      Test file/function detection heuristics
  incremental/incremental.go  Full build + incremental update orchestration
  incremental/fileutil.go     File discovery, ignore patterns, hashing
  incremental/gitops.go       Git operations (diff, ls-files)
  incremental/watch.go        Filesystem watcher (fsnotify)
  embeddings/embeddings.go    Vector embedding support
  visualization/              D3.js HTML graph generator
skills/                       Claude Code skill definitions (SKILL.md files)
hooks/                        Claude Code hooks (PostEdit, PostGit auto-update)
testdata/                     Test fixtures (sample.go, sample.py, sample.ts)
docs/                         Full documentation
```

## Supported Languages

Only 4 languages currently: **Python** (.py), **JavaScript** (.js/.jsx), **TypeScript** (.ts/.tsx), **Go** (.go).

To add a new language: edit `internal/parser/languages.go` — add extension mapping, tree-sitter grammar, and node type mappings (ClassTypes, FunctionTypes, ImportTypes, CallTypes). Add a test fixture in `testdata/`.

## Key Design Decisions

- File discovery uses `git ls-files` — untracked files are not indexed
- SQLite with WAL mode for concurrent reads during updates
- Qualified names: `file_path::FunctionName` or `file_path::Type.Method`
- Ignore patterns loaded from `.claudetteignore` (gitignore syntax)
- Graph stored in `.claudette/graph.db` (gitignored)
- MCP server uses stdio transport (no HTTP)

## Testing

```bash
make test                         # all tests
CGO_ENABLED=1 go test ./internal/graph/...    # graph store tests
CGO_ENABLED=1 go test ./internal/parser/...   # parser tests
```

## Code Style

- Standard Go conventions: `gofmt`, `go vet`
- Packages under `internal/` — not importable externally
- Error handling: return errors up, log.Fatal only in CLI main
- No generics used — straightforward map/slice patterns
