package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"claudette/internal/graph"
	"claudette/internal/incremental"
)

func getStore(repoRoot string) (*graph.GraphStore, string, error) {
	if repoRoot == "" {
		repoRoot = incremental.FindProjectRoot("")
	}
	dbPath := incremental.GetDBPath(repoRoot)
	store, err := graph.NewGraphStore(dbPath)
	if err != nil {
		return nil, "", err
	}
	return store, repoRoot, nil
}

// BuildOrUpdateGraph builds or incrementally updates the code knowledge graph.
func BuildOrUpdateGraph(fullRebuild bool, repoRoot, base string) map[string]any {
	if base == "" {
		base = "HEAD~1"
	}
	store, root, err := getStore(repoRoot)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	defer store.Close()

	if fullRebuild {
		result := incremental.FullBuild(root, store)
		return map[string]any{
			"status":       "ok",
			"build_type":   "full",
			"summary":      fmt.Sprintf("Full build complete: parsed %d files, created %d nodes and %d edges.", result.FilesParsed, result.TotalNodes, result.TotalEdges),
			"files_parsed": result.FilesParsed,
			"total_nodes":  result.TotalNodes,
			"total_edges":  result.TotalEdges,
			"errors":       result.Errors,
		}
	}

	result := incremental.IncrementalUpdate(root, store, base, nil)
	if result.FilesUpdated == 0 {
		return map[string]any{
			"status":     "ok",
			"build_type": "incremental",
			"summary":    "No changes detected. Graph is up to date.",
		}
	}
	return map[string]any{
		"status":          "ok",
		"build_type":      "incremental",
		"summary":         fmt.Sprintf("Incremental update: %d files re-parsed, %d nodes and %d edges updated.", result.FilesUpdated, result.TotalNodes, result.TotalEdges),
		"files_updated":   result.FilesUpdated,
		"total_nodes":     result.TotalNodes,
		"total_edges":     result.TotalEdges,
		"changed_files":   result.ChangedFiles,
		"dependent_files": result.DependentFiles,
		"errors":          result.Errors,
	}
}

// GetImpactRadius analyzes the blast radius of changed files.
func GetImpactRadius(changedFiles []string, maxDepth int, repoRoot, base string) map[string]any {
	if base == "" {
		base = "HEAD~1"
	}
	if maxDepth == 0 {
		maxDepth = 2
	}
	store, root, err := getStore(repoRoot)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	defer store.Close()

	if changedFiles == nil {
		changedFiles = incremental.GetChangedFiles(root, base)
		if len(changedFiles) == 0 {
			changedFiles = incremental.GetStagedAndUnstaged(root)
		}
	}

	if len(changedFiles) == 0 {
		return map[string]any{
			"status":         "ok",
			"summary":        "No changed files detected.",
			"changed_nodes":  []any{},
			"impacted_nodes": []any{},
			"impacted_files": []any{},
		}
	}

	absFiles := make([]string, len(changedFiles))
	for i, f := range changedFiles {
		absFiles[i] = filepath.Join(root, f)
	}
	result := store.GetImpactRadius(absFiles, maxDepth)

	changedDicts := make([]any, 0, len(result.ChangedNodes))
	for _, n := range result.ChangedNodes {
		changedDicts = append(changedDicts, graph.NodeToDict(n))
	}
	impactedDicts := make([]any, 0, len(result.ImpactedNodes))
	for _, n := range result.ImpactedNodes {
		impactedDicts = append(impactedDicts, graph.NodeToDict(n))
	}
	edgeDicts := make([]any, 0, len(result.Edges))
	for _, e := range result.Edges {
		edgeDicts = append(edgeDicts, graph.EdgeToDict(e))
	}

	summary := fmt.Sprintf("Blast radius for %d changed file(s):\n  - %d nodes directly changed\n  - %d nodes impacted (within %d hops)\n  - %d additional files affected",
		len(changedFiles), len(changedDicts), len(impactedDicts), maxDepth, len(result.ImpactedFiles))

	return map[string]any{
		"status":         "ok",
		"summary":        summary,
		"changed_files":  changedFiles,
		"changed_nodes":  changedDicts,
		"impacted_nodes": impactedDicts,
		"impacted_files": result.ImpactedFiles,
		"edges":          edgeDicts,
	}
}

