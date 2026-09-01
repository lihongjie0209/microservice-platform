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
- 已登录用户可校验当前密码后原子更新 Argon2id 凭证；保留当前会话、撤销其他有效会话，并在同一事务写入密码变更 Outbox 事件
- `/me` 只从认证主体推导用户 ID 和会话作用域，返回 identity-service 自有的当前账号资料；固定字段形状与前端快照归一化共同防止切换作用域后残留旧身份数据
- 后续对接 OIDC、LDAP、企业微信等身份源

不负责角色权限规则，避免认证与授权强耦合。

### 2. authorization-service

负责统一授权决策。

- 角色、权限、资源和策略
- RBAC，必要时扩展 ABAC
- HTTP/gRPC 权限校验接口
- 权限变更事件和本地缓存失效
- 管理端权限与服务间权限分离
- 浏览器权限、角色、角色权限和角色绑定接口显式区分租户与平台作用域，由 JWT 推导目标主体并在目标命名空间再次授权；平台绑定仅接受全局用户/服务账号，租户绑定仅接受成员/成员组/服务账号；内部 gRPC/PSK 编排保持独立服务身份边界

可使用 Casbin/OPA 实现策略引擎，但领域模型和管理 API 由本服务负责。

### 3. tenant-service

负责组织、租户和成员关系。

- 租户生命周期
- 组织树、部门、成员和邀请
- 用户与租户关系
- 租户配额和功能开关归属
- 为请求上下文签发可信 tenant_id
- 成员、组织、邀请、用户组和配额在领域层再次绑定 JWT tenant_id；全局租户目录单独使用平台作用域

如果系统确定永远是单租户，可暂缓建设，但应在契约中预留 tenant_id。

### 4. audit-service

负责不可抵赖的业务审计记录，而不是普通应用日志。

- 操作者、租户、请求 ID、Trace ID、资源和动作
- 业务事件保留应用归属；用户查询/导出必须提供租户和当前应用并校验有效授权，按 ID 读取根据记录持久化归属校验；租户/平台级动作允许应用为空并仅供受服务级授权的内部合规能力访问
- 变更前后摘要
- 查询、导出和保留策略
- 消费业务事件异步写入
- 敏感字段脱敏和访问审计

### 5. config-service

负责业务级动态配置和功能开关。

- 按环境、租户、应用、服务和键管理配置
- 支持平台默认、租户默认、应用覆盖三级作用域，解析优先级为“应用覆盖 > 租户默认 > 平台默认”
- 灰度判断逐级执行；高优先级配置未命中灰度时继续回退，而不是直接返回空值
- 只有应用覆盖需要校验租户的应用授权；平台默认和租户默认保持可复用
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
- 模板、投递、回执和幂等键均按 `tenant_id + application_id` 隔离
- 通过 application-service 校验租户是否已获当前应用授权
- 工作进程从持久化投递记录继承应用归属，事件 Envelope 同时携带租户与应用
- 历史数据采用 expand/backfill/contract 迁移，禁止用虚拟默认应用掩盖未知归属

### 7. file-service

负责对象存储的业务访问边界。

- 上传授权和预签名 URL
- 文件元数据、归属和访问控制
- 分片上传、校验、生命周期和删除
- 控制台对不超过 100 MB 的文件使用直传，更大的文件自动使用有界并发分片上传；失败会中止服务端会话，对象存储 CORS 必须向浏览器暴露 `ETag`
- 文件元数据、上传会话、幂等键及所有授权操作按 `tenant_id + application_id` 隔离
- 对象键使用租户/应用/文件层级，状态事件 Envelope 携带同一应用归属
- 对象存储连接与桶是平台基础设施配置，不代表文件可以跨应用访问
- 历史记录通过 expand/backfill/contract 迁移补齐真实应用归属
- 病毒扫描/内容审核扩展点
- S3 兼容存储，不让业务服务持有存储主密钥

### 8. scheduler-service

负责集中管理和执行跨服务的定时调用。

- 保存 Cron、时区、超时、目标服务、完整 RPC 方法名和 JSON 请求
- 任务与执行记录按 `tenant_id + application_id` 隔离，定时执行前重新校验应用授权
- 下游调用携带租户和应用元数据；业务请求中的作用域仍由下游接口契约显式校验
- 通过服务端反射在运行时解析下游 Protobuf 描述并调用一元 gRPC 接口
- 下游仅能选择配置中的服务白名单，连接认证、mTLS 和地址不进入任务表
- 分布式锁防止多副本重复执行，执行记录保留请求、响应、耗时和错误状态
- 历史无可靠应用归属的任务升级时自动禁用，完成权威回填后才能重新启用
- 管理接口自身使用版本化 `platform.scheduler.v1` 契约，HTTP 与 gRPC 均可操作

