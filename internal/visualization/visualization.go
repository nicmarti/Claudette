// Package visualization generates an interactive D3.js HTML graph visualization.
package visualization

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"claudette/internal/graph"
)

//go:embed template.html
var htmlTemplate string

// ExportGraphData exports all graph nodes and edges as JSON-serializable maps.
func ExportGraphData(store *graph.GraphStore) map[string]any {
	var nodes []map[string]any
	seenQN := make(map[string]bool)

	for _, filePath := range store.GetAllFiles() {
		for _, gnode := range store.GetNodesByFile(filePath) {
			if seenQN[gnode.QualifiedName] {
				continue
			}
			seenQN[gnode.QualifiedName] = true
			d := graph.NodeToDict(gnode)
			d["params"] = gnode.Params
			d["return_type"] = gnode.ReturnType
			nodes = append(nodes, d)
		}
	}

	// Build name index for resolving unqualified edge targets
	nameIndex := buildNameIndex(nodes, seenQN)

	allEdges := store.GetAllEdges()
	var edges []map[string]any
	for _, e := range allEdges {
		ed := graph.EdgeToDict(e)
		src := resolveTarget(ed["source"].(string), ed["source"].(string), seenQN, nameIndex)
		tgt := resolveTarget(ed["target"].(string), ed["source"].(string), seenQN, nameIndex)
		if src != "" && tgt != "" {
			ed["source"] = src
			ed["target"] = tgt
			edges = append(edges, ed)
		}
	}

	stats := store.GetStats()

	return map[string]any{
		"nodes": nodes,
		"edges": edges,
		"stats": map[string]any{
			"total_nodes":   stats.TotalNodes,
			"total_edges":   stats.TotalEdges,
			"nodes_by_kind": stats.NodesByKind,
			"edges_by_kind": stats.EdgesByKind,
			"languages":     stats.Languages,
			"files_count":   stats.FilesCount,
			"last_updated":  stats.LastUpdated,
		},
	}
}

// GenerateHTML generates a self-contained interactive HTML visualization.
func GenerateHTML(store *graph.GraphStore, outputPath string) error {
	data := ExportGraphData(store)
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	html := strings.Replace(htmlTemplate, "__GRAPH_DATA__", string(dataJSON), 1)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(html), 0o644)
}

func buildNameIndex(nodes []map[string]any, seenQN map[string]bool) map[string][]string {
	index := make(map[string][]string)

	add := func(key, qn string) {
		index[key] = append(index[key], qn)
	}

	for _, n := range nodes {
		qn := n["qualified_name"].(string)
		name := n["name"].(string)
		add(name, qn)

		if strings.Contains(qn, "::") {
			parts := strings.Split(qn, "/")
			add(parts[len(parts)-1], qn)
		}

		fp, _ := n["file_path"].(string)
		if fp != "" {
			mod := strings.ReplaceAll(fp, "/", ".")
			mod = strings.TrimSuffix(mod, ".py")
			kind, _ := n["kind"].(string)
			if kind == "File" {
				add(mod, qn)
			} else {
				add(mod+"."+name, qn)
			}
		}
	}
	return index
}

func resolveTarget(target, source string, seenQN map[string]bool, nameIndex map[string][]string) string {
	if seenQN[target] {
		return target
	}

	candidates := nameIndex[target]
	if len(candidates) == 0 {
		return ""
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	srcFile := source
	if idx := strings.Index(source, "::"); idx >= 0 {
		srcFile = source[:idx]
	}
	var sameFile []string
	for _, c := range candidates {
		if strings.HasPrefix(c, srcFile) {
			sameFile = append(sameFile, c)
		}
	}
	if len(sameFile) == 1 {
		return sameFile[0]
	}

	return candidates[0]
}
