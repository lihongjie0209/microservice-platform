# Local environment

一个 PostgreSQL 数据库 `platform` 通过 schema 隔离三个服务：

| Service | Role | Schema |
| --- | --- | --- |
| identity-service | `identity_service` | `identity` |
| tenant-service | `tenant_service` | `tenant` |
| authorization-service | `authorization_service` | `authorization` |

本地密码只用于开发环境。测试和生产通过 Secret 管理系统注入独立凭证。

```bash
docker compose up -d
docker compose ps
```

NATS 在 `4222` 提供客户端连接，`8222` 提供监控；JetStream 使用文件持久化。Redis 在 `6379` 提供缓存、限流、幂等和锁能力。

