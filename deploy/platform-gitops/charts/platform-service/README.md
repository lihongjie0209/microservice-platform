# platform-service

This chart is the single GitOps consumer of `platform-library` for platform Go services. ApplicationSet supplies service ownership facts (schema, migration table and infrastructure capabilities) plus environment facts (Namespace, image version, APISIX domain/class and secret path).

External exposure is opt-in. A service marked public receives an `ApisixRoute` only after the environment has a non-empty `baseDomain`; internal gRPC remains a ClusterIP port. Database migration defaults to a Pod init container: configuration and ExternalSecret must exist, migration succeeds before the API container starts, and concurrent replicas are serialized by the service's PostgreSQL advisory lock. Application auto-migration stays disabled. A standalone Helm hook Job remains an explicit mode only for environments that pre-provision its ConfigMap and Secret.
