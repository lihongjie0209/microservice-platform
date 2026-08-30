# Platform GitOps

Argo CD application set definitions are environment declarations only: secrets must be supplied by External Secrets/Vault and image tags must be immutable release tags or digests. Production changes move through pull requests and automated sync; services remain independently releasable.

`gateway-applicationset.yaml` installs a pinned APISIX release for development, testing, staging, and production. Each environment has a distinct Namespace, IngressClass, LoadBalancer, etcd state and Admin Key. The expected secret path is `platform/<environment>/apisix` with `admin` and `viewer` fields; no chart default credential is accepted as a platform secret.

`applicationset.yaml` deploys every platform service from the public `microservice-platform` repository through the generic `platform-service` chart; it no longer references a nonexistent repository or per-service overlays. Service facts declare database/Redis/JetStream requirements and schema ownership. Environment facts declare profile, Namespace, APISIX class and immutable release tag. A public service gets an APISIX route only after `baseDomain` is configured for that environment.
