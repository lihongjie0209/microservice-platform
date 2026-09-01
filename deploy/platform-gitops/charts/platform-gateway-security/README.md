# platform-gateway-security

Deploy this chart once per environment into that environment's service Namespace (for example `platform-development`). It creates a namespaced ZeroSSL ACME `Issuer`, a wildcard `Certificate`, the APISIX SNI binding (`ApisixTls`), and optionally the ExternalSecret resources for EAB and DNS credentials. `ingressClassName` must match that environment's APISIX controller.

`issuer.solvers` is deliberately provider-neutral. Copy `values.cloudflare.example.yaml` for Cloudflare or supply the cert-manager DNS-01 solver supported by the authoritative DNS provider. `issuer.eabKeyID` must be injected by secret-aware GitOps tooling; the EAB HMAC key and DNS API token must only exist in Vault/External Secrets.

With the provided Cloudflare configuration, the remote record selected by `externalSecrets.eabRemoteKey` must expose the ZeroSSL EAB HMAC value under `secret`, and `externalSecrets.dnsRemoteKey` must expose the scoped Cloudflare token under `api-token`. These names match `issuer.eabSecretKey` and the solver's `apiTokenSecretRef.key`; changing either consumer requires changing the corresponding Vault property in the same rollout.

The environment domain is `<environmentLabel>.<baseDomain>` outside production and exactly `<baseDomain>` in production. This produces separate certificates such as `*.dev.aaa.com` and `*.aaa.com`.
