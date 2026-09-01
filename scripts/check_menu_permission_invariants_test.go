package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestValidateMenuPermissions(t *testing.T) {
	t.Parallel()
	implementations := []permissionImplementation{
		{Code: "application.catalog.list", Scope: "ScopePlatform"},
		{Code: "file.object.list", Scope: "ScopePrincipal"},
	}
	if err := validateMenuPermissions([]permissionImplementation{{Code: "application.catalog.list", Scope: "platform"}, {Code: "file.object.list", Scope: "tenant"}}, implementations); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, test := range []struct {
		name string
		menu permissionImplementation
		want string
	}{
		{name: "missing code", menu: permissionImplementation{Code: "unknown.list", Scope: "tenant"}, want: "no matching"},
		{name: "wrong scope", menu: permissionImplementation{Code: "file.object.list", Scope: "platform"}, want: "no matching"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateMenuPermissions([]permissionImplementation{test.menu}, implementations)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestScopeAliases(t *testing.T) {
	t.Parallel()
	file, err := parser.ParseFile(token.NewFileSet(), "source.go", `package test
func requirement() { principal := authz.ScopePrincipal; _ = authz.Requirement{Resource: "x", Action: "list", Scope: principal} }`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopeAliases(file)["principal"]; got != "ScopePrincipal" {
		t.Fatalf("principal alias = %q, want ScopePrincipal", got)
	}
}
