package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	for _, root := range os.Args[1:] {
		for _, path := range []string{
			filepath.Join(root, "internal", "transport", "grpc", "server.go"),
			filepath.Join(root, "internal", "transport", "http", "handler.go"),
			filepath.Join(root, "internal", "transport", "http", "handler_test.go"),
		} {
			if err := rewrite(path); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}

func rewrite(path string) error {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	kept := file.Decls[:0]
	for _, declaration := range file.Decls {
		if removable(declaration) {
			continue
		}
		if imports, ok := declaration.(*ast.GenDecl); ok && imports.Tok == token.IMPORT {
			specs := imports.Specs[:0]
			for _, spec := range imports.Specs {
				entry := spec.(*ast.ImportSpec)
				value, _ := strconv.Unquote(entry.Path.Value)
				if strings.HasSuffix(value, "/internal/user") {
					continue
				}
				specs = append(specs, spec)
			}
			imports.Specs = specs
		}
		removeUserFieldsAndParameters(declaration)
		kept = append(kept, declaration)
	}
	file.Decls = kept
	output, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	if err := format.Node(output, set, file); err != nil {
		_ = output.Close()
		return fmt.Errorf("format %s: %w", path, err)
	}
	return output.Close()
}

func removable(declaration ast.Decl) bool {
	switch value := declaration.(type) {
	case *ast.FuncDecl:
		return removableFunction(value)
	case *ast.GenDecl:
		for _, spec := range value.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok && removableType(typeSpec.Name.Name) {
				return true
			}
		}
	}
	return false
}

func removableFunction(function *ast.FuncDecl) bool {
	if receiverName(function) == "userServer" {
		return true
	}
	switch function.Name.Name {
	case "userResponse", "toProtoUser", "Login", "CreateUser", "GetUser", "ListUsers", "UpdateUser", "DeleteUser", "TestHandler_Login":
		return true
	default:
		return false
	}
}

func removableType(name string) bool {
	switch name {
	case "userServer", "LoginRequest", "LoginResponseBody", "CreateUserRequest", "GetUserRequest", "ListUsersRequest", "UpdateUserRequest", "DeleteUserRequest":
		return true
	default:
		return false
	}
}

func removeUserFieldsAndParameters(declaration ast.Decl) {
	function, ok := declaration.(*ast.FuncDecl)
	if ok && function.Name.Name == "NewHandler" {
		function.Type.Params.List = filterFields(function.Type.Params.List, "authService")
		ast.Inspect(function.Body, func(node ast.Node) bool {
			literal, literalOK := node.(*ast.CompositeLit)
			if !literalOK {
				return true
			}
			kept := literal.Elts[:0]
			for _, element := range literal.Elts {
				pair, pairOK := element.(*ast.KeyValueExpr)
				if pairOK {
					identifier, identifierOK := pair.Key.(*ast.Ident)
					if identifierOK && identifier.Name == "auth" {
						continue
					}
				}
				kept = append(kept, element)
			}
			literal.Elts = kept
			return true
		})
	}
	general, ok := declaration.(*ast.GenDecl)
	if !ok || general.Tok != token.TYPE {
		return
	}
	for _, spec := range general.Specs {
		typeSpec, typeOK := spec.(*ast.TypeSpec)
		structure, structOK := typeSpec.Type.(*ast.StructType)
		if typeOK && structOK && typeSpec.Name.Name == "Handler" {
			structure.Fields.List = filterFields(structure.Fields.List, "auth", "users")
		}
	}
}

func filterFields(fields []*ast.Field, names ...string) []*ast.Field {
	blocked := make(map[string]struct{}, len(names))
	for _, name := range names {
		blocked[name] = struct{}{}
	}
	kept := fields[:0]
	for _, field := range fields {
		remove := false
		for _, name := range field.Names {
			if _, exists := blocked[name.Name]; exists {
				remove = true
			}
		}
		if !remove {
			kept = append(kept, field)
		}
	}
	return kept
}

func receiverName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return ""
	}
	typeExpression := function.Recv.List[0].Type
	if pointer, ok := typeExpression.(*ast.StarExpr); ok {
		typeExpression = pointer.X
	}
	if identifier, ok := typeExpression.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}
