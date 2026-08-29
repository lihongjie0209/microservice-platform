package main

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: rewrite-go-import OLD NEW ROOT...")
		os.Exit(2)
	}
	for _, root := range os.Args[3:] {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			return rewrite(path, os.Args[1], os.Args[2])
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func rewrite(path, oldPrefix, newPrefix string) error {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	changed := false
	for _, entry := range file.Imports {
		value, unquoteErr := strconv.Unquote(entry.Path.Value)
		if unquoteErr != nil || (value != oldPrefix && !strings.HasPrefix(value, oldPrefix+"/")) {
			continue
		}
		entry.Path.Value = strconv.Quote(newPrefix + strings.TrimPrefix(value, oldPrefix))
		changed = true
	}
	if !changed {
		return nil
	}
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
