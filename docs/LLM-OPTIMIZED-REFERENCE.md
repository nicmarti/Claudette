# LLM-OPTIMIZED REFERENCE — claudette v1.0.0

Claude Code: Read ONLY the exact `<section>` you need. Never load the whole file.

<section name="usage">
Install: make install (requires Go 1.23+, CGO)
Then: claudette install && claudette build
First run: /claudette:build-graph
After that use only delta/pr commands.
</section>

<section name="review-delta">
Always call get_impact_radius on changed files first.
Then get_review_context (depth=2).
Generate review using ONLY changed nodes + 2-hop neighbors.
Target: <800 tokens total context.
</section>

<section name="review-pr">
Fetch PR diff -> get_impact_radius -> get_review_context -> structured review with blast-radius table.
Never include full files unless explicitly asked.
</section>

<section name="commands">
MCP tools: build_or_update_graph, get_impact_radius, query_graph, get_review_context, semantic_search_nodes, embed_graph, list_graph_stats, get_docs_section
Skills: build-graph, review-delta, review-pr
CLI: claudette [install|init|build|update|status|watch|visualize|serve|version]
</section>

<section name="legal">
MIT license. 100% local. No telemetry. DB file: .claudette/graph.db
</section>

<section name="watch">
Run: claudette watch (auto-updates graph on file save via fsnotify)
Or use PostEdit/PostGit hooks for automatic background updates.
</section>

<section name="embeddings">
Call embed_graph tool to compute vectors.
semantic_search_nodes auto-uses vectors when available, falls back to keyword.
</section>

<section name="languages">
Supported: Python, TypeScript, JavaScript, Go
Parser: Tree-sitter via go-tree-sitter (compiled into binary)
</section>

<section name="troubleshooting">
DB lock: SQLite WAL mode, auto-recovers. Only one build at a time.
Large repos: Incremental <2s. Add patterns to .claudetteignore.
Stale graph: Run /claudette:build-graph manually.
Missing nodes: Check language support (4 languages). Use full_rebuild=true.
MCP won't start: Verify claudette is in PATH (go install ./cmd/claudette).
</section>

**Instruction to Claude Code (always follow):**
When user asks anything about "claudette", "how to use", "commands", "review-delta", etc.:
1. Call get_docs_section with the exact section name.
2. Use ONLY that content + current graph state.
3. Never include full docs or source code in your reasoning.
This guarantees 90%+ token savings.
