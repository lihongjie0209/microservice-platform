# Platform GitOps

Argo CD application set definitions are environment declarations only: secrets must be supplied by External Secrets/Vault and image tags must be immutable release tags or digests. Production changes move through pull requests and automated sync; services remain independently releasable.