它不生成任何下游业务 Client Stub，因此下游新增兼容字段或 RPC 时无需发布 scheduler-service；删除字段、改类型等破坏性变更仍由契约仓库的 Buf 门禁禁止。下游必须在内网端口开放 Server Reflection，并通过 NetworkPolicy 与 mTLS/PSK 限制访问。

### 9. swagger-service

负责聚合所有服务的 OpenAPI 文档并提供统一 Swagger UI。开发环境使用静态服务清单；Kubernetes 环境通过只读 Service Informer 自动发现带 `platform.swagger/enabled=true` 标签与注解的服务，缓存最后一次有效文档，单个服务故障不影响目录和其他文档。控制台的 `swagger-center` 通过受保护的 POST+JSON API 获取服务目录与文档，复用当前 JWT，并直接渲染服务托管的固定版本 Swagger UI 资源，无需 iframe 二次登录。

### 10. application-service

负责应用目录、菜单发布版本和租户应用授权。菜单仅引用 authorization-service 的权限码，并显式声明 `tenant` 或 `platform` 决策作用域；租户与成员事实仍归 tenant-service。第一阶段不单独拆 menu-service；详细边界见 `application-service-design.md`。

- `platform-bootstrap` 使用 Cobra/Viper 和声明式清单，通过 application-service 的 POST+JSON API 幂等创建/更新 17 个平台应用、42 个页面菜单、不可变菜单发布版本及初始租户授权
- CLI 支持 flag > `PLATFORM_BOOTSTRAP_*` 环境变量 > 可选配置文件 > 默认值、JSON 输出和 Shell 补全；镜像内置二进制与清单，可通过受限 Kubernetes Job 重复执行
- 默认收敛不会删除清单外应用或菜单；破坏性 prune 必须是未来显式、可审计的操作，不得混入启动流程
- 应用启动器通过批量导航接口一次获取最多 100 个已授权应用的发布菜单；服务端一次校验当前租户授权，并安全过滤已撤销、未发布或已删除的应用，避免浏览器按应用产生 N+1 请求

### 11. dictionary-service

负责静态字典版本和动态业务字典的统一入口。

- 静态字典草稿、乐观锁更新、不可变发布快照、分页搜索、树与批量编码解析
- 动态 Provider 能力由 service-registry-service 注册、续租、发现和失效隔离
- 统一 `DictionaryProviderService` 数据面，不生成服务专用 Client，也不跨 schema 查询
- Provider DNS 白名单、PSK/mTLS、超时、重试、熔断、Redis 缓存、指标和 Trace
- Provider 注册、心跳和注销在 HTTP/gRPC 上强制使用配置的服务 PSK，禁止降级为用户 JWT；平台级 Provider 目录及其菜单使用 `__platform__` 授权
- 每个 Provider 实例独立注册；字典网关通过公共 SDK 的缓存、变更流、被动摘除和最后有效快照完成故障恢复
- 发布和 Provider 变更通过事务 outbox 投递 NATS JetStream

tenant-service 首先提供 `tenant.organization_units` 动态字典，后续业务服务只注册自己拥有的数据。

### 12. service-registry-service

负责平台应用层的实例租约、元数据目录和动态发现，不替代 Kubernetes DNS/Service，也不自行实现一致性协议。

- Redis Lua 原子维护实例、所有权令牌、TTL、服务索引和全局 revision
- Redis Stream 保存可按 revision 恢复的实例变更，支持 gRPC server-stream watch
- 实例注册、续租、draining 和注销使用 PSK/mTLS；查询按服务名或元数据选择器发现
- `platform-go/serviceregistry` 提供自动重注册、抖动退避、内存/磁盘快照、最大陈旧时间、加权轮询和故障实例临时摘除
- Kubernetes 环境仍由 Service/DNS 提供基础网络寻址；注册中心只补充应用元数据和实例级状态

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

### APISIX API Gateway

平台选用 Apache APISIX，不开发 `gateway-service` 反向代理。Kubernetes 使用 APISIX Ingress Controller 监听显式启用的 `ApisixRoute` 与 Service/EndpointSlice，实现路由和上游实例自动更新。

