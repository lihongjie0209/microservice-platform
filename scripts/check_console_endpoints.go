package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	composeServicePattern  = regexp.MustCompile(`^  ([a-z0-9-]+-service):\s*$`)
	composeHTTPPortPattern = regexp.MustCompile(`ports:\s*\["([0-9]+):8080"`)
	consoleEndpointPattern = regexp.MustCompile(`^\s*['"]?([a-z0-9-]+)['"]?:\s*['"](http://127\.0\.0\.1:[0-9]+)['"]`)
)

func main() {
	composeContents, err := os.ReadFile("environments/local/docker-compose.yml")
	if err != nil {
		consoleEndpointFail("read Compose: %v", err)
	}
	consoleContents, err := os.ReadFile("frontend/platform-console/public/platform-config.js")
	if err != nil {
		consoleEndpointFail("read console development configuration: %v", err)
	}
	composeEndpoints, err := parseComposeHTTPEndpoints(string(composeContents))
	if err != nil {
		consoleEndpointFail("parse Compose: %v", err)
	}
	consoleEndpoints, err := parseConsoleDevelopmentEndpoints(string(consoleContents))
	if err != nil {
		consoleEndpointFail("parse console configuration: %v", err)
	}
	if err := compareConsoleEndpoints(composeEndpoints, consoleEndpoints); err != nil {
		consoleEndpointFail("%v", err)
	}
	chartContents, err := os.ReadFile("deploy/platform-gitops/charts/platform-console/templates/configmap.yaml")
	if err != nil {
		consoleEndpointFail("read console Helm configuration: %v", err)
	}
	if err := checkChartRuntimeVariables(composeEndpoints, string(chartContents)); err != nil {
		consoleEndpointFail("%v", err)
	}
	fmt.Printf("console endpoint invariants: %d service endpoints passed\n", len(composeEndpoints))
}

func checkChartRuntimeVariables(compose map[string]int, chart string) error {
	services := make([]string, 0, len(compose))
	for service := range compose {
		services = append(services, service)
	}
	sort.Strings(services)
	for _, service := range services {
		variable := "PLATFORM_" + strings.ToUpper(strings.ReplaceAll(service, "-", "_")) + "_URL:"
		if !strings.Contains(chart, variable) {
			return fmt.Errorf("console Helm configuration is missing %s", variable)
		}
	}
	return nil
}

func parseComposeHTTPEndpoints(contents string) (map[string]int, error) {
	result := make(map[string]int)
	currentService := ""
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := scanner.Text()
		if match := composeServicePattern.FindStringSubmatch(line); len(match) == 2 {
			currentService = strings.TrimSuffix(match[1], "-service")
			continue
		}
		if currentService == "" {
			continue
		}
		if match := composeHTTPPortPattern.FindStringSubmatch(line); len(match) == 2 {
			port, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, fmt.Errorf("%s has invalid HTTP port %q", currentService, match[1])
			}
			result[currentService] = port
			currentService = ""
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no service HTTP ports found")
	}
	return result, nil
}

func parseConsoleDevelopmentEndpoints(contents string) (map[string]int, error) {
	result := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		match := consoleEndpointPattern.FindStringSubmatch(scanner.Text())
		if len(match) != 3 {
			continue
		}
		parsed, err := url.Parse(match[2])
		if err != nil {
			return nil, fmt.Errorf("%s has invalid URL: %w", match[1], err)
		}
		port, err := strconv.Atoi(parsed.Port())
		if err != nil {
			return nil, fmt.Errorf("%s has invalid port %q", match[1], parsed.Port())
		}
		result[match[1]] = port
	}
	return result, scanner.Err()
}

func compareConsoleEndpoints(compose, console map[string]int) error {
	services := make([]string, 0, len(compose))
	for service := range compose {
		services = append(services, service)
	}
	sort.Strings(services)
	for _, service := range services {
		consolePort, ok := console[service]
		if !ok {
			return fmt.Errorf("console development configuration is missing %s", service)
		}
		if consolePort != compose[service] {
			return fmt.Errorf("console %s port is %d, want Compose port %d", service, consolePort, compose[service])
		}
	}
	return nil
}

func consoleEndpointFail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "console endpoint invariants: "+format+"\n", args...)
	os.Exit(1)
}
