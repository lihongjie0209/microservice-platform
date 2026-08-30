# platform-gateway-security

Deploy this chart once per environment into that environment's service Namespace (for example `platform-development`). It creates a namespaced ZeroSSL ACME `Issuer`, a wildcard `Certificate`, the APISIX SNI binding (`ApisixTls`), and optionally the ExternalSecret resources for EAB and DNS credentials. `ingressClassName` must match that environment's APISIX controller.

`issuer.solvers` is deliberately provider-neutral. Copy `values.cloudflare.example.yaml` for Cloudflare or supply the cert-manager DNS-01 solver supported by the authoritative DNS provider. `issuer.eabKeyID` must be injected by secret-aware GitOps tooling; the EAB HMAC key and DNS API token must only exist in Vault/External Secrets.

The environment domain is `<environmentLabel>.<baseDomain>` outside production and exactly `<baseDomain>` in production. This produces separate certificates such as `*.dev.aaa.com` and `*.aaa.com`.
