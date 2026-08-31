package main

import (
	"strings"
	"testing"
)

func TestCheckCreateTableStatements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{
			name: "accepts multiline columns",
			sql: `CREATE TABLE records (
 id TEXT PRIMARY KEY,
 version BIGINT NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL,
 created_by TEXT NOT NULL,
 updated_by TEXT NOT NULL
);`,
		},
		{
			name:    "rejects missing version",
			sql:     `CREATE TABLE records (id TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, created_by TEXT, updated_by TEXT);`,
			wantErr: "missing version",
		},
		{
			name: "ignores physical partition",
			sql:  `CREATE TABLE records_default PARTITION OF records DEFAULT;`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := checkCreateTableStatements("migration.sql", test.sql)
			if test.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestNestedYAMLValue(t *testing.T) {
	t.Parallel()

	config := "database:\n  name: platform\n  schema: identity\nmigration:\n  table: identity_schema_migrations\n"
	if got := nestedYAMLValue(config, "database", "schema"); got != "identity" {
		t.Fatalf("schema = %q, want identity", got)
	}
	if got := nestedYAMLValue(config, "migration", "table"); got != "identity_schema_migrations" {
		t.Fatalf("migration table = %q, want identity_schema_migrations", got)
	}
}
