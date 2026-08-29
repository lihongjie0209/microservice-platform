# PostgreSQL operations

`audit-pg-partman.sql.example` is intentionally not an application migration: extension installation and dropping retained partitions require database-owner privileges that service roles must not have. Archive a partition to object storage before the configured 13-month retention boundary when regulatory retention exceeds the online window. The DBA scheduler should call `partman.run_maintenance_proc()` at least daily and alert on the default partition receiving rows.
