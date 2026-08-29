//go:build system

package systemtests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	for _, baseURL := range []string{identityURL, tenantURL, authorizationURL, auditURL} {
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
	for time.Now().Before(deadline) {
		audits := post(t, ctx, auditURL+"/api/v1/audit/records/query", tokens.AccessToken, map[string]any{"tenant_id": tenantID, "page": 1, "page_size": 100})
		var page struct {
			Total int64 `json:"total"`
		}
		decodeBody(t, audits, &page)
		if page.Total > 0 {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("domain events were not projected into audit-service within 30 seconds")
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
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	defer httpResponse.Body.Close()
	var result response
	if err := json.NewDecoder(httpResponse.Body).Decode(&result); err != nil {
		t.Fatalf("decode %s response: %v", target, err)
	}
	if httpResponse.StatusCode != http.StatusOK || result.Code != 0 || result.RequestID == "" {
		t.Fatalf("POST %s status=%d response=%+v", target, httpResponse.StatusCode, result)
	}
	return result
}

func decodeBody(t *testing.T, value response, target any) {
	t.Helper()
	if err := json.Unmarshal(value.Body, target); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}
