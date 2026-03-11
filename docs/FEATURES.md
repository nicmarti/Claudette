# Features & Changelog

## v1.0.0 — Initial Release

Go rewrite of [code-review-graph](https://github.com/tirth8205/code-review-graph). Single binary, no runtime dependencies.

### Core Features
- Tree-sitter parsing for Python, JavaScript, TypeScript, Go
- SQLite-backed persistent knowledge graph with WAL mode
- Incremental updates via `git diff` (re-parses only changed files)
- BFS-based blast-radius analysis (configurable depth)
- Token-optimised review context generation
- Interactive D3.js force-directed graph visualisation
- Watch mode with filesystem notifications (fsnotify)
- MCP server (stdio transport) with 8 tools
- Claude Code skills: build-graph, review-delta, review-pr
- Auto-update hooks (PostEdit, PostGit)
- Semantic search with optional vector embeddings
- CLI with install, build, update, watch, status, visualize, serve commands

### Privacy & Data
- Zero telemetry
- All data stored locally in `.claudette/graph.db`
- No network calls during normal operation
- Single binary — no Python, no pip, no virtual environments
