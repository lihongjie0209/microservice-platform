package main

import (
	"strings"
	"testing"
)

func TestConsoleEndpointInvariants(t *testing.T) {
	t.Parallel()
	compose, err := parseComposeHTTPEndpoints(`services:
  identity-service:
    ports: ["18081:8080", "19081:9090"]
  service-registry-service:
    ports: ["18092:8080", "19092:9090"]`)
	if err != nil {
		t.Fatal(err)
	}
	console, err := parseConsoleDevelopmentEndpoints(`services: {
  identity: 'http://127.0.0.1:18081',
  'service-registry': 'http://127.0.0.1:18092'
}`)
	if err != nil {
		t.Fatal(err)
	}
	if err := compareConsoleEndpoints(compose, console); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	console["identity"] = 18080
	if err := compareConsoleEndpoints(compose, console); err == nil || !strings.Contains(err.Error(), "want Compose port 18081") {
		t.Fatalf("error = %v, want port mismatch", err)
	}
}

func TestCompareConsoleEndpointsRejectsMissingService(t *testing.T) {
	t.Parallel()
	err := compareConsoleEndpoints(map[string]int{"swagger": 18089}, map[string]int{})
	if err == nil || !strings.Contains(err.Error(), "missing swagger") {
		t.Fatalf("error = %v, want missing service", err)
	}
}