// QueryPatterns lists available graph query patterns.
var QueryPatterns = map[string]string{
	"callers_of":    "Find all functions that call a given function",
	"callees_of":    "Find all functions called by a given function",
	"imports_of":    "Find all imports of a given file or module",
	"importers_of":  "Find all files that import a given file or module",
	"children_of":   "Find all nodes contained in a file or class",
	"tests_for":     "Find all tests for a given function or class",
	"inheritors_of": "Find all classes that inherit from a given class",
	"file_summary":  "Get a summary of all nodes in a file",
}

// QueryGraph runs a predefined graph query.
func QueryGraph(pattern, target, repoRoot string) map[string]any {
	store, root, err := getStore(repoRoot)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	defer store.Close()

	if _, ok := QueryPatterns[pattern]; !ok {
		keys := make([]string, 0, len(QueryPatterns))
		for k := range QueryPatterns {
			keys = append(keys, k)
		}
		return map[string]any{
			"status": "error",
			"error":  fmt.Sprintf("Unknown pattern '%s'. Available: %v", pattern, keys),
		}
	}

	// Resolve target
	node := store.GetNode(target)
	if node == nil {
		absTarget := filepath.Join(root, target)
		node = store.GetNode(absTarget)
	}
	if node == nil {
		candidates := store.SearchNodes(target, 5)
		if len(candidates) == 1 {
			node = candidates[0]
			target = node.QualifiedName
		} else if len(candidates) > 1 {
			cDicts := make([]any, 0, len(candidates))
			for _, c := range candidates {
				cDicts = append(cDicts, graph.NodeToDict(c))
			}
			return map[string]any{
				"status":     "ambiguous",
				"summary":    fmt.Sprintf("Multiple matches for '%s'. Please use a qualified name.", target),
				"candidates": cDicts,
			}
		}
	}

	if node == nil && pattern != "file_summary" {
		return map[string]any{
			"status":  "not_found",
			"summary": fmt.Sprintf("No node found matching '%s'.", target),
		}
	}

	qn := target
	if node != nil {
		qn = node.QualifiedName
	}

	var results []any
	var edgesOut []any

	switch pattern {
	case "callers_of":
		for _, e := range store.GetEdgesByTarget(qn) {
			if e.Kind == "CALLS" {
				if caller := store.GetNode(e.SourceQualified); caller != nil {
					results = append(results, graph.NodeToDict(caller))
				}
				edgesOut = append(edgesOut, graph.EdgeToDict(e))
			}
		}
	case "callees_of":
		for _, e := range store.GetEdgesBySource(qn) {
			if e.Kind == "CALLS" {
				if callee := store.GetNode(e.TargetQualified); callee != nil {
					results = append(results, graph.NodeToDict(callee))
				}
				edgesOut = append(edgesOut, graph.EdgeToDict(e))
			}
		}
	case "imports_of":
		for _, e := range store.GetEdgesBySource(qn) {
			if e.Kind == "IMPORTS_FROM" {
				results = append(results, map[string]any{"import_target": e.TargetQualified})
				edgesOut = append(edgesOut, graph.EdgeToDict(e))
			}
		}
	case "importers_of":
		absTarget := filepath.Join(root, target)
		if node != nil {
			absTarget = node.FilePath
		}
		for _, e := range store.GetEdgesByTarget(absTarget) {
			if e.Kind == "IMPORTS_FROM" {
				results = append(results, map[string]any{"importer": e.SourceQualified, "file": e.FilePath})
				edgesOut = append(edgesOut, graph.EdgeToDict(e))
			}
		}
	case "children_of":
		for _, e := range store.GetEdgesBySource(qn) {
			if e.Kind == "CONTAINS" {
				if child := store.GetNode(e.TargetQualified); child != nil {
					results = append(results, graph.NodeToDict(child))
				}
			}
		}
	case "tests_for":
		for _, e := range store.GetEdgesByTarget(qn) {
			if e.Kind == "TESTED_BY" {
				if test := store.GetNode(e.SourceQualified); test != nil {
					results = append(results, graph.NodeToDict(test))
				}
			}
		}
		// Also search by naming convention
		name := target
		if node != nil {
			name = node.Name
		}
		seen := make(map[string]bool)
		for _, r := range results {
			if m, ok := r.(map[string]any); ok {
				if qn, ok := m["qualified_name"].(string); ok {
					seen[qn] = true
				}
			}
		}
		for _, t := range store.SearchNodes("test_"+name, 10) {
			if !seen[t.QualifiedName] && t.IsTest {
				results = append(results, graph.NodeToDict(t))
			}
		}
		for _, t := range store.SearchNodes("Test"+name, 10) {
			if !seen[t.QualifiedName] && t.IsTest {
				results = append(results, graph.NodeToDict(t))
			}
		}
	case "inheritors_of":
		for _, e := range store.GetEdgesByTarget(qn) {
			if e.Kind == "INHERITS" || e.Kind == "IMPLEMENTS" {
				if child := store.GetNode(e.SourceQualified); child != nil {
					results = append(results, graph.NodeToDict(child))
				}
				edgesOut = append(edgesOut, graph.EdgeToDict(e))
			}
		}
	case "file_summary":
		absPath := filepath.Join(root, target)
		for _, n := range store.GetNodesByFile(absPath) {
			results = append(results, graph.NodeToDict(n))
		}
	}

	if results == nil {
		results = []any{}
	}
	if edgesOut == nil {
		edgesOut = []any{}
	}

	return map[string]any{
		"status":      "ok",
		"pattern":     pattern,
		"target":      target,
		"description": QueryPatterns[pattern],
		"summary":     fmt.Sprintf("Found %d result(s) for %s('%s')", len(results), pattern, target),
		"results":     results,
		"edges":       edgesOut,
	}
}

