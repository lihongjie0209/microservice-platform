package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAuthorizationSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		wantErr string
	}{
		{name: "accepts explicit principal scope", source: `package test
var value = authz.Requirement{Resource: "file.object", Action: "read", Scope: authz.ScopePrincipal}`},
		{name: "accepts explicit tenant zero value", source: `package test
var value = authz.Requirement{Resource: "tenant.member", Action: "read", Scope: authz.ScopeTenant}`},
		{name: "rejects omitted scope", source: `package test
var value = authz.Requirement{Resource: "file.object", Action: "read"}`, wantErr: "must declare Scope explicitly"},
		{name: "rejects omitted scope in inferred map element", source: `package test
var value = map[string]authz.Requirement{"read": {Resource: "file.object", Action: "read"}}`, wantErr: "must declare Scope explicitly"},
		{name: "ignores unrelated literals", source: `package test
var value = struct{ Resource, Action string }{Resource: "x", Action: "y"}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "source.go")
			if err := os.WriteFile(path, []byte(test.source), 0o600); err != nil {
				t.Fatal(err)
			}
			err := checkAuthorizationSource(path)
			if test.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}
