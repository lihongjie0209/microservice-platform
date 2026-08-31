# platform-console chart

Deploys the non-root platform console image, its runtime URL configuration, Service, APISIX route, HPA, PDB, and default-deny NetworkPolicy.

The browser calls service subdomains directly, so `gateway.baseDomain` is required and must match the service ApplicationSet and environment certificate. Runtime configuration is written at container startup into a dedicated writable file while the image root filesystem remains read-only.

```bash
helm template platform-console . \
  --set namespace=platform-testing \
  --set environment=testing \
  --set image.tag=v1.4.0 \
  --set gateway.baseDomain=example.com \
  --set gateway.environmentLabel=test
```
