package main

import (
	"strings"
	"testing"
)

func TestApplicationManifestInvariant(t *testing.T) {
	t.Parallel()
	bootstrap, err := parseBootstrapApplications(`applications:
  - code: orders
    menus:
      - {code: list, component: orders.list}
      - {code: detail, component: orders.detail}
  - code: billing
    menus:
      - {code: invoices, component: billing.invoices}
`)
	if err != nil {
		t.Fatal(err)
	}
	ordersCode, ordersPages, err := parseApplicationManifest(`export const value = {
  code: 'orders',
  pages: {'orders.list': load, 'orders.detail': load}
}`)
	if err != nil {
		t.Fatal(err)
	}
	billingCode, billingPages, err := parseApplicationManifest(`export const value = {
  code: 'billing',
  pages: {'billing.invoices': load}
}`)
	if err != nil {
		t.Fatal(err)
	}
	if err = compareApplicationManifests(bootstrap, applicationPages{
		ordersCode: ordersPages, billingCode: billingPages,
	}); err != nil {
		t.Fatalf("compareApplicationManifests() error = %v", err)
	}
}

func TestApplicationManifestInvariantReportsDrift(t *testing.T) {
	t.Parallel()
	bootstrap := applicationPages{
		"orders": {"orders.list": {}, "billing.detail": {}},
		"future": {"future.home": {}},
	}
	err := compareApplicationManifests(bootstrap, applicationPages{"orders": {"orders.other": {}}})
	if err == nil {
		t.Fatal("compareApplicationManifests() error = nil, want drift")
	}
	for _, message := range []string{
		`bootstrap application "future" has no console manifest`,
		`bootstrap component "orders.list" is missing from the console manifest`,
		`bootstrap component "billing.detail" is outside "orders"`,
	} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("compareApplicationManifests() error = %q, want %q", err, message)
		}
	}
}
