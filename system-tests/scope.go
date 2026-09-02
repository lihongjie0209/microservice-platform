package systemtests

func scopedPayload(tenantID, applicationID string, values map[string]any) map[string]any {
	result := make(map[string]any, len(values)+2)
	for key, value := range values {
		result[key] = value
	}
	result["tenant_id"] = tenantID
	result["application_id"] = applicationID
	return result
}
