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

func TestDynamicManagementImplementations(t *testing.T) {
	t.Parallel()
	file, err := parser.ParseFile(token.NewFileSet(), "source.go", `package test
func (h *Handler) authorize(c Context, tenantID, scope, action string) {
	h.service.AuthorizeUserManagementScope(c, tenantID, scope, "authorization.role", action)
}

func TestValidateActionPermissionReferences(t *testing.T) {
	t.Parallel()
	actions := []string{"file.object.upload", "file.object.delete", "file.object.upload"}
	frontend := map[string]struct{}{"file.object.upload": {}}
	err := validateActionPermissionReferences(actions, frontend)
	if err == nil || !strings.Contains(err.Error(), "file.object.delete") {
		t.Fatalf("error = %v, want missing delete permission", err)
	}
	frontend["file.object.delete"] = struct{}{}
	if err := validateActionPermissionReferences(actions, frontend); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
func (h *Handler) create(c Context) { h.authorize(c, "tenant-1", "tenant", "create") }
func (h *Handler) update(c Context) { h.authorize(c, "tenant-1", "platform", "update") }
func (h *Handler) ignored(c Context, action string) { h.authorize(c, "tenant-1", "tenant", action) }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := dynamicManagementImplementations(file)
	want := map[permissionImplementation]bool{
		{Code: "authorization.role.create", Scope: "ScopePrincipal"}: true,
		{Code: "authorization.role.create", Scope: "ScopePlatform"}:  true,
		{Code: "authorization.role.update", Scope: "ScopePrincipal"}: true,
		{Code: "authorization.role.update", Scope: "ScopePlatform"}:  true,
	}
	if len(got) != len(want) {
		t.Fatalf("implementations = %#v, want %d entries", got, len(want))
	}
	for _, implementation := range got {
		if !want[implementation] {
			t.Fatalf("unexpected implementation %#v", implementation)
		}
	}
}
