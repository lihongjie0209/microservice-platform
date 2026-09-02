package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var menuPermissionPattern = regexp.MustCompile(`permission_code:\s*([^, }]+).*permission_scope:\s*(tenant|platform)`)

type permissionImplementation struct {
	Code  string
	Scope string
}

func main() {
	implementations, err := collectPermissionImplementations("services")
	if err != nil {
		menuPermissionFail("collect service permissions: %v", err)
	}
	menus, err := parseMenuPermissions("services/application-service/bootstrap/platform-applications.yaml")
	if err != nil {
		menuPermissionFail("parse application manifest: %v", err)
	}
	if err := validateMenuPermissions(menus, implementations); err != nil {
		menuPermissionFail("%v", err)
	}
	fmt.Printf("menu permission invariants: %d menu permissions passed\n", len(menus))
}

func collectPermissionImplementations(root string) ([]permissionImplementation, error) {
	paths, err := filepath.Glob(filepath.Join(root, "*-service", "internal", "transport", "*", "*.go"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	result := make([]permissionImplementation, 0)
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		file, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", path, parseErr)
		}
		aliases := scopeAliases(file)
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok || (literal.Type != nil && !isMenuRequirementType(literal.Type)) {
				return true
			}
			values := literalStringFields(literal, aliases)
			resource, hasResource := values["Resource"]
			action, hasAction := values["Action"]
			scope, hasScope := values["Scope"]
			if hasResource && hasAction && hasScope {
				result = append(result, permissionImplementation{Code: resource + "." + action, Scope: scope})
			}
			return true
		})
		result = append(result, dynamicManagementImplementations(file)...)
	}
	return result, nil
}

// Dynamic management endpoints derive tenant/platform scope from a validated request field.
// Resolve their wrapper function to the concrete resource, then collect every literal action
// used by a handler so the manifest verifier still proves that an executable endpoint exists.
func dynamicManagementImplementations(file *ast.File) []permissionImplementation {
	helperResources := make(map[string]string)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || selectorName(call.Fun) != "AuthorizeUserManagementScope" || len(call.Args) < 5 {
				return true
			}
			if resource, ok := stringLiteral(call.Args[3]); ok {
				helperResources[function.Name.Name] = resource
			}
			return true
		})
	}

	result := make([]permissionImplementation, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		resource, ok := helperResources[selectorName(call.Fun)]
		if !ok {
			return true
		}
		action, ok := stringLiteral(call.Args[len(call.Args)-1])
		if !ok {
			return true
		}
		code := resource + "." + action
		result = append(result,
			permissionImplementation{Code: code, Scope: "ScopePrincipal"},
			permissionImplementation{Code: code, Scope: "ScopePlatform"},
		)
		return true
	})
	return result
}

func selectorName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.SelectorExpr:
		return value.Sel.Name
	case *ast.Ident:
		return value.Name
	default:
		return ""
	}
}

func stringLiteral(expression ast.Expr) (string, bool) {
	value, ok := expression.(*ast.BasicLit)
	if !ok || value.Kind != token.STRING {
		return "", false
	}
	decoded, err := strconv.Unquote(value.Value)
	return decoded, err == nil
}

func isMenuRequirementType(expression ast.Expr) bool {
	switch value := expression.(type) {
	case *ast.SelectorExpr:
		return value.Sel.Name == "Requirement"
	case *ast.Ident:
		return value.Name == "Requirement"
	default:
		return false
	}
}

func literalStringFields(literal *ast.CompositeLit, aliases map[string]string) map[string]string {
	fields := make(map[string]string)
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch value := pair.Value.(type) {
		case *ast.BasicLit:
			if value.Kind == token.STRING {
				decoded, err := strconv.Unquote(value.Value)
				if err == nil {
					fields[key.Name] = decoded
				}
			}
		case *ast.SelectorExpr:
			fields[key.Name] = value.Sel.Name
		case *ast.Ident:
			if resolved, ok := aliases[value.Name]; ok {
				fields[key.Name] = resolved
			}
		}
	}
	return fields
}

func scopeAliases(file *ast.File) map[string]string {
	aliases := make(map[string]string)
	ast.Inspect(file, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != len(assignment.Rhs) {
			return true
		}
		for index, left := range assignment.Lhs {
			name, nameOK := left.(*ast.Ident)
			value, valueOK := assignment.Rhs[index].(*ast.SelectorExpr)
			if nameOK && valueOK && strings.HasPrefix(value.Sel.Name, "Scope") {
				aliases[name.Name] = value.Sel.Name
			}
		}
		return true
	})
	return aliases
}

func parseMenuPermissions(path string) ([]permissionImplementation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	result := make([]permissionImplementation, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := menuPermissionPattern.FindStringSubmatch(scanner.Text())
		if len(match) == 3 {
			result = append(result, permissionImplementation{Code: strings.TrimSpace(match[1]), Scope: strings.TrimSpace(match[2])})
		}
	}
	return result, scanner.Err()
}

func validateMenuPermissions(menus, implementations []permissionImplementation) error {
	available := make(map[permissionImplementation]struct{}, len(implementations))
	for _, implementation := range implementations {
		available[implementation] = struct{}{}
	}
	for _, menu := range menus {
		acceptedScopes := []string{"ScopePrincipal", "ScopeTenant"}
		if menu.Scope == "platform" {
			acceptedScopes = []string{"ScopePlatform"}
		}
		found := false
		for _, scope := range acceptedScopes {
			if _, ok := available[permissionImplementation{Code: menu.Code, Scope: scope}]; ok {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("menu permission %s (%s) has no matching service Requirement", menu.Code, menu.Scope)
		}
	}
	return nil
}

func menuPermissionFail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "menu permission invariants: "+format+"\n", args...)
	os.Exit(1)
}
