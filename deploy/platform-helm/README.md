# platform-library

Helm library chart for all platform services. Consumer charts include it as a dependency and render the named `platform.deployment`, `platform.service`, `platform.serviceAccount`, `platform.migrationJob`, `platform.podDisruptionBudget`, `platform.horizontalPodAutoscaler`, `platform.networkPolicy`, `platform.serviceMonitor`, `platform.externalSecret`, and `platform.apisixRoute` templates. Database migration runs as a pre-install/pre-upgrade hook, while the application still retains advisory-lock protection for safe concurrent startup.

Services that intentionally expose a frontend HTTP API invoke `platform.apisixRoute` and set `gateway.enabled=true`. The hostname is generated as `<service>.<baseDomain>` for production and `<service>.<environmentLabel>.<baseDomain>` elsewhere. APISIX resolves the referenced Service to EndpointSlices; internal-only services leave the option disabled.

Public routes enable `client-control`, `limit-req`, and `response-rewrite` by default. Consumers can tune the body-size and edge-rate budgets under `gateway.security`, but disabling the bundle is an explicit per-service decision; backend distributed limits and validation remain mandatory.
