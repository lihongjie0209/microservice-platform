package systemtests

import "testing"

func TestScopedPayloadAlwaysCarriesTenantAndApplication(t *testing.T) {
	t.Parallel()

	values := map[string]any{"id": "job-1", "tenant_id": "stale", "application_id": "stale"}
	result := scopedPayload("tenant-1", "app-1", values)
	if result["tenant_id"] != "tenant-1" || result["application_id"] != "app-1" || result["id"] != "job-1" {
		t.Fatalf("scopedPayload() = %#v", result)
	}
	if values["tenant_id"] != "stale" || values["application_id"] != "stale" {
		t.Fatalf("scopedPayload() mutated input: %#v", values)
	}
}
