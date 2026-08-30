# Local environment

一个 PostgreSQL 数据库 `platform` 通过 schema 隔离平台服务：

| Service | Role | Schema |
| --- | --- | --- |
| identity-service | `identity_service` | `identity` |
| tenant-service | `tenant_service` | `tenant` |
| authorization-service | `authorization_service` | `authorization` |
| audit-service | `audit_service` | `audit` |
| config-service | `config_service` | `config` |
| notification-service | `notification_service` | `notification` |
| file-service | `file_service` | `file` |
| scheduler-service | `scheduler_service` | `scheduler` |
| application-service | `application_service` | `application` |
| dictionary-service | `dictionary_service` | `dictionary` |
| webhook-service | `webhook_service` | `webhook` |
| workflow-service | `workflow_service` | `workflow` |
| search-service | `search_service` | `search` |
| metering-service | `metering_service` | `metering` |

本地密码只用于开发环境。测试和生产通过 Secret 管理系统注入独立凭证。

```bash
docker compose up -d
docker compose ps
```

NATS 在 `4222` 提供客户端连接，`8222` 提供监控；JetStream 使用文件持久化。Temporal gRPC 在 `7233` 提供工作流运行时，不启动 Temporal Web UI。Redis 在 `6379` 提供缓存、限流、幂等和锁能力。兼容性开发可使用 MySQL `3306`，文件服务使用 MinIO S3 API `9000` 与控制台 `9001`。本环境按要求不包含 Prometheus、Grafana、OTel Collector 或 Jaeger。
