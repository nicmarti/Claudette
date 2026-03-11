package incremental

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"claudette/internal/graph"
	"claudette/internal/parser"
)

const debounceDelay = 300 * time.Millisecond

// Watch watches for file changes and auto-updates the graph.
func Watch(repoRoot string, store *graph.GraphStore) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	p := parser.NewCodeParser()
	ignorePatterns := LoadIgnorePatterns(repoRoot)

	var mu sync.Mutex
	pending := make(map[string]bool)
	var timer *time.Timer

	shouldHandle := func(path string) bool {
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return false
		}
		if ShouldIgnore(rel, ignorePatterns) {
			return false
		}
		return parser.DetectLanguage(path) != ""
	}

	flush := func() {
		mu.Lock()
		paths := make([]string, 0, len(pending))
		for p := range pending {
			paths = append(paths, p)
		}
		pending = make(map[string]bool)
		timer = nil
		mu.Unlock()

		for _, absPath := range paths {
			if IsBinary(absPath) {
				continue
			}
			fhash, err := FileHash(absPath)
			if err != nil {
				continue
			}
			nodes, edges, err := p.ParseFile(absPath)
			if err != nil {
				log.Printf("Error parsing %s: %v", absPath, err)
				continue
			}
			if err := store.StoreFileNodesEdges(absPath, nodes, edges, fhash); err != nil {
				log.Printf("Error storing %s: %v", absPath, err)
				continue
			}
			now := time.Now().Format("2006-01-02T15:04:05")
			store.SetMetadata("last_updated", now)
			rel, _ := filepath.Rel(repoRoot, absPath)
			log.Printf("Updated: %s (%d nodes, %d edges)", rel, len(nodes), len(edges))
		}
	}

	schedule := func(absPath string) {
		mu.Lock()
		defer mu.Unlock()
		pending[absPath] = true
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(debounceDelay, flush)
	}

	// Add the repo root for recursive watching
	if err := watcher.Add(repoRoot); err != nil {
		return err
	}
	// Walk subdirectories
	filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == ".claudette" {
				return filepath.SkipDir
			}
			watcher.Add(path)
		}
		return nil
	})

	log.Printf("Watching %s for changes... (Ctrl+C to stop)", repoRoot)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if shouldHandle(event.Name) {
					schedule(event.Name)
				}
			}
			if event.Op&fsnotify.Remove != 0 {
				rel, err := filepath.Rel(repoRoot, event.Name)
				if err == nil && !ShouldIgnore(rel, ignorePatterns) {
					store.RemoveFileData(event.Name)
					log.Printf("Removed: %s", rel)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watch error: %v", err)
		}
	}
}
