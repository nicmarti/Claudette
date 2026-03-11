# Architecture

## System Overview

Claudette is a Claude Code plugin that maintains a persistent, incrementally-updated knowledge graph of a codebase. It provides structural understanding of code relationships to make code reviews faster and more context-aware.

Written in Go as a single binary with no runtime dependencies beyond the system's C library (for CGO/tree-sitter/SQLite).

## Component Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                        Claude Code                           │
│                                                              │
│  Skills (SKILL.md)          Hooks (hooks.json)               │
│  ├── build-graph            ├── PostEdit → incremental       │
│  ├── review-delta           └── PostGit  → incremental       │
│  └── review-pr                                               │
│          │                        │                          │
│          ▼                        ▼                          │
│  ┌────────────────────────────────────────────┐              │
│  │            MCP Server (stdio)              │              │
│  │                                            │              │
│  │  Tools:                                    │              │
│  │  ├── build_or_update_graph                 │              │
│  │  ├── get_impact_radius                     │              │
│  │  ├── query_graph                           │              │
│  │  ├── get_review_context                    │              │
│  │  ├── semantic_search_nodes                 │              │
│  │  ├── embed_graph                           │              │
│  │  ├── list_graph_stats                      │              │
│  │  └── get_docs_section                      │              │
│  └────────────────┬───────────────────────────┘              │
└───────────────────┼──────────────────────────────────────────┘
                    │
        ┌───────────┼───────────────┐
        ▼           ▼               ▼
   ┌─────────┐ ┌─────────┐  ┌─────────────┐
   │ Parser  │ │  Graph  │  │ Incremental │
   │         │ │  Store  │  │   Engine    │
   └────┬────┘ └────┬────┘  └──────┬──────┘
        │           │              │
        ▼           ▼              ▼
   Tree-sitter   SQLite DB      git diff
   grammars      (.claudette/   subprocess
   (compiled     graph.db)
    into binary)
```

## Go Package Layout

```
cmd/claudette/main.go        CLI entry point (cobra commands)
internal/
  server/server.go            MCP server (stdio, mcp-go library)
  tools/tools.go              MCP tool implementations
  tools/helpers.go            Shared helpers for tool handlers
  graph/store.go              SQLite-backed graph storage
  graph/models.go             Node, Edge, Stats types
  graph/bfs.go                BFS impact analysis
  parser/parser.go            Tree-sitter AST parsing
  parser/languages.go         Language definitions and node type mappings
  parser/testpatterns.go      Test file/function detection
  incremental/incremental.go  Full build and incremental update logic
  incremental/fileutil.go     File discovery and ignore patterns
  incremental/gitops.go       Git diff and tracked file listing
  incremental/watch.go        Filesystem watcher (fsnotify)
  embeddings/embeddings.go    Vector embedding support
  visualization/              D3.js HTML graph generator
```

## Data Flow

### Full Build
1. `GetAllTrackedFiles()` gathers all git-tracked files with supported extensions
2. For each file, `CodeParser.ParseFile()` uses tree-sitter to extract the AST
3. AST walker identifies structural nodes (classes, functions, imports) and edges (calls, inheritance)
4. `GraphStore.StoreFileNodesEdges()` persists to SQLite with file hash for change detection
5. Metadata updated with timestamp

### Incremental Update
1. `GetChangedFiles()` runs `git diff --name-only` against the base ref
2. Changed files are re-parsed (others skipped via hash comparison)
3. Only affected rows in SQLite are updated

### Review Context Generation
1. Changed files identified (git diff or explicit list)
2. `BFSImpact()` performs BFS from changed nodes through the graph
3. Source snippets extracted for changed areas only
4. Review guidance generated (test coverage gaps, wide blast radius warnings)
5. Assembled into a structured, token-efficient context for Claude

## Storage

### SQLite
- **nodes** table: id, kind, name, qualified_name, file_path, line_start/end, language, etc.
- **edges** table: id, kind, source_qualified, target_qualified, file_path, line
- **metadata** table: key-value pairs (last_updated, build_type)

Indexes on qualified_name, file_path, and edge source/target for fast lookups.

WAL mode enabled for concurrent read access during updates.

### Qualified Names
Nodes are uniquely identified by qualified names:
- Files: relative path (e.g., `internal/graph/store.go`)
- Functions: `file_path::function_name` (e.g., `internal/graph/store.go::NewGraphStore`)
- Methods: `file_path::TypeName.method_name` (e.g., `internal/graph/store.go::GraphStore.GetStats`)

## Parsing Strategy

Tree-sitter provides language-agnostic AST access. The parser:
1. Walks the AST recursively
2. Pattern-matches on node types (language-specific mappings in `ClassTypes`, `FunctionTypes`, `ImportTypes`, `CallTypes`)
3. Extracts names, parameters, return types, base classes
4. Identifies calls within function bodies
5. Resolves imports to module paths

This approach is more robust than tree-sitter queries across grammar versions.

## Visualization

The `visualization` package generates an interactive D3.js force-directed graph as a self-contained HTML file. It reads all nodes and edges from the SQLite store and renders them in the browser, allowing developers to visually explore code relationships, filter by node kind, and inspect dependencies.

## Impact Analysis Algorithm

BFS from seed nodes (changed files' contents):
1. Seed = all qualified names in changed files
2. For each node in frontier:
   - Follow forward edges (what this node affects)
   - Follow reverse edges (what depends on this node)
3. Expand up to `max_depth` hops (default: 2)
4. Collect all reached nodes as "impacted"

This captures both downstream effects (things that call changed code) and upstream context (things that the changed code depends on).