- 生产域名为 `<service>.<base-domain>`，开发、测试和预发布为 `<service>.<environment>.<base-domain>`
- 每个环境独立 APISIX release、IngressClass、LoadBalancer、Namespace、Admin Key 和 ZeroSSL DNS-01 wildcard 证书
- 路由、TLS、跨域、外部限流、安全响应头、真实客户端 IP、灰度和外部 API 版本管理
- 只有声明允许外部暴露的服务生成路由；服务注册不等于公网暴露
- 网关只做通用策略，不承载业务编排；服务端继续执行 JWT/PSK、授权和租户校验

详细规范见 `docs/api-gateway.md`。

### workflow-service

- 已实现流程定义草稿、不可变发布版本、实例、人工任务、超时、条件分支、服务任务和逆序补偿
- Temporal 负责持久执行与重试；状态与运行命令通过事务 Outbox 写入 NATS JetStream，由 durable consumer 幂等启动、通知和取消流程
- 页面使用独立 POST+JSON DTO，内部使用中央 `platform.workflow.v1` gRPC 合约；服务任务通过共享动态 Reflection SDK 调用受控上游
- 任务角色由 authorization-service 解析，客户端不得提交可信角色；所有写操作受 tenant、actor、审计字段和乐观版本约束

### search-service

- 已实现 OpenSearch 跨域投影、查询、建议、过滤、排序、高亮和聚合
- 应用内页面将查询与联想固定到当前 `tenant_id + application_id`；后端批量校验所有应用过滤项，按 ID 读取后再次校验文档持久化应用归属
- 业务服务通过事务 Outbox 发布 SearchDocument 事件；Search 使用 durable JetStream + Inbox 消费，并以 source_version 保证单调幂等
- JWT tenant、membership 与 authorization-service 角色绑定生成服务端可见性令牌；前端不能提交可信角色
- 页面 POST+JSON DTO 与中央 gRPC/事件 Proto 分离；批量 gRPC 仅供受保护的重建和回填
- 搜索结果最终一致，不成为业务事实来源

### metering-service

- 计量器定义是平台级目录，目录读写接口和控制台菜单统一在 `__platform__` 范围授权；批量用量采集、追加式调整和按小时/日/月聚合查询统一按 `tenant_id + application_id` 隔离，并通过 application-service 校验租户应用授权
- 同一批次允许包含多个租户/应用，但事务 Outbox 必须按租户/应用拆分事件，禁止用首条记录的作用域代表整个批次
- HTTP 页面接口使用 POST+JSON，内部调用使用中央 `platform.metering.v1` gRPC 契约
- 用量写入以 event ID 幂等，事务内同时写事实与 Outbox，并发布 MeterChanged/UsageRecorded 事件
- 查询在 PostgreSQL/MySQL 内完成 JSON 维度过滤、聚合和结果分页；PostgreSQL/Kingbase 明细表按时间分区
- JWT Principal 强制租户范围，服务账号用于内部采集；Billing 通过 API/事件读取，不跨 schema 查询

### billing-service

- 套餐和用量价格属于平台级目录，目录读写接口和控制台菜单统一在 `__platform__` 范围授权；订阅、账单、支付尝试和退款统一按 `tenant_id + application_id` 隔离，提供独立 POST+JSON 页面接口与中央 `platform.billing.v1` gRPC 契约
- 每个租户应用独立维护有效订阅；通过 application-service 校验应用授权，通过 metering-service API 按相同租户/应用获取用量，通过 authorization-service 统一决策套餐管理权限，绝不跨 schema 查询
- 发票、支付、提供商回调和退款具有持久化幂等边界；所有更新使用版本号乐观锁
- billing-center 提供独立“支付与退款”工作区：分页查询当前租户应用的支付尝试与退款，幂等发起支付和记录退款；支付方式引用只在提交时发送，不进入浏览器持久状态
- 下周期换套餐及期末取消由服务内定时任务推进，Redis 分布式锁限制多副本并发
- 领域事件使用携带租户和应用作用域的公共 Envelope，并通过事务 Outbox 发布到 `PLATFORM_EVENTS`

### rule-service

- 按 `tenant_id + application_id` 管理规则集、不可变规则版本、发布状态和历史版本；所有管理与评估接口显式携带应用 ID，并通过 application-service 校验租户应用授权
- 使用 Google CEL 校验并有界执行有序规则，不允许任意脚本、文件或网络访问
- 前端通过独立 POST+JSON DTO 管理、校验、发布和试算；内部服务通过中央 `platform.rule.v1.RuleService` 单次/批量 Evaluate
- 调用方显式提供 facts，规则服务不查询其他服务 Schema；发布版本以校验后的 canonical JSON 和 SHA-256 checksum 固化
- 版本号分配和持久幂等键在 rule-set 行锁内串行化，发布使用双版本乐观锁
- 发布事件与状态在同一事务写入 Outbox，公共 EventEnvelope 同时携带租户与应用作用域并投递到 `PLATFORM_EVENTS`

