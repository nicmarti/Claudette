<h1 align="center">Claudette</h1>

Claudette builds a structural knowledge graph of your codebase using [Tree-sitter](https://tree-sitter.github.io/tree-sitter/), tracks changes incrementally, and gives Claude precise context so it reads only what matters instead of re-reading your entire codebase on every task.

Based on the [code-review-graph](https://github.com/tirth8205/code-review-graph) project by Tirth Kanani, Claudette is the rewritten in Go for fast, single-binary deployment, implemented by Nicolas Martignole. 

## Supported Languages

| Language | Extensions |
|----------|------------|
| Python | `.py` |
| JavaScript | `.js`, `.jsx` |
| TypeScript | `.ts`, `.tsx` |
| Go | `.go` |

More languages can be added by extending `internal/parser/languages.go` with the appropriate tree-sitter grammar and node type mappings.

## Installation

### 1. Install Go (macOS)

If you don't have Go installed yet, the easiest way is via Homebrew:

```bash
brew install go
```

Alternatively, download the installer from [go.dev/dl](https://go.dev/dl/) and follow the instructions.

Verify the installation:

```bash
go version
```

Make sure `$GOPATH/bin` is in your `PATH`. Add this to your `~/.zshrc` if needed:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 2. Build and install Claudette

```bash
git clone https://github.com/nicmarti/Claudette.git
cd Claudette
make build
make install
```

This compiles the binary with CGO (required for tree-sitter and SQLite) and installs it to `$GOPATH/bin`.

Verify:

```bash
claudette version
```

### Register with Claude Code

Go to a standard project where you used Claude Code, then install claudette : 

```bash
claudette install
```

This creates a `.mcp.json` in your repository root (or merges into an existing one):

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

Then execute `claudette build` to create the local DB for all the code tracked by Github. Claudette
ignores untracked files, and respect the `.gitignore` file.

Restart Claude Code the run `/mcp` to connect to Claudette


## Getting Started

Open your project in Claude Code and ask:

```
Build the code review graph for this project
```

The initial build takes a few seconds. After that, the graph updates incrementally on changed files only.

You can also use `claudette build` from the command line

## How It Works (credits to Tirth Kanani)

The graph maps every function, class, import, call, inheritance relationship, and test in your codebase. When you ask Claude to review code or make changes, it queries the graph to determine what changed and what depends on those changes, then reads only the relevant files along with their blast-radius information rather than scanning everything.

You continue using Claude Code exactly as before. The graph operates in the background, updating itself as you work.

## CLI

```
claudette install     Register MCP server with Claude Code (creates .mcp.json)
claudette build       Full graph build (parse all tracked files)
claudette update      Incremental update (changed files only)
claudette watch       Auto-update on file changes
claudette status      Show graph statistics
claudette visualize   Generate interactive HTML graph
claudette serve       Start MCP server (stdio transport)
claudette version     Show version
```

All commands accept `--repo <path>` to specify the repository root (auto-detected by default).

## Slash Commands

| Command | Description |
|---------|-------------|
| `/claudette:build-graph` | Build or rebuild the code graph |
| `/claudette:review-delta` | Review changes since last commit |
| `/claudette:review-pr` | Full PR review with blast-radius analysis |

## MCP Tools

Claude uses these automatically once the graph is built.

| Tool | Description |
|------|-------------|
| `build_or_update_graph` | Build or incrementally update the graph |
| `get_impact_radius` | Blast radius of changed files |
| `get_review_context` | Token-optimised review context with structural summary |
| `query_graph` | Callers, callees, tests, imports, inheritance queries |
| `semantic_search_nodes` | Search code entities by name or meaning |
| `embed_graph` | Compute vector embeddings for semantic search |
| `list_graph_stats` | Graph size and health |
| `get_docs_section` | Retrieve documentation sections |

## Features

| Feature | Details |
|---------|---------|
| Incremental updates | Re-parses only changed files. Subsequent updates complete in under 2 seconds. |
| 4 languages | Python, TypeScript, JavaScript, Go |
| Blast-radius analysis | Shows exactly which functions, classes, and files are affected by any change |
| Auto-update hooks | Graph updates on every file edit and git commit without manual intervention |
| Semantic search | Optional vector embeddings for searching code entities by meaning |
| Interactive visualisation | D3.js force-directed graph with edge-type toggles and search |
| Local storage | SQLite file in `.claudette/`. No external database, no cloud dependency. |
| Watch mode | Continuous graph updates as you work |
| Single binary | No runtime dependencies. One `go install` and you're done. |

## Configuration

To exclude paths from indexing, create a `.claudetteignore` file in your repository root:

```
generated/**
*.generated.ts
vendor/**
node_modules/**
```

## Development

```bash
make build    # Build to ./bin/claudette
make test     # Run tests
make fmt      # Format code
make vet      # Run go vet
make tidy     # Run go mod tidy
make clean    # Remove build artifacts
```

To add a new language, edit `internal/parser/languages.go` and add:
1. The file extension mapping in `ExtensionToLanguage`
2. The tree-sitter grammar import and entry in `languageFunc`
3. Node type mappings in `ClassTypes`, `FunctionTypes`, `ImportTypes`, and `CallTypes`
4. A test fixture in `testdata/`

## Licence

MIT. See [LICENSE](LICENSE).
