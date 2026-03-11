// Package server provides the MCP stdio server for claudette.
package server

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"claudette/internal/tools"
)

// Serve starts the MCP server on stdio transport.
func Serve() error {
	s := server.NewMCPServer(
		"claudette",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Tool 1: build_or_update_graph
	s.AddTool(mcp.NewTool("build_or_update_graph",
		mcp.WithDescription("Build or incrementally update the code knowledge graph. Call this first to initialize, or after making changes. Set full_rebuild=true to re-parse every file."),
		mcp.WithBoolean("full_rebuild", mcp.Description("If true, re-parse all files. Default: false (incremental).")),
		mcp.WithString("repo_root", mcp.Description("Repository root path. Auto-detected if omitted.")),
		mcp.WithString("base", mcp.Description("Git ref to diff against for incremental updates. Default: HEAD~1.")),
	), handleBuildOrUpdateGraph)

	// Tool 2: get_impact_radius
	s.AddTool(mcp.NewTool("get_impact_radius",
		mcp.WithDescription("Analyze the blast radius of changed files. Shows which functions, classes, and files are impacted by changes."),
		mcp.WithArray("changed_files", mcp.Description("List of changed file paths. Auto-detected if omitted.")),
		mcp.WithNumber("max_depth", mcp.Description("Number of hops to traverse. Default: 2.")),
		mcp.WithString("repo_root", mcp.Description("Repository root path. Auto-detected if omitted.")),
		mcp.WithString("base", mcp.Description("Git ref for auto-detecting changes. Default: HEAD~1.")),
	), handleGetImpactRadius)

	// Tool 3: query_graph
	s.AddTool(mcp.NewTool("query_graph",
		mcp.WithDescription("Run a predefined graph query. Patterns: callers_of, callees_of, imports_of, importers_of, children_of, tests_for, inheritors_of, file_summary."),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Query pattern name.")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Node name, qualified name, or file path.")),
		mcp.WithString("repo_root", mcp.Description("Repository root path. Auto-detected if omitted.")),
	), handleQueryGraph)

	// Tool 4: get_review_context
	s.AddTool(mcp.NewTool("get_review_context",
		mcp.WithDescription("Generate a focused, token-efficient review context for code changes."),
		mcp.WithArray("changed_files", mcp.Description("Files to review. Auto-detected if omitted.")),
		mcp.WithNumber("max_depth", mcp.Description("Impact radius depth. Default: 2.")),
		mcp.WithBoolean("include_source", mcp.Description("Include source snippets. Default: true.")),
		mcp.WithNumber("max_lines_per_file", mcp.Description("Max source lines per file. Default: 200.")),
		mcp.WithString("repo_root", mcp.Description("Repository root path. Auto-detected if omitted.")),
		mcp.WithString("base", mcp.Description("Git ref for change detection. Default: HEAD~1.")),
	), handleGetReviewContext)

	// Tool 5: semantic_search_nodes
	s.AddTool(mcp.NewTool("semantic_search_nodes",
		mcp.WithDescription("Search for code entities by name or keyword."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search string.")),
		mcp.WithString("kind", mcp.Description("Optional filter: File, Class, Function, Type, or Test.")),
		mcp.WithNumber("limit", mcp.Description("Maximum results. Default: 20.")),
		mcp.WithString("repo_root", mcp.Description("Repository root path. Auto-detected if omitted.")),
	), handleSemanticSearchNodes)

	// Tool 6: embed_graph
	s.AddTool(mcp.NewTool("embed_graph",
		mcp.WithDescription("Compute vector embeddings for semantic search. Not available in Go version."),
		mcp.WithString("repo_root", mcp.Description("Repository root path.")),
	), handleEmbedGraph)

	// Tool 7: list_graph_stats
	s.AddTool(mcp.NewTool("list_graph_stats",
		mcp.WithDescription("Get aggregate statistics about the code knowledge graph."),
		mcp.WithString("repo_root", mcp.Description("Repository root path. Auto-detected if omitted.")),
	), handleListGraphStats)

	// Tool 8: get_docs_section
	s.AddTool(mcp.NewTool("get_docs_section",
		mcp.WithDescription("Get a specific section from the LLM-optimized documentation."),
		mcp.WithString("section_name", mcp.Required(), mcp.Description("The section to retrieve.")),
	), handleGetDocsSection)

	return server.ServeStdio(s)
}

func resultToContent(result map[string]any) *mcp.CallToolResult {
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data))
}

func getArgs(request mcp.CallToolRequest) map[string]any {
	if m, ok := request.Params.Arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func handleBuildOrUpdateGraph(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	fullRebuild, _ := args["full_rebuild"].(bool)
	repoRoot, _ := args["repo_root"].(string)
	base, _ := args["base"].(string)
	result := tools.BuildOrUpdateGraph(fullRebuild, repoRoot, base)
	return resultToContent(result), nil
}

func handleGetImpactRadius(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	repoRoot, _ := args["repo_root"].(string)
	base, _ := args["base"].(string)
	maxDepth := intArg(args, "max_depth", 2)

	var changedFiles []string
	if cf, ok := args["changed_files"].([]interface{}); ok {
		for _, f := range cf {
			if s, ok := f.(string); ok {
				changedFiles = append(changedFiles, s)
			}
		}
	}

	result := tools.GetImpactRadius(changedFiles, maxDepth, repoRoot, base)
	return resultToContent(result), nil
}

func handleQueryGraph(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	pattern, _ := args["pattern"].(string)
	target, _ := args["target"].(string)
	repoRoot, _ := args["repo_root"].(string)
	result := tools.QueryGraph(pattern, target, repoRoot)
	return resultToContent(result), nil
}

func handleGetReviewContext(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	repoRoot, _ := args["repo_root"].(string)
	base, _ := args["base"].(string)
	maxDepth := intArg(args, "max_depth", 2)
	includeSource := true
	if v, ok := args["include_source"].(bool); ok {
		includeSource = v
	}
	maxLines := intArg(args, "max_lines_per_file", 200)

	var changedFiles []string
	if cf, ok := args["changed_files"].([]interface{}); ok {
		for _, f := range cf {
			if s, ok := f.(string); ok {
				changedFiles = append(changedFiles, s)
			}
		}
	}

	result := tools.GetReviewContext(changedFiles, maxDepth, includeSource, maxLines, repoRoot, base)
	return resultToContent(result), nil
}

func handleSemanticSearchNodes(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	query, _ := args["query"].(string)
	kind, _ := args["kind"].(string)
	limit := intArg(args, "limit", 20)
	repoRoot, _ := args["repo_root"].(string)
	result := tools.SemanticSearchNodes(query, kind, limit, repoRoot)
	return resultToContent(result), nil
}

func handleEmbedGraph(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	repoRoot, _ := args["repo_root"].(string)
	result := tools.EmbedGraph(repoRoot)
	return resultToContent(result), nil
}

func handleListGraphStats(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	repoRoot, _ := args["repo_root"].(string)
	result := tools.ListGraphStats(repoRoot)
	return resultToContent(result), nil
}

func handleGetDocsSection(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	sectionName, _ := args["section_name"].(string)
	result := tools.GetDocsSection(sectionName)
	return resultToContent(result), nil
}

func intArg(args map[string]any, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}
