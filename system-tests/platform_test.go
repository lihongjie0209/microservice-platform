//go:build system

package systemtests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

type response struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	Body      json.RawMessage `json:"body"`
	RequestID string          `json:"request_id"`
}

func TestIdentityTenantAuthorizationAndAuditJourney(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	identityURL := serviceURL("IDENTITY", "http://127.0.0.1:18081")
	tenantURL := serviceURL("TENANT", "http://127.0.0.1:18082")
	authorizationURL := serviceURL("AUTHORIZATION", "http://127.0.0.1:18083")
	auditURL := serviceURL("AUDIT", "http://127.0.0.1:18084")
	schedulerURL := serviceURL("SCHEDULER", "http://127.0.0.1:18088")
	swaggerURL := serviceURL("SWAGGER", "http://127.0.0.1:18089")
	applicationURL := serviceURL("APPLICATION", "http://127.0.0.1:18090")
	dictionaryURL := serviceURL("DICTIONARY", "http://127.0.0.1:18091")
	workflowURL := serviceURL("WORKFLOW", "http://127.0.0.1:18094")
	searchURL := serviceURL("SEARCH", "http://127.0.0.1:18095")
	meteringURL := serviceURL("METERING", "http://127.0.0.1:18096")
	billingURL := serviceURL("BILLING", "http://127.0.0.1:18097")
	ruleURL := serviceURL("RULE", "http://127.0.0.1:18098")
	exportURL := serviceURL("DATA_EXPORT", "http://127.0.0.1:18099")
	importURL := serviceURL("IMPORT", "http://127.0.0.1:18100")
	for _, baseURL := range []string{identityURL, tenantURL, authorizationURL, auditURL, schedulerURL, swaggerURL, applicationURL, dictionaryURL, workflowURL, searchURL, meteringURL, billingURL, ruleURL, exportURL, importURL} {
		waitReady(t, ctx, baseURL)
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	bootstrapPSK := envOr("SYSTEM_TEST_BOOTSTRAP_PSK", "local-system-test-psk-00000000000000000000")
	registered := post(t, ctx, identityURL+"/api/v1/identities/register", "PSK "+bootstrapPSK, map[string]any{"username": "system_" + suffix, "display_name": "System Test", "email": "system_" + suffix + "@example.com", "password": "correct horse battery staple"})
	var user struct {
		ID string `json:"id"`
	}
	decodeBody(t, registered, &user)
	login := post(t, ctx, identityURL+"/api/v1/auth/login", "", map[string]any{"login": "system_" + suffix, "password": "correct horse battery staple"})
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	decodeBody(t, login, &tokens)
	if tokens.AccessToken == "" {
		t.Fatal("identity login returned an empty access token")
	}
	post(t, ctx, tenantURL+"/api/v1/me", tokens.AccessToken, map[string]any{})
	createdTenant := post(t, ctx, tenantURL+"/api/v1/tenants/create", tokens.AccessToken, map[string]any{"code": "tenant_" + suffix, "name": "System Tenant", "owner_user_id": user.ID})
	var tenantResult struct {
		Tenant struct {
			ID string `json:"id"`
		} `json:"tenant"`
		OwnerMembership struct {
			ID string `json:"id"`
		} `json:"owner_membership"`
	}
	decodeBody(t, createdTenant, &tenantResult)
	tenantID := tenantResult.Tenant.ID
	tokens.AccessToken = selectTenantToken(t, ctx, tenantURL, tokens.AccessToken, tenantID)
	waitImportDataset(t, ctx, importURL, tokens.AccessToken, tenantID, "billing.plans")
	planCode := "system-plan-" + suffix
	createdImport := post(t, ctx, importURL+"/api/v1/imports/create", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "provider_service": "billing-service", "dataset_code": "billing.plans", "format": "csv", "filename": "plans.csv", "idempotency_key": "system-import-" + suffix})
	var importCreation struct {
		Job struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"job"`
		UploadURL     string            `json:"upload_url"`
		UploadHeaders map[string]string `json:"upload_headers"`
	}
	decodeBody(t, createdImport, &importCreation)
	content := []byte("code,name,description,currency,billing_interval,base_amount_minor,trial_days,entitlements_json\n" + planCode + ",System Plan,,CNY,month,9900,7,{}\n")
	putUpload(t, ctx, importCreation.UploadURL, importCreation.UploadHeaders, content)
	digest := sha256.Sum256(content)
	completedImport := post(t, ctx, importURL+"/api/v1/imports/complete-upload", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "id": importCreation.Job.ID, "version": importCreation.Job.Version, "source_bytes": len(content), "source_checksum": hex.EncodeToString(digest[:])})
	var queuedImport struct {
		Version int64 `json:"version"`
	}
	decodeBody(t, completedImport, &queuedImport)
	readyImport := waitImportStatus(t, ctx, importURL, tokens.AccessToken, tenantID, importCreation.Job.ID, "ready")
	post(t, ctx, importURL+"/api/v1/imports/confirm", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "id": importCreation.Job.ID, "version": readyImport.Version, "idempotency_key": "system-import-confirm-" + suffix})
	waitImportStatus(t, ctx, importURL, tokens.AccessToken, tenantID, importCreation.Job.ID, "succeeded")
	importedPlan := post(t, ctx, billingURL+"/api/v1/plans/get", tokens.AccessToken, map[string]any{"code": planCode})
	var plan struct {
		Plan struct {
			Code string `json:"code"`
		} `json:"plan"`
	}
	decodeBody(t, importedPlan, &plan)
	if plan.Plan.Code != planCode {
		t.Fatalf("imported billing plan mismatch: %+v", plan)
	}
	organizationResponse := post(t, ctx, tenantURL+"/api/v1/organization-units/create", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "code": "engineering", "name": "Engineering"})
	var organization struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	decodeBody(t, organizationResponse, &organization)
	dictionaryResponse := post(t, ctx, dictionaryURL+"/api/v1/dictionaries/query", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "dictionary_code": "tenant.organization_units", "keyword": "Engineer", "page": 1, "page_size": 20})
	var dictionaryPage struct {
		Items []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	decodeBody(t, dictionaryResponse, &dictionaryPage)
	if dictionaryPage.Total != 1 || len(dictionaryPage.Items) != 1 || dictionaryPage.Items[0].Code != organization.Code || dictionaryPage.Items[0].Name != organization.Name {
		t.Fatalf("dynamic organization dictionary query mismatch: %+v", dictionaryPage)
	}
	resolvedResponse := post(t, ctx, dictionaryURL+"/api/v1/dictionaries/resolve", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "dictionary_code": "tenant.organization_units", "codes": []string{"engineering", "missing"}})
	var resolved []struct {
		Code  string `json:"code"`
		Found bool   `json:"found"`
	}
	decodeBody(t, resolvedResponse, &resolved)
	if len(resolved) != 2 || !resolved[0].Found || resolved[1].Found {
		t.Fatalf("dynamic organization dictionary resolve mismatch: %+v", resolved)
	}
	applicationResponse := post(t, ctx, applicationURL+"/api/v1/applications/create", tokens.AccessToken, map[string]any{
		"code": "system_" + suffix, "name": "System Application", "default_route": "/home", "metadata_json": map[string]any{},
	})
	var application struct {
		ID      string `json:"id"`
		Version int64  `json:"version"`
	}
	decodeBody(t, applicationResponse, &application)
	post(t, ctx, applicationURL+"/api/v1/applications/menus/upsert", tokens.AccessToken, map[string]any{
		"menu":             map[string]any{"application_id": application.ID, "code": "home", "type": "page", "name": "Home", "route": "/home", "component": "HomePage", "permission_code": "home.read", "visible": true},
		"expected_version": 0,
	})
	post(t, ctx, applicationURL+"/api/v1/applications/menus/publish", tokens.AccessToken, map[string]any{
		"application_id": application.ID, "application_version": application.Version, "comment": "system test",
	})
	post(t, ctx, applicationURL+"/api/v1/applications/tenant-grants/grant", tokens.AccessToken, map[string]any{
		"tenant_id": tenantID, "application_id": application.ID, "source": "system-test", "entitlements_json": map[string]any{},
	})
	waitSearchDocument(t, ctx, searchURL, tokens.AccessToken, tenantID, application.ID, "System Application")
	checkResponse := post(t, ctx, applicationURL+"/api/v1/applications/tenant-grants/batch-check", tokens.AccessToken, map[string]any{
		"tenant_id": tenantID, "application_ids": []string{application.ID},
	})
	var checks []struct {
		ApplicationID string `json:"application_id"`
		Granted       bool   `json:"granted"`
	}
	decodeBody(t, checkResponse, &checks)
	if len(checks) != 1 || !checks[0].Granted {
		t.Fatalf("tenant application grant was not active: %+v", checks)
	}
	post(t, ctx, applicationURL+"/api/v1/applications/navigation/get", tokens.AccessToken, map[string]any{"application_id": application.ID})
	definitionResponse := post(t, ctx, workflowURL+"/api/v1/workflow/definitions/create", tokens.AccessToken, map[string]any{
		"tenant_id": tenantID, "application_id": application.ID, "key": "system_" + suffix, "name": "System Workflow",
		"nodes": []map[string]any{{"id": "start", "name": "Start", "type": "start"}, {"id": "end", "name": "End", "type": "end"}},
		"edges": []map[string]any{{"from_node_id": "start", "to_node_id": "end", "priority": 1}},
	})
	var definition struct {
		ID      string `json:"id"`
		Key     string `json:"key"`
		Version int64  `json:"version"`
	}
	decodeBody(t, definitionResponse, &definition)
	post(t, ctx, workflowURL+"/api/v1/workflow/definitions/publish", tokens.AccessToken, map[string]any{"id": definition.ID, "tenant_id": tenantID, "expected_version": definition.Version})
	instanceResponse := post(t, ctx, workflowURL+"/api/v1/workflow/instances/start", tokens.AccessToken, map[string]any{
		"tenant_id": tenantID, "definition_key": definition.Key, "business_key": "system-" + suffix, "title": "System workflow", "variables_json": map[string]any{}, "idempotency_key": "system-" + suffix,
	})
	var instance struct {
		ID string `json:"id"`
	}
	decodeBody(t, instanceResponse, &instance)
	waitWorkflowCompleted(t, ctx, workflowURL, tokens.AccessToken, tenantID, instance.ID)
	meterCode := "system." + suffix
	post(t, ctx, meteringURL+"/api/v1/meters/create", tokens.AccessToken, map[string]any{
		"code": meterCode, "name": "System usage", "unit": "request", "aggregation": "sum", "dimension_keys": []string{"endpoint"},
	})
	occurredAt := time.Now().In(time.FixedZone("UTC+8", 8*60*60)).Truncate(time.Second)
	usageResponse := post(t, ctx, meteringURL+"/api/v1/usage/record", tokens.AccessToken, map[string]any{"events": []map[string]any{
		{"event_id": "system-usage-" + suffix, "tenant_id": tenantID, "meter_code": meterCode, "quantity": 7, "dimensions": map[string]string{"endpoint": "/system"}, "occurred_at": occurredAt.Format(time.RFC3339), "source_service": "system-test"},
	}})
	var usageResults []struct {
		Duplicate bool `json:"duplicate"`
	}
	decodeBody(t, usageResponse, &usageResults)
	if len(usageResults) != 1 || usageResults[0].Duplicate {
		t.Fatalf("unexpected usage ingestion result: %+v", usageResults)
	}
	usageQuery := post(t, ctx, meteringURL+"/api/v1/usage/query", tokens.AccessToken, map[string]any{
		"tenant_id": tenantID, "meter_code": meterCode, "start_at": occurredAt.Add(-time.Hour).Format(time.RFC3339), "end_at": occurredAt.Add(time.Hour).Format(time.RFC3339),
		"dimensions": map[string]string{"endpoint": "/system"}, "granularity": "hour", "page": 1, "page_size": 20,
	})
	var usagePage struct {
		Items []struct {
			Quantity int64 `json:"quantity"`
		} `json:"items"`
		TotalQuantity int64 `json:"total_quantity"`
	}
	decodeBody(t, usageQuery, &usagePage)
	if len(usagePage.Items) != 1 || usagePage.Items[0].Quantity != 7 || usagePage.TotalQuantity != 7 {
		t.Fatalf("unexpected metering aggregate: %+v", usagePage)
	}
	permissionResponse := post(t, ctx, authorizationURL+"/api/v1/authorization/permissions/create", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "code": "orders.read", "name": "Read orders", "resource_type": "order", "action": "read"})
	var permission struct {
		ID string `json:"id"`
	}
	decodeBody(t, permissionResponse, &permission)
	roleResponse := post(t, ctx, authorizationURL+"/api/v1/authorization/roles/create", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "code": "viewer", "name": "Viewer", "data_scope": "all"})
	var role struct {
		ID string `json:"id"`
	}
	decodeBody(t, roleResponse, &role)
	post(t, ctx, authorizationURL+"/api/v1/authorization/role-permissions/grant", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "role_id": role.ID, "permission_id": permission.ID})
	post(t, ctx, authorizationURL+"/api/v1/authorization/bindings/create", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "subject_id": tenantResult.OwnerMembership.ID, "subject_type": "membership", "role_id": role.ID})
	decisionResponse := post(t, ctx, authorizationURL+"/api/v1/authorization/check", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "subject_id": tenantResult.OwnerMembership.ID, "subject_type": "membership", "resource_type": "order", "resource_id": "order-1", "action": "read"})
	var decision struct {
		Allowed bool `json:"allowed"`
	}
	decodeBody(t, decisionResponse, &decision)
	if !decision.Allowed {
		t.Fatal("authorization decision denied the granted permission")
	}
	deadline := time.Now().Add(30 * time.Second)
	auditProjected := false
	for time.Now().Before(deadline) {
		audits := post(t, ctx, auditURL+"/api/v1/audit/records/query", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "page": 1, "page_size": 100})
		var page struct {
			Total int64 `json:"total"`
		}
		decodeBody(t, audits, &page)
		if page.Total > 0 {
			auditProjected = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !auditProjected {
		t.Fatal("domain events were not projected into audit-service within 30 seconds")
	}
	createdJob := post(t, ctx, schedulerURL+"/api/v1/scheduler/jobs/create", tokens.AccessToken, map[string]any{
		"name": "system-health-" + suffix, "cron_expression": "0 0 0 1 1 *", "timezone": "Asia/Shanghai",
		"upstream": "audit", "full_method": "/grpc.health.v1.Health/Check", "request_json": `{"service":""}`,
		"timeout_milliseconds": 5000, "enabled": false,
	})
	var job struct {
		ID string `json:"id"`
	}
	decodeBody(t, createdJob, &job)
	executionResponse := post(t, ctx, schedulerURL+"/api/v1/scheduler/jobs/trigger", tokens.AccessToken, map[string]any{"id": job.ID})
	var execution struct {
		Status       string `json:"status"`
		ResponseJSON string `json:"response_json"`
	}
	decodeBody(t, executionResponse, &execution)
	if execution.Status != "succeeded" || execution.ResponseJSON == "" {
		t.Fatalf("scheduler dynamic gRPC invocation failed: %+v", execution)
	}
	getContains(t, ctx, swaggerURL+"/swagger/services", "scheduler-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "application-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "dictionary-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "workflow-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "search-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "metering-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "billing-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "rule-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "data-export-service")
	getContains(t, ctx, swaggerURL+"/swagger/services", "import-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/identity-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/dictionary-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/workflow-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/search-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/metering-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/billing-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/rule-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/data-export-service")
	getOpenAPISpec(t, ctx, swaggerURL+"/swagger/spec/import-service")
}

func selectTenantToken(t *testing.T, ctx context.Context, tenantURL, token, tenantID string) string {
	t.Helper()
	response := post(t, ctx, tenantURL+"/api/v1/tenants/select", token, map[string]any{"tenant_id": tenantID})
	var result struct {
		AccessToken  string `json:"access_token"`
		TenantID     string `json:"tenant_id"`
		MembershipID string `json:"membership_id"`
	}
	decodeBody(t, response, &result)
	if result.AccessToken == "" || result.TenantID != tenantID || result.MembershipID == "" {
		t.Fatalf("select tenant returned invalid scope: %+v", result)
	}
	return result.AccessToken
}

func waitWorkflowCompleted(t *testing.T, ctx context.Context, baseURL, token, tenantID, instanceID string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		result := post(t, ctx, baseURL+"/api/v1/workflow/instances/get", token, map[string]any{"tenant_id": tenantID, "id": instanceID})
		var instance struct {
			Status string `json:"status"`
		}
		decodeBody(t, result, &instance)
		if instance.Status == "completed" {
			return
		}
		if instance.Status == "failed" || instance.Status == "cancelled" {
			t.Fatalf("workflow instance reached terminal status %q", instance.Status)
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatal("workflow instance did not complete within 30 seconds")
}

func waitSearchDocument(t *testing.T, ctx context.Context, baseURL, token, tenantID, applicationID, query string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		encoded, err := json.Marshal(map[string]any{"tenant_id": tenantID, "query": query, "document_types": []string{"application"}, "page": 1, "page_size": 20})
		if err != nil {
			t.Fatal(err)
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/search/query", bytes.NewReader(encoded))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("X-Request-ID", "system-search-"+fmt.Sprint(time.Now().UnixNano()))
		result, err := http.DefaultClient.Do(request)
		if err == nil {
			var envelope response
			decodeErr := json.NewDecoder(result.Body).Decode(&envelope)
			_ = result.Body.Close()
			if result.StatusCode == http.StatusOK && decodeErr == nil && envelope.Code == 0 {
				var page struct {
					Items []struct {
						Document struct {
							ApplicationID string `json:"application_id"`
						} `json:"document"`
					} `json:"items"`
				}
				if json.Unmarshal(envelope.Body, &page) == nil {
					for _, item := range page.Items {
						if item.Document.ApplicationID == applicationID {
							return
						}
					}
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("application %s was not projected into search within 30 seconds", applicationID)
}

func waitImportDataset(t *testing.T, ctx context.Context, baseURL, token, tenantID, code string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		result, statusCode, err := tryPost(ctx, baseURL+"/api/v1/imports/datasets/list", token, map[string]any{"tenant_id": tenantID, "search": code, "page": 1, "page_size": 20})
		if err != nil || statusCode != http.StatusOK || result.Code != 0 {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		var page struct {
			Items []struct {
				Code string `json:"code"`
			} `json:"items"`
		}
		decodeBody(t, result, &page)
		for _, item := range page.Items {
			if item.Code == code {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("import dataset %s was not discovered within 30 seconds", code)
}

type importJobStatus struct {
	Status       string `json:"status"`
	Version      int64  `json:"version"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

func waitImportStatus(t *testing.T, ctx context.Context, baseURL, token, tenantID, id, expected string) importJobStatus {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		result := post(t, ctx, baseURL+"/api/v1/imports/get", token, map[string]any{"tenant_id": tenantID, "id": id})
		var job importJobStatus
		decodeBody(t, result, &job)
		if job.Status == expected {
			return job
		}
		if job.Status == "failed" || job.Status == "validation_failed" || job.Status == "canceled" || job.Status == "expired" {
			t.Fatalf("import job reached %s while waiting for %s: %+v", job.Status, expected, job)
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("import job %s did not reach %s within 45 seconds", id, expected)
	return importJobStatus{}
}

func putUpload(t *testing.T, ctx context.Context, target string, headers map[string]string, content []byte) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	result, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(result.Body)
		t.Fatalf("PUT import source status=%d body=%s", result.StatusCode, body)
	}
}

func serviceURL(name, fallback string) string {
	return envOr("SYSTEM_TEST_"+name+"_URL", fallback)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func waitReady(t *testing.T, ctx context.Context, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		request, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/ready", bytes.NewReader([]byte(`{}`)))
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("service %s did not become ready", baseURL)
}

func post(t *testing.T, ctx context.Context, target, token string, body any) response {
	t.Helper()
	result, statusCode, err := tryPost(ctx, target, token, body)
	if err != nil {
		t.Fatal(err)
	}
	if statusCode != http.StatusOK || result.Code != 0 || result.RequestID == "" {
		t.Fatalf("POST %s status=%d response=%+v", target, statusCode, result)
	}
	return result
}

func tryPost(ctx context.Context, target, token string, body any) (response, int, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return response{}, 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(encoded))
	if err != nil {
		return response{}, 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "system-test-"+fmt.Sprint(time.Now().UnixNano()))
	if token != "" {
		if len(token) > 4 && token[:4] == "PSK " {
			request.Header.Set("Authorization", token)
		} else {
			request.Header.Set("Authorization", "Bearer "+token)
		}
	}
	httpResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		return response{}, 0, err
	}
	defer httpResponse.Body.Close()
	var result response
	if err := json.NewDecoder(httpResponse.Body).Decode(&result); err != nil {
		return response{}, httpResponse.StatusCode, err
	}
	return result, httpResponse.StatusCode, nil
}

func decodeBody(t *testing.T, value response, target any) {
	t.Helper()
	if err := json.Unmarshal(value.Body, target); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func getContains(t *testing.T, ctx context.Context, target, fragment string) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusOK || !bytes.Contains(body, []byte(fragment)) {
		t.Fatalf("GET %s status=%d body=%s", target, result.StatusCode, body)
	}
}

func getOpenAPISpec(t *testing.T, ctx context.Context, target string) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	var document struct {
		Swagger string `json:"swagger"`
		OpenAPI string `json:"openapi"`
	}
	if err := json.NewDecoder(result.Body).Decode(&document); err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusOK || (document.Swagger == "" && document.OpenAPI == "") {
		t.Fatalf("GET %s did not return an OpenAPI document: status=%d", target, result.StatusCode)
	}
}
