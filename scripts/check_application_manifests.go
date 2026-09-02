package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var bootstrapApplicationPattern = regexp.MustCompile(`^  - code: ([a-z][a-z0-9._-]*)\s*$`)
var bootstrapComponentPattern = regexp.MustCompile(`component:\s*([a-z][a-z0-9._-]*)`)
var manifestCodePattern = regexp.MustCompile(`\bcode:\s*'([a-z][a-z0-9._-]*)'`)
var manifestPagePattern = regexp.MustCompile(`'([a-z][a-z0-9._-]*)'\s*:`)

type applicationPages map[string]map[string]struct{}

func parseBootstrapApplications(payload string) (applicationPages, error) {
	result := applicationPages{}
	current := ""
	scanner := bufio.NewScanner(strings.NewReader(payload))
	for scanner.Scan() {
		line := scanner.Text()
		if match := bootstrapApplicationPattern.FindStringSubmatch(line); len(match) == 2 {
			current = match[1]
			if _, exists := result[current]; exists {
				return nil, fmt.Errorf("duplicate bootstrap application %q", current)
			}
			result[current] = map[string]struct{}{}
			continue
		}
		for _, match := range bootstrapComponentPattern.FindAllStringSubmatch(line, -1) {
			if current == "" {
				return nil, fmt.Errorf("component %q appears before an application", match[1])
			}
			result[current][match[1]] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("bootstrap application catalog is empty")
	}
	return result, nil
}

func parseApplicationManifest(payload string) (string, map[string]struct{}, error) {
	codeMatch := manifestCodePattern.FindStringSubmatch(payload)
	if len(codeMatch) != 2 {
		return "", nil, errors.New("manifest application code is missing")
	}
	pages := map[string]struct{}{}
	for _, match := range manifestPagePattern.FindAllStringSubmatch(payload, -1) {
		pages[match[1]] = struct{}{}
	}
	if len(pages) == 0 {
		return "", nil, fmt.Errorf("application %q manifest has no pages", codeMatch[1])
	}
	return codeMatch[1], pages, nil
}

func loadApplicationManifests(root string) (applicationPages, error) {
	paths, err := filepath.Glob(filepath.Join(root, "src", "apps", "*", "manifest.ts"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no application manifests found under %s", root)
	}
	result := applicationPages{}
	for _, path := range paths {
		payload, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		code, pages, parseErr := parseApplicationManifest(string(payload))
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", path, parseErr)
		}
		if _, exists := result[code]; exists {
			return nil, fmt.Errorf("duplicate console application manifest %q", code)
		}
		result[code] = pages
	}
	return result, nil
}

func compareApplicationManifests(bootstrap, console applicationPages) error {
	problems := []string{}
	for applicationCode, components := range bootstrap {
		pages, exists := console[applicationCode]
		if !exists {
			problems = append(problems, fmt.Sprintf("bootstrap application %q has no console manifest", applicationCode))
			continue
		}
		for component := range components {
			if !strings.HasPrefix(component, applicationCode+".") {
				problems = append(problems, fmt.Sprintf("bootstrap component %q is outside %q", component, applicationCode))
			} else if _, exists = pages[component]; !exists {
				problems = append(problems, fmt.Sprintf("bootstrap component %q is missing from the console manifest", component))
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	slices.Sort(problems)
	return errors.New(strings.Join(problems, "; "))
}

func main() {
	bootstrapPayload, err := os.ReadFile("services/application-service/bootstrap/platform-applications.yaml")
	if err != nil {
		panic(err)
	}
	bootstrap, err := parseBootstrapApplications(string(bootstrapPayload))
	if err != nil {
		panic(err)
	}
	console, err := loadApplicationManifests("frontend/platform-console")
	if err != nil {
		panic(err)
	}
	if err = compareApplicationManifests(bootstrap, console); err != nil {
		panic(err)
	}
	fmt.Printf("verified %d bootstrap applications against console manifests\n", len(bootstrap))
}
