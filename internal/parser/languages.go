package parser

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// ExtensionToLanguage maps file extensions to tree-sitter language names.
var ExtensionToLanguage = map[string]string{
	".py":  "python",
	".js":  "javascript",
	".jsx": "javascript",
	".ts":  "typescript",
	".tsx": "typescript",
	".go":  "go",
}

// languageFunc maps language names to tree-sitter language objects.
var languageFunc = map[string]*sitter.Language{
	"python":     python.GetLanguage(),
	"javascript": javascript.GetLanguage(),
	"typescript": typescript.GetLanguage(),
	"go":         golang.GetLanguage(),
}

// GetLanguage returns the tree-sitter Language for a given name, or nil.
func GetLanguage(name string) *sitter.Language {
	return languageFunc[name]
}

// ClassTypes maps language to tree-sitter node types for class definitions.
var ClassTypes = map[string][]string{
	"python":     {"class_definition"},
	"javascript": {"class_declaration", "class"},
	"typescript": {"class_declaration", "class"},
	"go":         {"type_declaration"},
}

// FunctionTypes maps language to tree-sitter node types for function definitions.
var FunctionTypes = map[string][]string{
	"python":     {"function_definition"},
	"javascript": {"function_declaration", "method_definition", "arrow_function"},
	"typescript": {"function_declaration", "method_definition", "arrow_function"},
	"go":         {"function_declaration", "method_declaration"},
}

// ImportTypes maps language to tree-sitter node types for import statements.
var ImportTypes = map[string][]string{
	"python":     {"import_statement", "import_from_statement"},
	"javascript": {"import_statement"},
	"typescript": {"import_statement"},
	"go":         {"import_declaration"},
}

// CallTypes maps language to tree-sitter node types for call expressions.
var CallTypes = map[string][]string{
	"python":     {"call"},
	"javascript": {"call_expression", "new_expression"},
	"typescript": {"call_expression", "new_expression"},
	"go":         {"call_expression"},
}
