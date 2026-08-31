# High-growth data policy

This inventory is the production gate for append-heavy platform tables. A service
may not silently add a high-growth table: it must add an entry here with a
retention window, archive destination, deletion owner, and a partition decision.
All timestamps use `TIMESTAMPTZ` and the configured `Asia/Shanghai` presentation
timezone. “Archive” means a read-only export/CDC sink with access controls; it is
not a second OLTP table shared by services.

| Owner | Table / data | Retention and deletion owner | Archive | Partition decision | Current implementation |
| --- | --- | --- | --- | --- | --- |
| audit-service | `audit_records` | 365 days; audit operations | Object storage / OLAP | Monthly PostgreSQL partitions; optional `pg_partman` | Implemented |
| metering-service | `usage_facts` | 730 days, subject to billing policy; metering operations | OLAP / object storage | Monthly PostgreSQL/Kingbase partitions; optional `pg_partman` | Implemented |
| identity-service | `sessions` | 30 days after expiry or revocation; identity worker | CDC / compliance export | Deferred: global refresh-token uniqueness remains authoritative | Implemented |
| notification-service | `notification_deliveries` | 30 days after `sent` / `dead_letter`; notification worker | CDC / export | Deferred: `(tenant_id,idempotency_key)` remains globally unique | Implemented |
| webhook-service | `webhook_deliveries` | 30 days after terminal delivery; webhook worker | CDC / export | Deferred: `(subscription_id,event_id)` remains globally unique | Implemented |
| scheduler-service | `job_executions` | 90 days after terminal execution; scheduler worker | CDC / export | Deferred: global execution ID and job FK remain authoritative | Implemented |
| workflow-service | `workflow_task_history` | 365 days after owner instance reaches terminal state; workflow worker | CDC / export | Deferred: task/workflow identity must first include a time bucket | Implemented |
| every event producer | published Outbox records | Service-specific, never shorter than the JetStream replay window; owning service | Event archive / object storage | Use existing native partitions where present; delete only `published_at IS NOT NULL` | Shared SDK primitive implemented; workflow, import, data-export, and billing scheduling complete; remaining producers in progress |
| webhook-service | durable Inbox records | 30 days after completion; webhook worker | Not normally archived | Retention cleanup preserves failed/processing duplicate-delivery boundaries | Implemented |
| search-service | durable Inbox records | 14 days after completion; search projection worker | Not normally archived | Retention cleanup preserves failed/processing duplicate-delivery boundaries | Implemented |
| billing-service | `payment_provider_events`, `payment_attempts`, and `refunds` | Minimum 7 years after the related invoice closes; billing operations; legal hold always wins | Immutable finance archive with a verified archive receipt | Deferred: provider/idempotency keys are globally authoritative and are not time-bucketed | Enforced conservatively: runtime deletion is disabled until archive receipts and legal-hold checks are implemented |
| data-export-service | `export_jobs` metadata | 365 days after the result is marked `expired`; export worker | CDC / controlled export before deletion | Use state expiry first; partition based on measured history volume | Implemented: bounded version-conditional cleanup with supporting index |
| import-service | `import_jobs` metadata | 365 days after result objects are removed and the job is marked `expired`; import worker | CDC / controlled export before deletion | Use state expiry first; partition based on measured history volume | Implemented: bounded version-conditional cleanup with supporting index |

## Partition and deletion rules

1. Never drop a partition or delete data before the configured archive job has
   completed and been verified for that window.
2. A cleanup query must be bounded, context-aware, and only target a terminal
   state. It must have a supporting index and unit coverage for its eligibility
   predicate.
3. A global primary key, idempotency key, or foreign key can prevent PostgreSQL
   declarative time partitioning. Preserve correctness first; evolve the identity
   contract to include the partition key before partitioning.
4. Services own their own cleanup worker and never delete another service's
   schema. `platform-go/outbox` contains the common published-event primitive,
   not a cross-schema reaper.
5. `pg_partman` is deployment/DBA automation only. Runtime migrations remain
   valid on a plain PostgreSQL/Kingbase installation.

## Finance retention gate

Billing payment attempts, provider-event deduplication records, and refunds are
retained in the owning OLTP schema for at least seven years. The billing runtime
must not expose or schedule physical deletion until an archive receipt and a
legal-hold decision are stored and checked atomically for the affected window.
Until that workflow exists, growing storage is an explicit operational cost and
manual SQL deletion is outside the supported runbook. This preserves provider
callback deduplication and payment/refund evidence instead of treating an
unimplemented archive as permission to delete.
