# Platform GitOps

Argo CD application set definitions are environment declarations only: secrets must be supplied by External Secrets/Vault and image tags must be immutable release tags or digests. Production changes move through pull requests and automated sync; services remain independently releasable.

`gateway-applicationset.yaml` installs a pinned APISIX release for development, testing, staging, and production. Each environment has a distinct Namespace, IngressClass, LoadBalancer, etcd state and Admin Key. The expected secret path is `platform/<environment>/apisix` with `admin` and `viewer` fields; no chart default credential is accepted as a platform secret.
