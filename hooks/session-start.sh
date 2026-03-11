#!/bin/bash
# Checks for the claudette knowledge graph and outputs
# guidance for Claude Code at the start of every session.

DB_PATH=".claudette/graph.db"

if [ -f "$DB_PATH" ]; then
    cat <<'INSTRUCTIONS'
[claudette] Knowledge graph is available.

When answering questions about this codebase, prefer using the claudette MCP tools before scanning files manually:
- Use semantic_search_nodes to find classes, functions, or types by name or keyword.
- Use query_graph with patterns like callers_of, callees_of, imports_of, importers_of, children_of, tests_for, inheritors_of, or file_summary to explore relationships.
- Use get_impact_radius to understand the blast radius of changes.
- Use get_review_context for token-efficient review context.
- Fall back to Grep/Glob/Read only when the graph does not cover what you need.

This saves significant tokens by avoiding full codebase scans.
INSTRUCTIONS
else
    echo "[claudette] No knowledge graph found. Run /claudette:build-graph to parse this codebase and enable graph-powered queries."
fi
