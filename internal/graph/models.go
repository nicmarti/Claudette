// Package graph provides SQLite-backed knowledge graph storage and query engine.
//
// Stores code structure as nodes (File, Class, Function, Type, Test) and
// edges (CALLS, IMPORTS_FROM, INHERITS, IMPLEMENTS, CONTAINS, TESTED_BY, DEPENDS_ON).
package graph

// NodeInfo represents a parsed code entity before insertion into the graph.
type NodeInfo struct {
	Kind       string            `json:"kind"`        // File, Class, Function, Type, Test
	Name       string            `json:"name"`
	FilePath   string            `json:"file_path"`
	LineStart  int               `json:"line_start"`
	LineEnd    int               `json:"line_end"`
	Language   string            `json:"language"`
	ParentName string            `json:"parent_name,omitempty"`
	Params     string            `json:"params,omitempty"`
	ReturnType string            `json:"return_type,omitempty"`
	Modifiers  string            `json:"modifiers,omitempty"`
	IsTest     bool              `json:"is_test"`
	Extra      map[string]string `json:"extra,omitempty"`
}

// EdgeInfo represents a relationship between code entities before insertion.
type EdgeInfo struct {
	Kind     string            `json:"kind"` // CALLS, IMPORTS_FROM, INHERITS, IMPLEMENTS, CONTAINS, TESTED_BY, DEPENDS_ON
	Source   string            `json:"source"`
	Target   string            `json:"target"`
	FilePath string            `json:"file_path"`
	Line     int               `json:"line"`
	Extra    map[string]string `json:"extra,omitempty"`
}

// GraphNode represents a node stored in the graph database.
type GraphNode struct {
	ID            int               `json:"id"`
	Kind          string            `json:"kind"`
	Name          string            `json:"name"`
	QualifiedName string            `json:"qualified_name"`
	FilePath      string            `json:"file_path"`
	LineStart     int               `json:"line_start"`
	LineEnd       int               `json:"line_end"`
	Language      string            `json:"language"`
	ParentName    string            `json:"parent_name,omitempty"`
	Params        string            `json:"params,omitempty"`
	ReturnType    string            `json:"return_type,omitempty"`
	IsTest        bool              `json:"is_test"`
	FileHash      string            `json:"file_hash,omitempty"`
	Extra         map[string]string `json:"extra,omitempty"`
}

// GraphEdge represents an edge stored in the graph database.
type GraphEdge struct {
	ID              int               `json:"id"`
	Kind            string            `json:"kind"`
	SourceQualified string            `json:"source"`
	TargetQualified string            `json:"target"`
	FilePath        string            `json:"file_path"`
	Line            int               `json:"line"`
	Extra           map[string]string `json:"extra,omitempty"`
}

// GraphStats holds aggregate statistics about the graph.
type GraphStats struct {
	TotalNodes  int            `json:"total_nodes"`
	TotalEdges  int            `json:"total_edges"`
	NodesByKind map[string]int `json:"nodes_by_kind"`
	EdgesByKind map[string]int `json:"edges_by_kind"`
	Languages   []string       `json:"languages"`
	FilesCount  int            `json:"files_count"`
	LastUpdated string         `json:"last_updated,omitempty"`
}

// NodeToDict converts a GraphNode to a map for JSON serialization.
func NodeToDict(n *GraphNode) map[string]any {
	return map[string]any{
		"id":             n.ID,
		"kind":           n.Kind,
		"name":           n.Name,
		"qualified_name": n.QualifiedName,
		"file_path":      n.FilePath,
		"line_start":     n.LineStart,
		"line_end":       n.LineEnd,
		"language":       n.Language,
		"parent_name":    n.ParentName,
		"is_test":        n.IsTest,
	}
}

// EdgeToDict converts a GraphEdge to a map for JSON serialization.
func EdgeToDict(e *GraphEdge) map[string]any {
	return map[string]any{
		"id":        e.ID,
		"kind":      e.Kind,
		"source":    e.SourceQualified,
		"target":    e.TargetQualified,
		"file_path": e.FilePath,
		"line":      e.Line,
	}
}

// CompactNodeDict returns a minimal representation of a node for size-constrained responses.
func CompactNodeDict(n *GraphNode) map[string]any {
	return map[string]any{
		"kind":      n.Kind,
		"name":      n.Name,
		"file_path": n.FilePath,
		"line":      n.LineStart,
	}
}

// CompactEdgeDict returns a minimal representation of an edge for size-constrained responses.
func CompactEdgeDict(e *GraphEdge) map[string]any {
	return map[string]any{
		"kind":   e.Kind,
		"source": e.SourceQualified,
		"target": e.TargetQualified,
	}
}
