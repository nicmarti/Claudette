package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"claudette/internal/graph"
)

// CodeParser parses source files using tree-sitter and extracts structural information.
type CodeParser struct {
	parsers map[string]*sitter.Parser
}

// NewCodeParser creates a new parser instance.
func NewCodeParser() *CodeParser {
	return &CodeParser{
		parsers: make(map[string]*sitter.Parser),
	}
}

func (cp *CodeParser) getParser(language string) *sitter.Parser {
	if p, ok := cp.parsers[language]; ok {
		return p
	}
	lang := GetLanguage(language)
	if lang == nil {
		return nil
	}
	p := sitter.NewParser()
	p.SetLanguage(lang)
	cp.parsers[language] = p
	return p
}

// DetectLanguage returns the language for a file based on its extension.
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return ExtensionToLanguage[ext]
}

// ParseFile parses a single file and returns extracted nodes and edges.
func (cp *CodeParser) ParseFile(path string) ([]graph.NodeInfo, []graph.EdgeInfo, error) {
	language := DetectLanguage(path)
	if language == "" {
		return nil, nil, nil
	}

	parser := cp.getParser(language)
	if parser == nil {
		return nil, nil, nil
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}
	defer tree.Close()

	var nodes []graph.NodeInfo
	var edges []graph.EdgeInfo

	// File node
	lineCount := strings.Count(string(source), "\n") + 1
	nodes = append(nodes, graph.NodeInfo{
		Kind:     "File",
		Name:     path,
		FilePath: path,
		LineStart: 1,
		LineEnd:   lineCount,
		Language:  language,
	})

	// Walk the tree
	cp.extractFromTree(tree.RootNode(), source, language, path, &nodes, &edges, "", "")

	return nodes, edges, nil
}

func (cp *CodeParser) extractFromTree(
	root *sitter.Node,
	source []byte,
	language string,
	filePath string,
	nodes *[]graph.NodeInfo,
	edges *[]graph.EdgeInfo,
	enclosingClass string,
	enclosingFunc string,
) {
	classTypes := toSet(ClassTypes[language])
	funcTypes := toSet(FunctionTypes[language])
	importTypes := toSet(ImportTypes[language])
	callTypes := toSet(CallTypes[language])

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		nodeType := child.Type()

		// Classes
		if classTypes[nodeType] {
			name := getName(child, language, "class", source)
			if name != "" {
				node := graph.NodeInfo{
					Kind:       "Class",
					Name:       name,
					FilePath:   filePath,
					LineStart:  int(child.StartPoint().Row) + 1,
					LineEnd:    int(child.EndPoint().Row) + 1,
					Language:   language,
					ParentName: enclosingClass,
				}
				*nodes = append(*nodes, node)

				// CONTAINS edge
				*edges = append(*edges, graph.EdgeInfo{
					Kind:     "CONTAINS",
					Source:   filePath,
					Target:   qualify(name, filePath, enclosingClass),
					FilePath: filePath,
					Line:     int(child.StartPoint().Row) + 1,
				})

				// Inheritance edges
				bases := getBases(child, language, source)
				for _, base := range bases {
					*edges = append(*edges, graph.EdgeInfo{
						Kind:     "INHERITS",
						Source:   qualify(name, filePath, enclosingClass),
						Target:   base,
						FilePath: filePath,
						Line:     int(child.StartPoint().Row) + 1,
					})
				}

				cp.extractFromTree(child, source, language, filePath, nodes, edges, name, "")
				continue
			}
		}

		// Functions
		if funcTypes[nodeType] {
			name := getName(child, language, "function", source)
			if name != "" {
				isTest := IsTestFunction(name)
				if language == "java" && !isTest {
					isTest = hasTestAnnotation(child, source)
				}
				kind := "Function"
				if isTest {
					kind = "Test"
				}
				qualified := qualify(name, filePath, enclosingClass)
				params := getParams(child, source)
				retType := getReturnType(child, language, source)

				node := graph.NodeInfo{
					Kind:       kind,
					Name:       name,
					FilePath:   filePath,
					LineStart:  int(child.StartPoint().Row) + 1,
					LineEnd:    int(child.EndPoint().Row) + 1,
					Language:   language,
					ParentName: enclosingClass,
					Params:     params,
					ReturnType: retType,
					IsTest:     isTest,
				}
				*nodes = append(*nodes, node)

				// CONTAINS edge
				container := filePath
				if enclosingClass != "" {
					container = qualify(enclosingClass, filePath, "")
				}
				*edges = append(*edges, graph.EdgeInfo{
					Kind:     "CONTAINS",
					Source:   container,
					Target:   qualified,
					FilePath: filePath,
					Line:     int(child.StartPoint().Row) + 1,
				})

				cp.extractFromTree(child, source, language, filePath, nodes, edges, enclosingClass, name)
				continue
			}
		}

		// Imports
		if importTypes[nodeType] {
			imports := extractImport(child, language, source)
			for _, impTarget := range imports {
				*edges = append(*edges, graph.EdgeInfo{
					Kind:     "IMPORTS_FROM",
					Source:   filePath,
					Target:   impTarget,
					FilePath: filePath,
					Line:     int(child.StartPoint().Row) + 1,
				})
			}
			continue
		}

		// Calls
		if callTypes[nodeType] {
			callName := getCallName(child, language, source)
			if callName != "" && enclosingFunc != "" {
				caller := qualify(enclosingFunc, filePath, enclosingClass)
				*edges = append(*edges, graph.EdgeInfo{
					Kind:     "CALLS",
					Source:   caller,
					Target:   callName,
					FilePath: filePath,
					Line:     int(child.StartPoint().Row) + 1,
				})
			}
		}

		// Recurse for other node types
		cp.extractFromTree(child, source, language, filePath, nodes, edges, enclosingClass, enclosingFunc)
	}
}

