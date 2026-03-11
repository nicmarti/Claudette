# All Available Commands

## Skills (Claude Code slash commands)

### `/claudette:build-graph`
Build or update the knowledge graph.
- First time: performs a full build
- Subsequent: incremental update (only changed files)

### `/claudette:review-delta`
Review only changes since last commit.
- Auto-detects changed files via git diff
- Computes blast radius (2-hop default)
- Generates structured review with guidance

### `/claudette:review-pr`
Review a PR or branch diff.
- Uses main/master as base
- Full impact analysis across all PR commits
- Structured output with risk assessment

## MCP Tools

### `build_or_update_graph`
```
full_rebuild: bool = false   # true for full re-parse
repo_root: string            # auto-detected
base: string = "HEAD~1"     # git diff base
```

### `get_impact_radius`
```
changed_files: []string      # auto-detected from git
max_depth: int = 2           # hops in graph
repo_root: string
base: string = "HEAD~1"
```

### `query_graph`
```
pattern: string   # callers_of, callees_of, imports_of, importers_of,
                  # children_of, tests_for, inheritors_of, file_summary
target: string    # node name, qualified name, or file path
repo_root: string
```

### `get_review_context`
```
changed_files: []string
max_depth: int = 2
include_source: bool = true
max_lines_per_file: int = 200
repo_root: string
base: string = "HEAD~1"
```

### `semantic_search_nodes`
```
query: string        # search string
kind: string         # File, Class, Function, Type, Test
limit: int = 20
repo_root: string
```

### `embed_graph`
```
repo_root: string
```

### `list_graph_stats`
```
repo_root: string
```

### `get_docs_section`
```
section_name: string   # usage, review-delta, review-pr, commands, legal, watch, embeddings, languages, troubleshooting
```

## CLI Commands

```bash
# Register MCP server with Claude Code
claudette install           # also available as: claudette init
claudette install --dry-run # preview without writing files

# Full build
claudette build
claudette build --repo /path/to/project

# Incremental update
claudette update
claudette update --base origin/main  # custom base ref

# Check status
claudette status

# Watch mode
claudette watch

# Generate graph visualisation
claudette visualize

# Start MCP server
claudette serve

# Show version
claudette version
```

## API Response Schemas

### `build_or_update_graph`
```json
{
  "files_parsed": 150,
  "total_nodes": 420,
  "total_edges": 380,
  "errors": [{"file": "bad.go", "error": "parse error"}]
}
```

### `get_impact_radius`
```json
{
  "changed_nodes": [
    {"id": 1, "kind": "Function", "name": "Login", "qualified_name": "src/auth.go::Login", "file_path": "src/auth.go", "line_start": 10, "line_end": 25, "language": "go", "is_test": false}
  ],
  "impacted_nodes": [],
  "impacted_files": ["src/routes.go", "src/middleware.go"],
  "edges": [
    {"id": 5, "kind": "CALLS", "source": "src/auth.go::Login", "target": "src/db.go::GetUser", "file_path": "src/auth.go", "line": 15}
  ]
}
```

### `query_graph`
```json
{
  "results": [
    {"id": 1, "kind": "Function", "name": "Login", "qualified_name": "...", "file_path": "...", "line_start": 10, "line_end": 25}
  ]
}
```

### `get_review_context`
```json
{
  "impact": {},
  "source_snippets": {
    "src/auth.go": "func Login(...) {\n    ..."
  },
  "review_guidance": "Focus on: Login() changed parameters, check callers in routes.go"
}
```

### `semantic_search_nodes`
```json
{
  "results": [
    {"id": 1, "kind": "Function", "name": "Authenticate", "qualified_name": "...", "file_path": "...", "similarity_score": 0.8732}
  ]
}
```

### `list_graph_stats`
```json
{
  "total_nodes": 420,
  "total_edges": 380,
  "nodes_by_kind": {"File": 50, "Function": 280, "Class": 60, "Type": 15, "Test": 15},
  "edges_by_kind": {"CALLS": 200, "CONTAINS": 100, "IMPORTS_FROM": 50, "INHERITS": 20, "TESTED_BY": 10},
  "languages": ["python", "typescript", "go"],
  "files_count": 50,
  "last_updated": "2026-03-11T14:30:00"
}
```