### data-export-service

- 按 `tenant_id + application_id` 管理异步导出任务，提供创建、查询、分页、取消、重试和短时下载 URL 的前端 POST+JSON 接口，并通过 application-service 校验应用授权
- 业务服务实现中央 `platform.export.v1.ExportProviderService` 流式协议；导出服务通过注册中心的 `platform.export.provider` 能力发现来源，不生成领域专用 client，也不查询其他服务 Schema
- CSV、JSONL、XLSX 均以批次流式编码并直接 multipart 上传 S3/MinIO，限制最大行数、字节数和执行时间，记录 SHA-256
- 幂等键、任务查询、对象路径、Provider 请求和事件 Envelope 均保留相同应用归属，不允许同租户不同应用共享任务或结果
- 任务请求与成功、失败、取消、重试、过期事件通过事务 Outbox 和 JetStream durable consumer 传递；数据库状态负责原子 claim，重投不会重复执行已结束任务
- 任务取消/重试使用版本号乐观锁；运行中取消会在进度边界中止并删除不完整对象；到期结果由 Cron 分批删除对象并标记 `expired`

### import-service

- 按 `tenant_id + application_id` 管理数据集访问和异步导入任务，所有页面与内部管理接口通过 application-service 校验租户应用授权
- 聚合注册中心中的 Import Provider 能力，提供数据集分页搜索与列定义页面接口
- 管理上传、异步校验、错误报告、人工确认、批量应用、取消、重试和任务查询
- CSV、JSONL、XLSX 使用有界批次解析，规范化数据和错误报告限制临时文件大小并存入 S3/MinIO
- 业务服务实现中央 `platform.import.v1.ImportProviderService`，导入服务不生成领域专用 Client，也不读取其他服务 Schema
- 任务、幂等键、对象路径、Worker claim、Provider 批次请求和事件 Envelope 保留同一应用归属；任务状态通过数据库原子 claim 与事务 Outbox 驱动 JetStream durable consumer，批次应用携带稳定幂等键
- 历史任务无法推断真实应用时保持空归属且不向应用 API 暴露；升级时取消未完成任务，待运营侧权威回填后再恢复访问
- 结果到期由 Cron 清理对象并写入过期事件；所有用户变更使用审计主体和版本号乐观锁

### webhook-service

- 外部订阅、签名、投递、重试和回放
- 消费中央 JetStream 事件但使用独立 durable 与事务 Inbox，不让外部回调反向耦合业务服务
- 每订阅独立加密密钥、HMAC 验签、SSRF/DNS rebinding 防护和可配置保留清理
- 已作为独立服务实现，notification-service 不承担通用外部投递职责
- 订阅、投递记录、查询、重放和测试投递统一按 `tenant_id + application_id` 隔离；应用归属由 application-service 批量授权接口校验。应用事件在公共 `EventEnvelope` 中携带 `application_id`，投递规划器在匹配 Subject 前先按租户和应用筛选，缺少应用作用域的旧事件默认不投递。

## P2：规模化阶段

- metadata-service：统一字典和可扩展元数据。

这些服务必须由真实业务边界驱动，不建议在平台初期预建空壳。

## 基础设施清单

基础设施由平台团队管理，但不作为自研微服务：

- PostgreSQL/Kingbase/MySQL
- Redis
- NATS JetStream（平台统一事件总线）
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
scheduler-service
swagger-service
application-service
dictionary-service
service-registry-service
webhook-service
search-service
metering-service
```

`microgen` 当前仍在模板仓库中，稳定后再拆成独立仓库发布二进制。

## 实施顺序

1. 创建 GitHub Organization、Team、仓库命名和权限规范。
2. 建设 `platform-protos`，定义公共错误、身份、租户和审计上下文。
3. 建设 `microservice-platform-go`，统一客户端、中间件和契约版本。
4. 建设 `platform-helm` 与 `platform-gitops`，打通一个服务的交付闭环。
5. 实现 identity、authorization、tenant 三个核心服务。
6. 接入 audit，再实现 notification、file、config 和 scheduler。
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
- 自研共识型注册中心：不实现 Raft/选主；当前注册能力建立在 Redis 租约和 Kubernetes 网络寻址之上。
- 自研配置密钥存储：密钥交给成熟 Secret 系统。