func qualify(name, filePath, enclosingClass string) string {
	if enclosingClass != "" {
		return fmt.Sprintf("%s::%s.%s", filePath, enclosingClass, name)
	}
	return fmt.Sprintf("%s::%s", filePath, name)
}

func getName(node *sitter.Node, language, kind string, source []byte) string {
	// For Go type declarations, look for type_spec first
	if language == "go" && node.Type() == "type_declaration" {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "type_spec" {
				return getName(child, language, kind, source)
			}
		}
		return ""
	}

	nameTypes := map[string]bool{
		"identifier":          true,
		"name":                true,
		"type_identifier":     true,
		"property_identifier": true,
		"simple_identifier":   true,
		"constant":            true,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if nameTypes[child.Type()] {
			return child.Content(source)
		}
	}
	return ""
}

func getParams(node *sitter.Node, source []byte) string {
	paramTypes := map[string]bool{
		"parameters":        true,
		"formal_parameters": true,
		"parameter_list":    true,
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if paramTypes[child.Type()] {
			return child.Content(source)
		}
	}
	return ""
}

func getReturnType(node *sitter.Node, language string, source []byte) string {
	retTypes := map[string]bool{
		"type":            true,
		"return_type":     true,
		"type_annotation": true,
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if retTypes[child.Type()] {
			return child.Content(source)
		}
	}
	// Java: return type is a direct child with specific type names
	if language == "java" {
		javaRetTypes := map[string]bool{
			"type_identifier":      true,
			"void_type":            true,
			"generic_type":         true,
			"array_type":           true,
			"integral_type":        true,
			"floating_point_type":  true,
			"boolean_type":         true,
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if javaRetTypes[child.Type()] {
				return child.Content(source)
			}
		}
	}
	// Python: -> annotation
	if language == "python" {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "->" && i+1 < int(node.ChildCount()) {
				return node.Child(i + 1).Content(source)
			}
		}
	}
	return ""
}

