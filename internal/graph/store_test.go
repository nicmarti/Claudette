package graph

import (
	"testing"
)

func newTestStore(t *testing.T) *GraphStore {
	t.Helper()
	store, err := NewGraphStore(":memory:")
	if err != nil {
		t.Fatalf("NewGraphStore: %v", err)
	}
	return store
}

func TestUpsertNodeAndGet(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	node := &NodeInfo{
		Kind:     "Function",
		Name:     "greet",
		FilePath: "main.py",
		LineStart: 10,
		LineEnd:   15,
		Language:  "python",
	}

	id, err := store.UpsertNode(node, "abc123")
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	got := store.GetNode("main.py::greet")
	if got == nil {
		t.Fatal("expected node, got nil")
	}
	if got.Name != "greet" {
		t.Errorf("expected name 'greet', got %q", got.Name)
	}
	if got.Kind != "Function" {
		t.Errorf("expected kind 'Function', got %q", got.Kind)
	}
	if got.FileHash != "abc123" {
		t.Errorf("expected file_hash 'abc123', got %q", got.FileHash)
	}
}

func TestUpsertEdge(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	edge := &EdgeInfo{
		Kind:     "CALLS",
		Source:   "main.py::greet",
		Target:   "main.py::speak",
		FilePath: "main.py",
		Line:     12,
	}

	id, err := store.UpsertEdge(edge)
	if err != nil {
		t.Fatalf("UpsertEdge: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	edges := store.GetEdgesBySource("main.py::greet")
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Kind != "CALLS" {
		t.Errorf("expected kind CALLS, got %q", edges[0].Kind)
	}
}

func TestStoreFileNodesEdges(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	nodes := []NodeInfo{
		{Kind: "File", Name: "test.py", FilePath: "test.py", LineStart: 1, LineEnd: 20, Language: "python"},
		{Kind: "Function", Name: "foo", FilePath: "test.py", LineStart: 5, LineEnd: 10, Language: "python"},
	}
	edges := []EdgeInfo{
		{Kind: "CONTAINS", Source: "test.py", Target: "test.py::foo", FilePath: "test.py", Line: 5},
	}

	err := store.StoreFileNodesEdges("test.py", nodes, edges, "hash1")
	if err != nil {
		t.Fatalf("StoreFileNodesEdges: %v", err)
	}

	files := store.GetAllFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	nodesByFile := store.GetNodesByFile("test.py")
	if len(nodesByFile) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodesByFile))
	}
}

func TestGetStats(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	nodes := []NodeInfo{
		{Kind: "File", Name: "test.py", FilePath: "test.py", LineStart: 1, LineEnd: 20, Language: "python"},
		{Kind: "Function", Name: "foo", FilePath: "test.py", LineStart: 5, LineEnd: 10, Language: "python"},
		{Kind: "Class", Name: "Bar", FilePath: "test.py", LineStart: 12, LineEnd: 18, Language: "python"},
	}
	edges := []EdgeInfo{
		{Kind: "CONTAINS", Source: "test.py", Target: "test.py::foo", FilePath: "test.py", Line: 5},
		{Kind: "CONTAINS", Source: "test.py", Target: "test.py::Bar", FilePath: "test.py", Line: 12},
	}

	store.StoreFileNodesEdges("test.py", nodes, edges, "hash1")

	stats := store.GetStats()
	if stats.TotalNodes != 3 {
		t.Errorf("expected 3 nodes, got %d", stats.TotalNodes)
	}
	if stats.TotalEdges != 2 {
		t.Errorf("expected 2 edges, got %d", stats.TotalEdges)
	}
	if stats.FilesCount != 1 {
		t.Errorf("expected 1 file, got %d", stats.FilesCount)
	}
	if len(stats.Languages) != 1 || stats.Languages[0] != "python" {
		t.Errorf("expected [python], got %v", stats.Languages)
	}
}

func TestSearchNodes(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	nodes := []NodeInfo{
		{Kind: "File", Name: "test.py", FilePath: "test.py", LineStart: 1, LineEnd: 20, Language: "python"},
		{Kind: "Function", Name: "hello_world", FilePath: "test.py", LineStart: 5, LineEnd: 10, Language: "python"},
		{Kind: "Function", Name: "goodbye", FilePath: "test.py", LineStart: 12, LineEnd: 15, Language: "python"},
	}
	store.StoreFileNodesEdges("test.py", nodes, nil, "hash1")

	results := store.SearchNodes("hello", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "hello_world" {
		t.Errorf("expected 'hello_world', got %q", results[0].Name)
	}
}

func TestImpactRadius(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Build a small graph: a.py -> b.py -> c.py
	store.StoreFileNodesEdges("a.py", []NodeInfo{
		{Kind: "File", Name: "a.py", FilePath: "a.py", LineStart: 1, LineEnd: 10, Language: "python"},
		{Kind: "Function", Name: "funcA", FilePath: "a.py", LineStart: 2, LineEnd: 5, Language: "python"},
	}, []EdgeInfo{
		{Kind: "CONTAINS", Source: "a.py", Target: "a.py::funcA", FilePath: "a.py", Line: 2},
		{Kind: "CALLS", Source: "a.py::funcA", Target: "b.py::funcB", FilePath: "a.py", Line: 3},
	}, "hash_a")

	store.StoreFileNodesEdges("b.py", []NodeInfo{
		{Kind: "File", Name: "b.py", FilePath: "b.py", LineStart: 1, LineEnd: 10, Language: "python"},
		{Kind: "Function", Name: "funcB", FilePath: "b.py", LineStart: 2, LineEnd: 5, Language: "python"},
	}, []EdgeInfo{
		{Kind: "CONTAINS", Source: "b.py", Target: "b.py::funcB", FilePath: "b.py", Line: 2},
		{Kind: "CALLS", Source: "b.py::funcB", Target: "c.py::funcC", FilePath: "b.py", Line: 3},
	}, "hash_b")

	store.StoreFileNodesEdges("c.py", []NodeInfo{
		{Kind: "File", Name: "c.py", FilePath: "c.py", LineStart: 1, LineEnd: 10, Language: "python"},
		{Kind: "Function", Name: "funcC", FilePath: "c.py", LineStart: 2, LineEnd: 5, Language: "python"},
	}, []EdgeInfo{
		{Kind: "CONTAINS", Source: "c.py", Target: "c.py::funcC", FilePath: "c.py", Line: 2},
	}, "hash_c")

	// Impact from b.py, depth 1
	result := store.GetImpactRadius([]string{"b.py"}, 1)
	if len(result.ChangedNodes) == 0 {
		t.Error("expected changed nodes from b.py")
	}
	// Should reach a.py::funcA (reverse edge) and c.py::funcC (forward edge)
	if len(result.ImpactedNodes) == 0 {
		t.Error("expected impacted nodes")
	}
}

func TestMetadata(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	store.SetMetadata("last_updated", "2026-03-11T12:00:00")
	val, ok := store.GetMetadata("last_updated")
	if !ok {
		t.Fatal("expected metadata to exist")
	}
	if val != "2026-03-11T12:00:00" {
		t.Errorf("expected '2026-03-11T12:00:00', got %q", val)
	}

	_, ok = store.GetMetadata("nonexistent")
	if ok {
		t.Error("expected nonexistent key to not be found")
	}
}
