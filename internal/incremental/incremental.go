package incremental

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"claudette/internal/graph"
	"claudette/internal/parser"
)

// CollectAllFiles collects all parseable files in the repo, respecting ignore patterns.
func CollectAllFiles(repoRoot string) []string {
	ignorePatterns := LoadIgnorePatterns(repoRoot)
	var files []string

	// Prefer git ls-files
	tracked := GetAllTrackedFiles(repoRoot)
	var candidates []string
	if len(tracked) > 0 {
		candidates = tracked
	} else {
		// Fallback: walk directory
		filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				rel, _ := filepath.Rel(repoRoot, path)
				candidates = append(candidates, rel)
			}
			return nil
		})
	}

	for _, relPath := range candidates {
		if ShouldIgnore(relPath, ignorePatterns) {
			continue
		}
		fullPath := filepath.Join(repoRoot, relPath)
		if parser.DetectLanguage(fullPath) == "" {
			continue
		}
		if IsBinary(fullPath) {
			continue
		}
		files = append(files, relPath)
	}
	return files
}

// FindDependents finds files that import from or depend on the given file.
func FindDependents(store *graph.GraphStore, filePath string) []string {
	dependents := make(map[string]bool)

	// Edges where someone imports from this file
	edges := store.GetEdgesByTarget(filePath)
	for _, e := range edges {
		if e.Kind == "IMPORTS_FROM" {
			dependents[e.FilePath] = true
		}
	}

	// CALLS, IMPORTS_FROM, INHERITS, IMPLEMENTS edges to nodes in this file
	nodes := store.GetNodesByFile(filePath)
	for _, node := range nodes {
		for _, e := range store.GetEdgesByTarget(node.QualifiedName) {
			switch e.Kind {
			case "CALLS", "IMPORTS_FROM", "INHERITS", "IMPLEMENTS":
				dependents[e.FilePath] = true
			}
		}
	}

	delete(dependents, filePath)
	result := make([]string, 0, len(dependents))
	for f := range dependents {
		result = append(result, f)
	}
	return result
}

// BuildResult holds the result of a full or incremental build.
type BuildResult struct {
	FilesParsed    int               `json:"files_parsed,omitempty"`
	FilesUpdated   int               `json:"files_updated,omitempty"`
	TotalNodes     int               `json:"total_nodes"`
	TotalEdges     int               `json:"total_edges"`
	ChangedFiles   []string          `json:"changed_files,omitempty"`
	DependentFiles []string          `json:"dependent_files,omitempty"`
	Errors         []map[string]string `json:"errors,omitempty"`
}

// FullBuild performs a full rebuild of the entire graph.
func FullBuild(repoRoot string, store *graph.GraphStore) *BuildResult {
	p := parser.NewCodeParser()
	files := CollectAllFiles(repoRoot)

	// Purge stale data
	existingFiles := make(map[string]bool)
	for _, f := range store.GetAllFiles() {
		existingFiles[f] = true
	}
	currentAbs := make(map[string]bool)
	for _, f := range files {
		abs, _ := filepath.Abs(filepath.Join(repoRoot, f))
		currentAbs[abs] = true
	}
	for stale := range existingFiles {
		if !currentAbs[stale] {
			store.RemoveFileData(stale)
		}
	}

	totalNodes := 0
	totalEdges := 0
	var errors []map[string]string
	fileCount := len(files)

	for i, relPath := range files {
		fullPath := filepath.Join(repoRoot, relPath)
		fhash, err := FileHash(fullPath)
		if err != nil {
			errors = append(errors, map[string]string{"file": relPath, "error": err.Error()})
			continue
		}

		nodes, edges, err := p.ParseFile(fullPath)
		if err != nil {
			errors = append(errors, map[string]string{"file": relPath, "error": err.Error()})
			continue
		}

		if err := store.StoreFileNodesEdges(fullPath, nodes, edges, fhash); err != nil {
			errors = append(errors, map[string]string{"file": relPath, "error": err.Error()})
			continue
		}
		totalNodes += len(nodes)
		totalEdges += len(edges)

		if (i+1)%50 == 0 || i+1 == fileCount {
			log.Printf("Progress: %d/%d files parsed", i+1, fileCount)
		}
	}

	now := time.Now().Format("2006-01-02T15:04:05")
	store.SetMetadata("last_updated", now)
	store.SetMetadata("last_build_type", "full")

	return &BuildResult{
		FilesParsed: len(files),
		TotalNodes:  totalNodes,
		TotalEdges:  totalEdges,
		Errors:      errors,
	}
}

// IncrementalUpdate re-parses changed + dependent files only.
func IncrementalUpdate(repoRoot string, store *graph.GraphStore, base string, changedFiles []string) *BuildResult {
	p := parser.NewCodeParser()
	ignorePatterns := LoadIgnorePatterns(repoRoot)

	if changedFiles == nil {
		changedFiles = GetChangedFiles(repoRoot, base)
	}

	if len(changedFiles) == 0 {
		return &BuildResult{
			FilesUpdated:   0,
			ChangedFiles:   []string{},
			DependentFiles: []string{},
		}
	}

	// Find dependent files
	dependentFiles := make(map[string]bool)
	for _, relPath := range changedFiles {
		fullPath := filepath.Join(repoRoot, relPath)
		deps := FindDependents(store, fullPath)
		for _, d := range deps {
			rel, err := filepath.Rel(repoRoot, d)
			if err != nil {
				dependentFiles[d] = true
			} else {
				dependentFiles[rel] = true
			}
		}
	}

	// Combine changed + dependent
	allFiles := make(map[string]bool)
	for _, f := range changedFiles {
		allFiles[f] = true
	}
	for f := range dependentFiles {
		allFiles[f] = true
	}

	totalNodes := 0
	totalEdges := 0
	var errors []map[string]string

	for relPath := range allFiles {
		if ShouldIgnore(relPath, ignorePatterns) {
			continue
		}
		absPath := filepath.Join(repoRoot, relPath)

		// Check if file exists
		if _, err := filepath.Abs(absPath); err != nil {
			store.RemoveFileData(absPath)
			continue
		}

		if parser.DetectLanguage(absPath) == "" {
			continue
		}

		fhash, err := FileHash(absPath)
		if err != nil {
			// File might be deleted
			store.RemoveFileData(absPath)
			continue
		}

		// Check if file actually changed
		existingNodes := store.GetNodesByFile(absPath)
		if len(existingNodes) > 0 && existingNodes[0].FileHash == fhash {
			continue
		}

		nodes, edges, err := p.ParseFile(absPath)
		if err != nil {
			errors = append(errors, map[string]string{"file": relPath, "error": err.Error()})
			continue
		}

		if err := store.StoreFileNodesEdges(absPath, nodes, edges, fhash); err != nil {
			errors = append(errors, map[string]string{"file": relPath, "error": err.Error()})
			continue
		}
		totalNodes += len(nodes)
		totalEdges += len(edges)
	}

	now := time.Now().Format("2006-01-02T15:04:05")
	store.SetMetadata("last_updated", now)
	store.SetMetadata("last_build_type", "incremental")

	depList := make([]string, 0, len(dependentFiles))
	for f := range dependentFiles {
		depList = append(depList, f)
	}

	return &BuildResult{
		FilesUpdated:   len(allFiles),
		TotalNodes:     totalNodes,
		TotalEdges:     totalEdges,
		ChangedFiles:   changedFiles,
		DependentFiles: depList,
		Errors:         errors,
	}
}
