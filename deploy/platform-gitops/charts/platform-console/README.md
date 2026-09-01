# platform-console chart

Deploys the non-root platform console image, its runtime URL configuration, Service, APISIX route, HPA, PDB, and default-deny NetworkPolicy.

The console route accepts only GET/HEAD and applies a source-IP edge rate limit plus security response headers in APISIX. Nginx repeats the non-TLS-specific headers so direct in-cluster access remains protected; APISIX owns HSTS at the TLS boundary.

The browser calls service subdomains directly, so `gateway.baseDomain` is required and must match the service ApplicationSet and environment certificate. Runtime configuration is written at container startup into a dedicated writable file while the image root filesystem remains read-only.

```bash
helm template platform-console . \
  --set namespace=platform-testing \
  --set environment=testing \
  --set image.tag=v1.4.0 \
  --set gateway.baseDomain=example.com \
  --set gateway.environmentLabel=test
```