func getBases(node *sitter.Node, language string, source []byte) []string {
	var bases []string
	switch language {
	case "python":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "argument_list" {
				for j := 0; j < int(child.ChildCount()); j++ {
					arg := child.Child(j)
					if arg.Type() == "identifier" || arg.Type() == "attribute" {
						bases = append(bases, arg.Content(source))
					}
				}
			}
		}
	case "javascript", "typescript":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "extends_clause" || child.Type() == "implements_clause" {
				for j := 0; j < int(child.ChildCount()); j++ {
					sub := child.Child(j)
					if sub.Type() == "identifier" || sub.Type() == "type_identifier" || sub.Type() == "nested_identifier" {
						bases = append(bases, sub.Content(source))
					}
				}
			}
		}
	case "go":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "type_spec" {
				for j := 0; j < int(child.ChildCount()); j++ {
					sub := child.Child(j)
					if sub.Type() == "struct_type" || sub.Type() == "interface_type" {
						for k := 0; k < int(sub.ChildCount()); k++ {
							field := sub.Child(k)
							if field.Type() == "field_declaration_list" {
								for l := 0; l < int(field.ChildCount()); l++ {
									f := field.Child(l)
									if f.Type() == "type_identifier" {
										bases = append(bases, f.Content(source))
									}
								}
							}
						}
					}
				}
			}
		}
	case "java":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "superclass", "super_interfaces", "extends_interfaces":
				for j := 0; j < int(child.ChildCount()); j++ {
					sub := child.Child(j)
					if sub.Type() == "type_identifier" {
						bases = append(bases, sub.Content(source))
					} else if sub.Type() == "type_list" {
						for k := 0; k < int(sub.ChildCount()); k++ {
							typ := sub.Child(k)
							if typ.Type() == "type_identifier" {
								bases = append(bases, typ.Content(source))
							}
						}
					}
				}
			}
		}
	}
	return bases
}

func extractImport(node *sitter.Node, language string, source []byte) []string {
	var imports []string
	text := node.Content(source)

	switch language {
	case "python":
		if node.Type() == "import_from_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "dotted_name" {
					imports = append(imports, child.Content(source))
					break
				}
			}
		} else {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "dotted_name" {
					imports = append(imports, child.Content(source))
				}
			}
		}
	case "javascript", "typescript":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "string" {
				val := strings.Trim(child.Content(source), "'\"")
				imports = append(imports, val)
			}
		}
	case "go":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "import_spec_list" {
				for j := 0; j < int(child.ChildCount()); j++ {
					spec := child.Child(j)
					if spec.Type() == "import_spec" {
						for k := 0; k < int(spec.ChildCount()); k++ {
							s := spec.Child(k)
							if s.Type() == "interpreted_string_literal" {
								imports = append(imports, strings.Trim(s.Content(source), "\""))
							}
						}
					}
				}
			} else if child.Type() == "import_spec" {
				for j := 0; j < int(child.ChildCount()); j++ {
					s := child.Child(j)
					if s.Type() == "interpreted_string_literal" {
						imports = append(imports, strings.Trim(s.Content(source), "\""))
					}
				}
			}
		}
	case "java":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "scoped_identifier" {
				imports = append(imports, child.Content(source))
				break
			}
		}
	default:
		imports = append(imports, text)
	}
	return imports
}

func getCallName(node *sitter.Node, language string, source []byte) string {
	if node.ChildCount() == 0 {
		return ""
	}
	first := node.Child(0)

	// Java method_invocation: obj.method(args) or method(args)
	if node.Type() == "method_invocation" {
		foundDot := false
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "." {
				foundDot = true
			} else if foundDot && child.Type() == "identifier" {
				return child.Content(source)
			}
		}
		if first.Type() == "identifier" {
			return first.Content(source)
		}
		return ""
	}

	// Java object_creation_expression: new ClassName(args)
	if node.Type() == "object_creation_expression" {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "type_identifier" {
				return child.Content(source)
			}
		}
		return ""
	}

	// Simple call: func_name(args)
	if first.Type() == "identifier" {
		return first.Content(source)
	}

	// Method call: obj.method(args)
	memberTypes := map[string]bool{
		"attribute":           true,
		"member_expression":   true,
		"field_expression":    true,
		"selector_expression": true,
	}
	if memberTypes[first.Type()] {
		// Get the rightmost identifier
		for j := int(first.ChildCount()) - 1; j >= 0; j-- {
			child := first.Child(j)
			switch child.Type() {
			case "identifier", "property_identifier", "field_identifier", "field_name":
				return child.Content(source)
			}
		}
		return first.Content(source)
	}

	// Scoped call
	if first.Type() == "scoped_identifier" || first.Type() == "qualified_name" {
		return first.Content(source)
	}

	return ""
}

func hasTestAnnotation(node *sitter.Node, source []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "modifiers" {
			for j := 0; j < int(child.ChildCount()); j++ {
				mod := child.Child(j)
				if mod.Type() == "marker_annotation" || mod.Type() == "annotation" {
					text := mod.Content(source)
					if text == "@Test" || strings.HasSuffix(text, ".Test") {
						return true
					}
				}
			}
		}
	}
	return false
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
