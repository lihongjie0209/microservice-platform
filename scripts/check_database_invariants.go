package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var createTablePattern = regexp.MustCompile(`(?is)\bCREATE\s+TABLE(?:\s+IF\s+NOT\s+EXISTS)?\s+([^\s(]+)\s*\(`)

func main() {
	services, err := filepath.Glob("services/*-service")
	if err != nil {
		fail("find services: %v", err)
	}
	sort.Strings(services)

	schemas := make(map[string]string)
	migrationTables := make(map[string]string)
	for _, service := range services {
		if err := checkConfig(service, schemas, migrationTables); err != nil {
			fail("%v", err)
		}
		if err := checkMigrations(service); err != nil {
			fail("%v", err)
		}
	}

	fmt.Printf("database invariants: %d services passed\n", len(services))
}

func checkConfig(service string, schemas, migrationTables map[string]string) error {
	path := filepath.Join(service, "config", "config.yaml")
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: read base config: %w", service, err)
	}

	databaseName := nestedYAMLValue(string(contents), "database", "name")
	schema := nestedYAMLValue(string(contents), "database", "schema")
	migrationTable := nestedYAMLValue(string(contents), "migration", "table")
	if databaseName != "platform" {
		return fmt.Errorf("%s: default database name is %q, want platform", service, databaseName)
	}
	if schema == "" {
		return fmt.Errorf("%s: database schema must not be empty", service)
	}
	if migrationTable == "" {
		return fmt.Errorf("%s: migration table must not be empty", service)
	}
	if owner, exists := schemas[schema]; exists {
		return fmt.Errorf("%s and %s share database schema %q", owner, service, schema)
	}
	if owner, exists := migrationTables[migrationTable]; exists {
		return fmt.Errorf("%s and %s share migration table %q", owner, service, migrationTable)
	}
	schemas[schema] = service
	migrationTables[migrationTable] = service
	return nil
}

func nestedYAMLValue(contents, section, key string) string {
	inSection := false
	for _, line := range strings.Split(contents, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			inSection = trimmed == section+":"
			continue
		}
		if inSection && strings.HasPrefix(line, "  "+key+":") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "  "+key+":")), `"'`)
		}
	}
	return ""
}

func checkMigrations(service string) error {
	root := filepath.Join(service, "migrations")
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dialect := filepath.Base(filepath.Dir(path))
		if dialect == "postgres" || dialect == "kingbase" {
			upper := strings.ToUpper(string(contents))
			if regexp.MustCompile(`\b(?:VARCHAR|CHARACTER\s+VARYING)\b`).MatchString(upper) {
				return fmt.Errorf("%s: PostgreSQL/Kingbase strings must use TEXT unless explicitly exempted", path)
			}
			if regexp.MustCompile(`\bTIMESTAMP(?:\s+WITHOUT\s+TIME\s+ZONE)?\b`).MatchString(upper) {
				return fmt.Errorf("%s: time instants must use TIMESTAMPTZ", path)
			}
		}
		return checkCreateTableStatements(path, string(contents))
	})
}

func checkCreateTableStatements(path, contents string) error {
	for _, raw := range strings.Split(contents, ";") {
		statement := stripLineComments(raw)
		if !createTablePattern.MatchString(statement) || regexp.MustCompile(`(?is)\bPARTITION\s+OF\b`).MatchString(statement) {
			continue
		}
		for _, column := range []string{"version", "created_at", "updated_at", "created_by", "updated_by"} {
			pattern := regexp.MustCompile(`(?im)(?:^|,)\s*[` + "`\"" + `]?` + regexp.QuoteMeta(column) + `[` + "`\"" + `]?\s+`)
			if !pattern.MatchString(statement) {
				matches := createTablePattern.FindStringSubmatch(statement)
				return fmt.Errorf("%s: CREATE TABLE %q is missing %s", path, matches[1], column)
			}
		}
	}
	return nil
}

func stripLineComments(statement string) string {
	lines := strings.Split(statement, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "database invariants: "+format+"\n", args...)
	os.Exit(1)
}
