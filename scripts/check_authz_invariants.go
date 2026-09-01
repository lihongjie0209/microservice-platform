package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	paths, err := authorizationSourceFiles("services")
	if err != nil {
		authzFail("find transport sources: %v", err)
	}
	for _, path := range paths {
		if err := checkAuthorizationSource(path); err != nil {
			authzFail("%v", err)
		}
	}
	fmt.Printf("authorization invariants: %d transport files passed\n", len(paths))
}

func authorizationSourceFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		clean := filepath.ToSlash(path)
		if strings.Contains(clean, "/internal/transport/http/") || strings.Contains(clean, "/internal/transport/grpc/") {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func checkAuthorizationSource(path string) error {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return fmt.Errorf("%s: parse: %w", path, err)
	}
	var violation error
	ast.Inspect(file, func(node ast.Node) bool {
		if violation != nil {
			return false
		}
		literal, ok := node.(*ast.CompositeLit)
		if !ok || (literal.Type != nil && !isRequirementType(literal.Type)) {
			return true
		}
		fields := keyedFields(literal)
		if fields["Resource"] && fields["Action"] && !fields["Scope"] {
			position := fileSet.Position(literal.Pos())
			violation = fmt.Errorf("%s:%d: authorization Requirement with Resource and Action must declare Scope explicitly", path, position.Line)
			return false
		}
		return true
	})
	return violation
}

func isRequirementType(expression ast.Expr) bool {
	switch value := expression.(type) {
	case *ast.SelectorExpr:
		return value.Sel.Name == "Requirement"
	case *ast.Ident:
		return value.Name == "Requirement"
	default:
		return false
	}
}

func keyedFields(literal *ast.CompositeLit) map[string]bool {
	fields := make(map[string]bool)
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := pair.Key.(*ast.Ident); ok {
			fields[key.Name] = true
		}
	}
	return fields
}

func authzFail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "authorization invariants: "+format+"\n", args...)
	os.Exit(1)
}
