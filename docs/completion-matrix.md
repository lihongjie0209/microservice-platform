# P0 平台完成与审计矩阵

审计日期：2026-08-31。本文记录可重复验证的已交付能力；后续能力按真实业务增长触发，不预建无边界空壳。

## 服务能力

| 服务 | 前端 POST+JSON API | 内部 gRPC | 领域事件 / 消费 | 核心持久化与并发约束 |
| --- | --- | --- | --- | --- |
| identity | 登录、刷新、登出、用户和服务账号管理、JWKS | 用户/会话/租户 Token/服务账号查询 | 用户、会话、服务账号事件 + outbox | 会话撤销、登录防爆破、乐观锁、审计主体 |
| tenant | 租户、成员、组织树、邀请、用户组、配额 | 完整管理与成员/组织范围查询 | 租户和成员事件；消费组关系用于授权投影 | Schema 隔离、配额原子消费、乐观锁、审计字段 |
| authorization | 权限、角色、绑定、RBAC/ABAC、批量决策 | 管理、决策、数据范围、缓存失效 | 权限变更事件；消费租户组事件 | CEL 条件、缓存版本、事务 outbox、乐观锁 |
| audit | 记录、查询、导出 | 记录、查询、导出 | durable 消费 `platform.>`，事件 ID 幂等 | 月分区、默认分区、保留/归档说明、脱敏 |
| config | 草稿、审批/拒绝、发布、回滚、解析 | 同等生命周期与查询契约 | 配置变更事件 + outbox | 环境/租户/服务作用域、审批分离、乐观锁 |
| notification | 模板、发送、供应商回执、状态查询 | 同等模板/发送/回执契约 | 投递状态事件 + outbox | 幂等、重试/死信、频控、回执去重 |
| file | 预签名上传/下载、分片上传、扫描、删除 | 同等文件和 multipart 契约 | 文件生命周期事件 + outbox | S3/MinIO、校验、重试删除、过期上传清理 |
| scheduler | 任务 CRUD、手动触发、执行记录 | 同等任务管理与触发契约 | 复用平台事件总线基础设施 | 动态 Reflection 调用、集群锁、乐观锁、执行幂等与审计 |
| swagger | 聚合服务目录、OpenAPI 文档和统一 Swagger UI | 不暴露业务 gRPC | 无状态，不接入事件总线 | 静态配置 + Kubernetes Service Informer 自动发现、TTL 缓存和 stale fallback |
| application | 应用目录、菜单草稿/发布、租户应用授权 | 应用、菜单版本和授权管理契约 | 应用、菜单发布、租户授权事件 + outbox | 菜单不可变发布快照、授权乐观锁、权限码引用 |
| dictionary | 静态字典、版本发布、分页/搜索/树/编码解析和 Provider 查询 | 字典管理与通用动态 Provider 数据面 | 字典发布和 Provider 变更事件 + outbox | 静态版本快照；动态数据由业务服务拥有并通过注册中心发现 |
| service-registry | 服务与实例查询管理页面接口 | 注册、续租、draining、注销、发现和 revision Watch | Redis Stream 作为可恢复的实例变更流 | Redis Lua 原子租约、令牌摘要、TTL、索引；无业务数据库 |
| workflow | 定义、发布、实例和我的任务页面接口 | 完整工作流管理契约；服务任务动态调用内部 gRPC | 状态/任务事件 + Outbox；命令 durable consumer 驱动 Temporal | 发布快照、实例/任务乐观锁、审计主体、Temporal 幂等与补偿 |

## 共享资产与交付

- `platform-protos` 是 gRPC 与事件 Proto 的唯一来源，执行 Buf lint、生成一致性和 breaking 检查。
- `platform-go` 维护主体/审计上下文、JWT/JWKS/PSK 拦截器、统一授权、全局错误码、Redsync 分布式锁、JetStream 可靠消费/重试/死信、事务 Inbox/Outbox、动态配置客户端、注册/发现缓存与故障恢复 SDK 和敏感字段脱敏。Inbox 与本服务领域写入共享事务，失败回滚领域副作用并保留尝试记录，重复投递不会重复执行处理函数。
- 所有服务使用独立 PostgreSQL Schema、角色和迁移表；Compose bootstrap 对新旧数据卷均幂等协调账号、密码、所有权和 search_path。
- 服务镜像使用非 root 用户，预创建可写日志目录，并通过 ldflags 注入版本、Git commit 和构建时间。
- Helm Library Chart 与 GitOps ApplicationSet 覆盖全部平台服务的 Service、Deployment、启动前迁移 initContainer、探针、HPA、PDB、NetworkPolicy、ExternalSecret、Swagger 发现和显式 APISIX 路由；服务能力与环境配置分层维护。
- 本地 `platform` Compose Profile 包含 PostgreSQL、MySQL、Redis、NATS JetStream、MinIO、Temporal 和全部已交付平台服务；不包含非必需的可观测性后端或 Temporal Web UI。

## 验证证据

```text
make verify            PASS  # race、vet、Proto、Swagger、Helm、Compose
make test-integration  PASS  # SDK + 各服务的隔离 Testcontainers 测试
system-tests           PASS  # PSK -> JWT -> tenant -> authorization -> NATS -> audit -> scheduler -> registry -> dynamic dictionary -> workflow -> Swagger
```

每个服务的单元/集成测试均可独立运行，不要求其他服务在线。多服务旅程只存在于平台级 `system-tests`。CI 会重新克隆各独立服务仓库后运行 Compose 和系统测试，防止本地 workspace 替换掩盖发布问题。

## 审计中修复的问题

- 非 root 镜像无法创建 `/app/logs`：镜像构建阶段创建目录并调整所有权。
- 已有 PostgreSQL 数据卷缺少后来加入的服务角色：增加幂等 startup bootstrap，不删除数据卷。
- JWT 只包含 identity audience：改为配置化多受众，以支持平台服务验证同一登录 Token。
- 多个 JetStream Stream 的主题重叠：平台领域事件统一归属 `PLATFORM_EVENTS`，消费者使用独立 durable 名称。
- 匿名注册缺少审计主体：Compose 系统引导改用受限 PSK，生产不开放。
- Swagger 无法展开 `json.RawMessage`：增加对象模型标注并重新生成完整审计请求文档。

## 尚未建设的增长能力

跨域搜索、计量和计费仍按 `platform-services.md` 的 P1/P2 触发条件建设。它们需要真实业务边界和容量目标，不应为了“看起来完整”提前部署空服务。
