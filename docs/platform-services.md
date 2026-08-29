# 平台基础服务规划

## 设计边界

- 每个服务独立仓库、镜像、数据库迁移表和发布周期。
- PostgreSQL/Kingbase 可以共享物理数据库，但每个服务使用独立 schema、账号和最小权限。
- 在线跨服务只通过 HTTP/gRPC/事件契约通信，不直接查询其他服务的数据表。报表/OLAP 优先通过 CDC、领域事件或定时导出建立独立只读模型；直连其他服务 OLTP 数据只作为经过架构评审的只读例外。
- Proto 进入统一契约仓库，生成的 Go SDK 按版本发布，生产者和消费者禁止复制定义。
- 基础设施组件不是业务微服务，不为 Redis、数据库或消息队列再包装一层无业务价值的服务。
- 所有可变持久化表统一包含 `version BIGINT NOT NULL DEFAULT 1`、`created_at`、`updated_at`、`created_by`、`updated_by`；调用主体由共享认证拦截器注入 Context，应用层显式传递给 Repository。
- 更新和软删除必须携带期望版本号，通过单条 SQL 原子递增版本；版本过期统一返回平台错误码。分布式锁仅用于乐观锁无法覆盖的跨资源/跨系统约束，锁键缩小到具体业务资源。
- PostgreSQL 字符串默认使用 `TEXT`，只有真实领域长度约束才使用限长类型；时间使用 `TIMESTAMPTZ`，数据库和连接会话统一以 `Asia/Shanghai`（UTC+08:00）展示。
- 对已知高增长表在上线前定义分区、保留、归档和删除负责人。服务迁移负责原生声明式分区，`pg_partman` 等自动维护组件由部署/DBA 层可选启用。

## P0：最小平台闭环

### 1. identity-service

负责用户身份、登录、Token 和服务身份。

- 用户、密码和账号状态
- JWT Access Token、Refresh Token、JWKS 与密钥轮换
- 服务账号和 Client Credentials
- 登录防爆破、会话撤销和设备会话
- 后续对接 OIDC、LDAP、企业微信等身份源

不负责角色权限规则，避免认证与授权强耦合。

### 2. authorization-service

负责统一授权决策。

- 角色、权限、资源和策略
- RBAC，必要时扩展 ABAC
- HTTP/gRPC 权限校验接口
- 权限变更事件和本地缓存失效
- 管理端权限与服务间权限分离

可使用 Casbin/OPA 实现策略引擎，但领域模型和管理 API 由本服务负责。

### 3. tenant-service

负责组织、租户和成员关系。

- 租户生命周期
- 组织树、部门、成员和邀请
- 用户与租户关系
- 租户配额和功能开关归属
- 为请求上下文签发可信 tenant_id

如果系统确定永远是单租户，可暂缓建设，但应在契约中预留 tenant_id。

### 4. audit-service

负责不可抵赖的业务审计记录，而不是普通应用日志。

- 操作者、租户、请求 ID、Trace ID、资源和动作
- 变更前后摘要
- 查询、导出和保留策略
- 消费业务事件异步写入
- 敏感字段脱敏和访问审计

### 5. config-service

负责业务级动态配置和功能开关。

- 按环境、租户、服务和键管理配置
- 版本、灰度、审批与回滚
- 变更推送和客户端本地缓存
- 敏感值只保存 Secret 引用，不自行保存明文密钥

服务启动所需的数据库密码、TLS 私钥等仍由 Kubernetes Secret、Vault 或 External Secrets 管理。

### 6. notification-service

负责统一发送渠道和模板。

- 邮件、短信、Webhook、站内信
- 模板、变量、语言和渠道路由
- 异步发送、重试、频控和死信
- 发送状态、回执和供应商切换
- 幂等键避免重复通知

### 7. file-service

负责对象存储的业务访问边界。

- 上传授权和预签名 URL
- 文件元数据、归属和访问控制
- 分片上传、校验、生命周期和删除
- 病毒扫描/内容审核扩展点
- S3 兼容存储，不让业务服务持有存储主密钥

## P0：共享工程资产

这些应独立版本化，但不是运行中的微服务：

### contracts

- `platform-protos`：统一管理公共类型和各服务 gRPC/事件 Proto。
- Buf lint、breaking change 检查和 BSR 发布。
- 生成带语义版本的 Go SDK。
- 公共定义包含分页、错误详情、Request ID、幂等键和审计主体。

### libraries

