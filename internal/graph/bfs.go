package graph

// ImpactResult holds the result of a BFS impact radius analysis.
type ImpactResult struct {
	ChangedNodes  []*GraphNode
	ImpactedNodes []*GraphNode
	ImpactedFiles []string
	Edges         []*GraphEdge
}

// GetImpactRadius performs BFS from changed files to find all impacted nodes within maxDepth hops.
func (s *GraphStore) GetImpactRadius(changedFiles []string, maxDepth int) *ImpactResult {
	// Build adjacency lists from all edges
	forward := make(map[string][]string)  // source -> targets
	reverse := make(map[string][]string)  // target -> sources

	allEdges := s.GetAllEdges()
	for _, e := range allEdges {
		forward[e.SourceQualified] = append(forward[e.SourceQualified], e.TargetQualified)
		reverse[e.TargetQualified] = append(reverse[e.TargetQualified], e.SourceQualified)
	}

	// Seed: all qualified names in changed files
	seeds := make(map[string]bool)
	for _, f := range changedFiles {
		nodes := s.GetNodesByFile(f)
		for _, n := range nodes {
			seeds[n.QualifiedName] = true
		}
	}

	// BFS outward through all edge types
	visited := make(map[string]bool)
	frontier := make(map[string]bool)
	for qn := range seeds {
		frontier[qn] = true
	}
	impacted := make(map[string]bool)

	for depth := 0; len(frontier) > 0 && depth < maxDepth; depth++ {
		nextFrontier := make(map[string]bool)
		for qn := range frontier {
			visited[qn] = true
			// Forward edges
			for _, neighbor := range forward[qn] {
				if !visited[neighbor] {
					nextFrontier[neighbor] = true
					impacted[neighbor] = true
				}
			}
			// Reverse edges
			for _, pred := range reverse[qn] {
				if !visited[pred] {
					nextFrontier[pred] = true
					impacted[pred] = true
				}
			}
		}
		frontier = nextFrontier
	}

	// Resolve to full node info
	var changedNodes []*GraphNode
	for qn := range seeds {
		if node := s.GetNode(qn); node != nil {
			changedNodes = append(changedNodes, node)
		}
	}

	var impactedNodes []*GraphNode
	impactedFilesSet := make(map[string]bool)
	for qn := range impacted {
		if seeds[qn] {
			continue
		}
		if node := s.GetNode(qn); node != nil {
			impactedNodes = append(impactedNodes, node)
			impactedFilesSet[node.FilePath] = true
		}
	}

	var impactedFiles []string
	for f := range impactedFilesSet {
		impactedFiles = append(impactedFiles, f)
	}

	// Collect relevant edges
	allQNs := make(map[string]bool)
	for qn := range seeds {
		allQNs[qn] = true
	}
	for qn := range impacted {
		allQNs[qn] = true
	}
	relevantEdges := s.GetEdgesAmong(allQNs)

	return &ImpactResult{
		ChangedNodes:  changedNodes,
		ImpactedNodes: impactedNodes,
		ImpactedFiles: impactedFiles,
		Edges:         relevantEdges,
	}
}

// GetSubgraph extracts nodes and their connecting edges.
func (s *GraphStore) GetSubgraph(qualifiedNames []string) ([]*GraphNode, []*GraphEdge) {
	var nodes []*GraphNode
	qnSet := make(map[string]bool)
	for _, qn := range qualifiedNames {
		qnSet[qn] = true
		if node := s.GetNode(qn); node != nil {
			nodes = append(nodes, node)
		}
	}

	var edges []*GraphEdge
	for _, qn := range qualifiedNames {
		for _, e := range s.GetEdgesBySource(qn) {
			if qnSet[e.TargetQualified] {
				edges = append(edges, e)
			}
		}
	}

	return nodes, edges
}