// Output size limits for MCP responses.
// File lists (paths only) are NEVER truncated so Claude can find any file.
// Only detailed data (nodes, edges, snippets) is capped.
const (
	largeChangesetThreshold = 30   // Warn when more changed files than this
	maxDetailedNodes        = 20   // Max nodes with full details
	maxDetailedEdges        = 30   // Max edges with full details
	maxSourceFiles          = 10   // Max files to include source snippets for
	maxSnippetBytes         = 2000 // Max bytes per source snippet
	maxTotalSourceLen       = 10000 // ~10KB total for all snippets
)

// GetReviewContext generates a focused review context from changed files.
func GetReviewContext(changedFiles []string, maxDepth int, includeSource bool, maxLinesPerFile int, repoRoot, base string) map[string]any {
	if base == "" {
		base = "HEAD~1"
	}
	if maxDepth == 0 {
		maxDepth = 2
	}
	if maxLinesPerFile == 0 {
		maxLinesPerFile = 200
	}
	store, root, err := getStore(repoRoot)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	defer store.Close()

	if changedFiles == nil {
		changedFiles = incremental.GetChangedFiles(root, base)
		if len(changedFiles) == 0 {
			changedFiles = incremental.GetStagedAndUnstaged(root)
		}
	}

	if len(changedFiles) == 0 {
		return map[string]any{
			"status":  "ok",
			"summary": "No changes detected. Nothing to review.",
			"context": map[string]any{},
		}
	}

	// Build absolute paths for ALL changed files (never truncated)
	absChangedFiles := make([]string, len(changedFiles))
	for i, f := range changedFiles {
		absChangedFiles[i] = filepath.Join(root, f)
	}

	impact := store.GetImpactRadius(absChangedFiles, maxDepth)

	// Keep impacted files as relative paths to save space.
	// The repo_root is provided in the response so Claude can resolve full paths.
	relImpactedFiles := make([]string, len(impact.ImpactedFiles))
	for i, f := range impact.ImpactedFiles {
		if filepath.IsAbs(f) {
			rel, err := filepath.Rel(root, f)
			if err == nil {
				relImpactedFiles[i] = rel
			} else {
				relImpactedFiles[i] = f
			}
		} else {
			relImpactedFiles[i] = f
		}
	}

	// Detailed nodes/edges use compact format and are capped to control response size
	changedDicts := make([]any, 0, len(impact.ChangedNodes))
	for i, n := range impact.ChangedNodes {
		if i >= maxDetailedNodes {
			break
		}
		changedDicts = append(changedDicts, graph.CompactNodeDict(n))
	}
	impactedDicts := make([]any, 0, len(impact.ImpactedNodes))
	for i, n := range impact.ImpactedNodes {
		if i >= maxDetailedNodes {
			break
		}
		impactedDicts = append(impactedDicts, graph.CompactNodeDict(n))
	}
	edgeDicts := make([]any, 0, len(impact.Edges))
	for i, e := range impact.Edges {
		if i >= maxDetailedEdges {
			break
		}
		edgeDicts = append(edgeDicts, graph.CompactEdgeDict(e))
	}

	context := map[string]any{
		"repo_root":      root,
		"changed_files":  absChangedFiles,
		"impacted_files": relImpactedFiles,
		"graph": map[string]any{
			"changed_nodes":  changedDicts,
			"impacted_nodes": impactedDicts,
			"edges":          edgeDicts,
		},
	}

	if includeSource {
		snippets := make(map[string]string)
		totalSourceLen := 0
		sourceFiles := absChangedFiles
		if len(sourceFiles) > maxSourceFiles {
			sourceFiles = sourceFiles[:maxSourceFiles]
		}
		for _, fullPath := range sourceFiles {
			relPath, _ := filepath.Rel(root, fullPath)
			if totalSourceLen >= maxTotalSourceLen {
				snippets[relPath] = "(skipped: output size limit reached)"
				continue
			}
			data, err := os.ReadFile(fullPath)
			if err != nil {
				snippets[relPath] = "(could not read file)"
				continue
			}
			lines := strings.Split(string(data), "\n")
			var snippet string
			if len(lines) > maxLinesPerFile {
				snippet = extractRelevantLines(lines, impact.ChangedNodes, fullPath)
			} else {
				var numbered []string
				for i, line := range lines {
					numbered = append(numbered, fmt.Sprintf("%d: %s", i+1, line))
				}
				snippet = strings.Join(numbered, "\n")
			}
			if len(snippet) > maxSnippetBytes {
				snippet = snippet[:maxSnippetBytes] + "\n... (truncated)"
			}
			snippets[relPath] = snippet
			totalSourceLen += len(snippet)
		}
		context["source_snippets"] = snippets
	}

	guidance := generateReviewGuidance(impact, changedFiles)
	context["review_guidance"] = guidance

	// Build summary
	summary := fmt.Sprintf("Review context for %d changed file(s):\n  - %d changed files (absolute paths provided)\n  - %d directly changed nodes (showing %d)\n  - %d impacted files (absolute paths provided)\n  - %d impacted nodes (showing %d)\n  - %d edges (showing %d)",
		len(changedFiles),
		len(absChangedFiles),
		len(impact.ChangedNodes), len(changedDicts),
		len(relImpactedFiles),
		len(impact.ImpactedNodes), len(impactedDicts),
		len(impact.Edges), len(edgeDicts))

	// Warn on large changesets
	if len(changedFiles) > largeChangesetThreshold {
		summary += fmt.Sprintf("\n\n  WARNING: Large changeset (%d files). Consider splitting into smaller PRs for more effective reviews.", len(changedFiles))
	}

	summary += fmt.Sprintf("\n\nReview guidance:\n%s", guidance)

	return map[string]any{
		"status":  "ok",
		"summary": summary,
		"context": context,
	}
}

