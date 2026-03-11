# Troubleshooting

## Database lock errors
The graph uses SQLite with WAL mode. If you see lock errors:
- Ensure only one build process runs at a time
- The database auto-recovers; just retry
- Delete `.claudette/graph.db-wal` and `.claudette/graph.db-shm` if corrupt

## Large repositories (>10k files)
- Subsequent incremental updates are fast (<2s)
- Add more ignore patterns to `.claudetteignore`:
  ```
  generated/**
  vendor/**
  *.min.js
  ```

## Missing nodes after build
- Check that the file's language is supported (Python, JavaScript, TypeScript, Go)
- Check that the file isn't matched by an ignore pattern
- Ensure the file is tracked by git (`git ls-files` is used for file discovery)
- Run with `full_rebuild=true` to force a complete re-parse

## Graph seems stale
- Hooks auto-update on edit/commit
- If stale, run `/claudette:build-graph` manually
- Check that hooks are configured in `hooks/hooks.json`

## MCP server won't start
- Verify `claudette` is installed and in your PATH: `which claudette`
- If not found, run `make install` or `go install ./cmd/claudette`
- Check that `.mcp.json` contains the correct config: `"command": "claudette"` with `"args": ["serve"]`
- Re-run `claudette install` to regenerate the config

## Graph is empty (0 files parsed)
- The graph builder uses `git ls-files` to discover files
- Untracked files are not indexed — make sure to `git add` your source files
- Run `claudette build` after committing or staging files
