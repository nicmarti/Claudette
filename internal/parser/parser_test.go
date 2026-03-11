package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"foo.py", "python"},
		{"bar.js", "javascript"},
		{"baz.ts", "typescript"},
		{"main.go", "go"},
		{"readme.md", ""},
		{"data.json", ""},
	}
	for _, tt := range tests {
		got := DetectLanguage(tt.path)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestParsePythonFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.py")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata/sample.py not found")
	}

	cp := NewCodeParser()
	nodes, edges, err := cp.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected nodes, got none")
	}
	if len(edges) == 0 {
		t.Fatal("expected edges, got none")
	}

	// Check for File node
	var hasFile, hasClass, hasFunc, hasTest bool
	for _, n := range nodes {
		switch n.Kind {
		case "File":
			hasFile = true
		case "Class":
			hasClass = true
		case "Function":
			hasFunc = true
		case "Test":
			hasTest = true
		}
	}

	if !hasFile {
		t.Error("expected a File node")
	}
	if !hasClass {
		t.Error("expected a Class node (Animal or Dog)")
	}
	if !hasFunc {
		t.Error("expected a Function node")
	}
	if !hasTest {
		t.Error("expected a Test node (test_dog_speak)")
	}

	// Check for edges
	var hasContains, hasImports, hasCalls, hasInherits bool
	for _, e := range edges {
		switch e.Kind {
		case "CONTAINS":
			hasContains = true
		case "IMPORTS_FROM":
			hasImports = true
		case "CALLS":
			hasCalls = true
		case "INHERITS":
			hasInherits = true
		}
	}

	if !hasContains {
		t.Error("expected CONTAINS edges")
	}
	if !hasImports {
		t.Error("expected IMPORTS_FROM edges")
	}
	if !hasCalls {
		t.Error("expected CALLS edges")
	}
	if !hasInherits {
		t.Error("expected INHERITS edges (Dog -> Animal)")
	}
}

func TestParseGoFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.go")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata/sample.go not found")
	}

	cp := NewCodeParser()
	nodes, edges, err := cp.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected nodes, got none")
	}

	// Should have File, Class (struct types), Function (methods + functions)
	var hasFile, hasClass, hasFunc bool
	for _, n := range nodes {
		switch n.Kind {
		case "File":
			hasFile = true
		case "Class":
			hasClass = true
		case "Function":
			hasFunc = true
		}
	}

	if !hasFile {
		t.Error("expected a File node")
	}
	if !hasClass {
		t.Error("expected Class nodes (Animal, Dog structs)")
	}
	if !hasFunc {
		t.Error("expected Function nodes (Speak, Greet)")
	}

	// Should have CONTAINS edges
	var hasContains bool
	for _, e := range edges {
		if e.Kind == "CONTAINS" {
			hasContains = true
			break
		}
	}
	if !hasContains {
		t.Error("expected CONTAINS edges")
	}
}

func TestParseTypeScriptFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.ts")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata/sample.ts not found")
	}

	cp := NewCodeParser()
	nodes, edges, err := cp.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected nodes, got none")
	}
	if len(edges) == 0 {
		t.Fatal("expected edges, got none")
	}

	// Check for class and function nodes
	var hasClass, hasFunc bool
	for _, n := range nodes {
		if n.Kind == "Class" {
			hasClass = true
		}
		if n.Kind == "Function" {
			hasFunc = true
		}
	}

	if !hasClass {
		t.Error("expected Class nodes")
	}
	if !hasFunc {
		t.Error("expected Function nodes")
	}
}

func TestIsTestFunction(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"test_something", true},
		{"TestSomething", true},
		{"something_test", true},
		{"something_spec", true},
		{"regular_func", false},
		{"helper", false},
	}
	for _, tt := range tests {
		got := IsTestFunction(tt.name)
		if got != tt.want {
			t.Errorf("IsTestFunction(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"test_main.py", true},
		{"main_test.py", true},
		{"main.test.ts", true},
		{"main.spec.js", true},
		{"main_test.go", true},
		{"tests/test_foo.py", true},
		{"main.py", false},
		{"main.go", false},
	}
	for _, tt := range tests {
		got := IsTestFile(tt.path)
		if got != tt.want {
			t.Errorf("IsTestFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