// SemanticSearchNodes searches for nodes by name/keyword.
func SemanticSearchNodes(query, kind string, limit int, repoRoot string) map[string]any {
	if limit == 0 {
		limit = 20
	}
	store, _, err := getStore(repoRoot)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	defer store.Close()

	results := store.SearchNodes(query, limit*2)

	if kind != "" {
		filtered := make([]*graph.GraphNode, 0)
		for _, r := range results {
			if r.Kind == kind {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// Score and sort
	qLower := strings.ToLower(query)
	sort.Slice(results, func(i, j int) bool {
		ni := strings.ToLower(results[i].Name)
		nj := strings.ToLower(results[j].Name)
		si, sj := 2, 2
		if ni == qLower {
			si = 0
		} else if strings.HasPrefix(ni, qLower) {
			si = 1
		}
		if nj == qLower {
			sj = 0
		} else if strings.HasPrefix(nj, qLower) {
			sj = 1
		}
		return si < sj
	})

	if len(results) > limit {
		results = results[:limit]
	}

	dicts := make([]any, 0, len(results))
	for _, r := range results {
		dicts = append(dicts, graph.NodeToDict(r))
	}

	kindSuffix := ""
	if kind != "" {
		kindSuffix = fmt.Sprintf(" (kind=%s)", kind)
	}

	return map[string]any{
		"status":      "ok",
		"query":       query,
		"search_mode": "keyword",
		"summary":     fmt.Sprintf("Found %d node(s) matching '%s'%s", len(dicts), query, kindSuffix),
		"results":     dicts,
	}
}

// ListGraphStats returns aggregate statistics about the knowledge graph.
func ListGraphStats(repoRoot string) map[string]any {
	store, root, err := getStore(repoRoot)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	defer store.Close()

	stats := store.GetStats()
	rootName := filepath.Base(root)

	langs := "none"
	if len(stats.Languages) > 0 {
		langs = strings.Join(stats.Languages, ", ")
	}
	lastUpdated := "never"
	if stats.LastUpdated != "" {
		lastUpdated = stats.LastUpdated
	}

	summary := fmt.Sprintf("Graph statistics for %s:\n  Files: %d\n  Total nodes: %d\n  Total edges: %d\n  Languages: %s\n  Last updated: %s",
		rootName, stats.FilesCount, stats.TotalNodes, stats.TotalEdges, langs, lastUpdated)

	// Nodes by kind
	summary += "\n\nNodes by kind:"
	kinds := make([]string, 0, len(stats.NodesByKind))
	for k := range stats.NodesByKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		summary += fmt.Sprintf("\n  %s: %d", k, stats.NodesByKind[k])
	}

	summary += "\n\nEdges by kind:"
	ekinds := make([]string, 0, len(stats.EdgesByKind))
	for k := range stats.EdgesByKind {
		ekinds = append(ekinds, k)
	}
	sort.Strings(ekinds)
	for _, k := range ekinds {
		summary += fmt.Sprintf("\n  %s: %d", k, stats.EdgesByKind[k])
	}

	return map[string]any{
		"status":        "ok",
		"summary":       summary,
		"total_nodes":   stats.TotalNodes,
		"total_edges":   stats.TotalEdges,
		"nodes_by_kind": stats.NodesByKind,
		"edges_by_kind": stats.EdgesByKind,
		"languages":     stats.Languages,
		"files_count":   stats.FilesCount,
		"last_updated":  stats.LastUpdated,
	}
}

// EmbedGraph is a stub - ML embeddings not available in Go version.
func EmbedGraph(repoRoot string) map[string]any {
	return map[string]any{
		"status": "error",
		"error":  "ML embeddings are not available in the Go version. Keyword search is used instead.",
	}
}

// GetDocsSection returns a specific section from the LLM-optimized reference.
func GetDocsSection(sectionName string) map[string]any {
	referencePaths := []string{
		"docs/LLM-OPTIMIZED-REFERENCE.md",
	}

	_, root, _ := getStore("")

	for _, relPath := range referencePaths {
		fullPath := filepath.Join(root, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(data)
		re := regexp.MustCompile(`(?is)<section name="` + regexp.QuoteMeta(sectionName) + `">(.*?)</section>`)
		match := re.FindStringSubmatch(content)
		if len(match) > 1 {
			return map[string]any{
				"status":  "ok",
				"section": sectionName,
				"content": strings.TrimSpace(match[1]),
			}
		}
	}

	available := []string{
		"usage", "review-delta", "review-pr", "commands",
		"legal", "watch", "embeddings", "languages", "troubleshooting",
	}
	return map[string]any{
		"status": "not_found",
		"error":  fmt.Sprintf("Section '%s' not found. Available: %s", sectionName, strings.Join(available, ", ")),
	}
}