- `microservice-platform-go`：认证拦截器、Request ID、错误映射、Trace 传播和客户端工厂。
- `microservice-platform-go` 统一提供认证/授权主体上下文、审计字段构造，以及基于成熟 Redsync 组件的 Redis 分布式锁；服务不得复制锁算法或自行拼装 `SET NX`。
- 全局错误码、HTTP/gRPC 映射和乐观锁冲突语义由 `microservice-platform-go` 统一维护，服务不得自行重复分配编号。
- 只放稳定的横切能力，不放跨服务业务模型和数据库 Repository。
- SDK 版本必须可独立升级，避免所有服务锁步发布。

### delivery

- `platform-helm`：通用 Helm Library Chart。
- `platform-gitops`：各环境声明、镜像版本和部署策略。
- ExternalSecret、NetworkPolicy、ServiceMonitor、HPA、PDB 和迁移 Job 规范。

## P1：业务增长后建设

### gateway-service / API Gateway

优先采用 APISIX、Kong、Envoy Gateway 等成熟网关，不从头实现代理。

- 路由、TLS、跨域、外部限流和统一认证入口
- 灰度、流量切分和外部 API 版本管理
- 网关只做通用策略，不承载业务编排

### workflow-service

- 长事务、人工审批、超时和补偿
- 优先评估 Temporal，避免自行实现可靠状态机
- 仅在多个业务明确需要流程编排时建设

### search-service

- 聚合跨域搜索索引
- 通过事件更新 Elasticsearch/OpenSearch 索引
- 搜索结果最终一致，不成为业务事实来源

### scheduler-service

- 跨服务、可视化、可重试的集中任务调度
- 单服务内部任务继续使用当前 Cron 能力
- 只有需要统一治理、分片或人工重跑时才独立

### webhook-service

- 外部订阅、签名、投递、重试和回放
- 与内部事件总线隔离
- 可在 notification-service 复杂度上升后拆出

## P2：规模化阶段

- metering-service：用量计量和配额消耗。
- billing-service：套餐、账单、支付和对账。
- metadata-service：统一字典和可扩展元数据。
- rule-service：业务规则版本、发布和执行。
- data-export-service：大数据量异步导入导出。

这些服务必须由真实业务边界驱动，不建议在平台初期预建空壳。

## 基础设施清单

基础设施由平台团队管理，但不作为自研微服务：

- PostgreSQL/Kingbase/MySQL
- Redis
- Kafka、NATS JetStream 或 RocketMQ（三选一作为主要事件总线）
- S3 兼容对象存储
- Kubernetes、Ingress/API Gateway
- Prometheus、日志平台和 OpenTelemetry 后端
- Vault/External Secrets
- 容器镜像仓库和 GitHub Actions

## 推荐首批仓库

```text
go-api-template
microgen
platform-protos
microservice-platform-go
platform-helm
platform-gitops
identity-service
authorization-service
tenant-service
audit-service
config-service
notification-service
file-service
```

`microgen` 当前仍在模板仓库中，稳定后再拆成独立仓库发布二进制。

## 实施顺序

1. 创建 GitHub Organization、Team、仓库命名和权限规范。
2. 建设 `platform-protos`，定义公共错误、身份、租户和审计上下文。
3. 建设 `microservice-platform-go`，统一客户端、中间件和契约版本。
4. 建设 `platform-helm` 与 `platform-gitops`，打通一个服务的交付闭环。
5. 实现 identity、authorization、tenant 三个核心服务。
6. 接入 audit，再实现 notification、file 和 config。
7. 用两个真实业务服务验证认证、授权、审计、事件和部署链路。

## 测试门禁

- 每项功能必须有单元测试覆盖正常、边界和错误路径。
- PostgreSQL、Redis、NATS、迁移以及本服务适配器使用 Testcontainers 集成测试，并通过 `integration` build tag 与快速测试隔离。
- 每个服务是独立测试和维护主体，单元测试与集成测试不得要求其他服务运行；外部服务客户端使用进程内 fake、gRPC bufconn 或契约桩验证。
- 多服务业务旅程单独放入平台级 `system-tests`，不作为任一服务仓库测试套件的运行前提。
- 手工测试或集成测试发现缺陷后，优先补充可稳定复现根因的单元回归测试；只有无法脱离基础设施复现时才仅保留集成回归。
- CI 必须执行 race detector、vet、Buf lint/breaking/generate 一致性检查和完整集成测试。

## 暂时不要拆分的服务

- 独立健康检查服务：健康检查属于每个服务及编排平台。
- 独立分布式锁服务：直接使用 Redis 成熟锁组件。
- 独立数据库访问服务：会形成低效且脆弱的数据代理层。
- 通用 CRUD 服务：缺少明确领域边界，最终会变成新的单体。
- 自研注册中心：Kubernetes DNS/Service 已覆盖大部分需求。
- 自研配置密钥存储：密钥交给成熟 Secret 系统。
