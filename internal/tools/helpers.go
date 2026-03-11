package tools

import (
	"fmt"
	"sort"
	"strings"

	"claudette/internal/graph"
)

// extractRelevantLines extracts only the lines relevant to changed nodes.
func extractRelevantLines(lines []string, nodes []*graph.GraphNode, filePath string) string {
	type lineRange struct{ start, end int }
	var ranges []lineRange

	for _, n := range nodes {
		if n.FilePath == filePath {
			start := n.LineStart - 3
			if start < 0 {
				start = 0
			}
			end := n.LineEnd + 2
			if end > len(lines) {
				end = len(lines)
			}
			ranges = append(ranges, lineRange{start, end})
		}
	}

	if len(ranges) == 0 {
		// Show first 50 lines as fallback
		limit := 50
		if limit > len(lines) {
			limit = len(lines)
		}
		var parts []string
		for i := 0; i < limit; i++ {
			parts = append(parts, fmt.Sprintf("%d: %s", i+1, lines[i]))
		}
		return strings.Join(parts, "\n")
	}

	// Sort and merge overlapping ranges
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	merged := []lineRange{ranges[0]}
	for _, r := range ranges[1:] {
		last := &merged[len(merged)-1]
		if r.start <= last.end+1 {
			if r.end > last.end {
				last.end = r.end
			}
		} else {
			merged = append(merged, r)
		}
	}

	var parts []string
	for _, r := range merged {
		if len(parts) > 0 {
			parts = append(parts, "...")
		}
		for i := r.start; i < r.end && i < len(lines); i++ {
			parts = append(parts, fmt.Sprintf("%d: %s", i+1, lines[i]))
		}
	}
	return strings.Join(parts, "\n")
}

// generateReviewGuidance generates review guidance based on impact analysis.
func generateReviewGuidance(impact *graph.ImpactResult, changedFiles []string) string {
	var parts []string

	// Check for test coverage
	var changedFuncs []*graph.GraphNode
	for _, n := range impact.ChangedNodes {
		if n.Kind == "Function" {
			changedFuncs = append(changedFuncs, n)
		}
	}
	testedFuncs := make(map[string]bool)
	for _, e := range impact.Edges {
		if e.Kind == "TESTED_BY" {
			testedFuncs[e.SourceQualified] = true
		}
	}

	var untested []string
	for _, f := range changedFuncs {
		if !testedFuncs[f.QualifiedName] && !f.IsTest {
			untested = append(untested, f.Name)
		}
	}
	if len(untested) > 0 {
		names := untested
		if len(names) > 5 {
			names = names[:5]
		}
		parts = append(parts, fmt.Sprintf("- %d changed function(s) lack test coverage: %s",
			len(untested), strings.Join(names, ", ")))
	}

	// Check for wide blast radius
	if len(impact.ImpactedNodes) > 20 {
		parts = append(parts, fmt.Sprintf("- Wide blast radius: %d nodes impacted. Review callers and dependents carefully.",
			len(impact.ImpactedNodes)))
	}

	// Check for inheritance changes
	var inheritanceCount int
	for _, e := range impact.Edges {
		if e.Kind == "INHERITS" || e.Kind == "IMPLEMENTS" {
			inheritanceCount++
		}
	}
	if inheritanceCount > 0 {
		parts = append(parts, fmt.Sprintf("- %d inheritance/implementation relationship(s) affected. Check for Liskov substitution violations.",
			inheritanceCount))
	}

	// Check for cross-file impact
	if len(impact.ImpactedFiles) > 3 {
		parts = append(parts, fmt.Sprintf("- Changes impact %d other files. Consider splitting into smaller PRs.",
			len(impact.ImpactedFiles)))
	}

	if len(parts) == 0 {
		parts = append(parts, "- Changes appear well-contained with minimal blast radius.")
	}

	return strings.Join(parts, "\n")
}
