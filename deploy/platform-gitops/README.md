# Platform GitOps

Argo CD application set definitions are environment declarations only: secrets must be supplied by External Secrets/Vault and image tags must be immutable release tags or digests. Production changes move through pull requests and automated sync; services remain independently releasable.

`gateway-applicationset.yaml` installs a pinned APISIX release for development, testing, staging, and production. Each environment has a distinct Namespace, IngressClass, LoadBalancer, etcd state and Admin Key. The expected secret path is `platform/<environment>/apisix` with `admin` and `viewer` fields; no chart default credential is accepted as a platform secret.

`gateway-security-applicationset.yaml` deploys the matching namespaced ZeroSSL Issuer, DNS-01 wildcard Certificate, APISIX TLS binding, and ExternalSecrets for every APISIX environment. An entry remains safely disabled while `baseDomain` is empty; before enabling it, configure the domain, ACME email, ZeroSSL EAB key ID, and the environment-specific Vault records at `platform/<environment>/zerossl-eab` and `platform/<environment>/dns`.

`applicationset.yaml` deploys every platform service from the public `microservice-platform` repository through the generic `platform-service` chart; it no longer references a nonexistent repository or per-service overlays. Service facts declare database/Redis/JetStream requirements and schema ownership. Environment facts declare profile, Namespace, APISIX class and immutable release tag. A public service gets an APISIX route only after `baseDomain` is configured for that environment.

`console-applicationset.yaml` independently deploys the SPA through the `platform-console` chart. It derives every browser-facing service URL from the same environment domain as APISIX: development uses `console.dev.<baseDomain>`, testing uses `console.test.<baseDomain>`, and production uses `console.<baseDomain>`. Configure `baseDomain` and replace the example image tags before enabling sync; the chart deliberately rejects an empty domain so a console cannot be deployed with unusable browser URLs.
