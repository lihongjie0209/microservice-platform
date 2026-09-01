# 应用、租户授权与菜单服务边界

## 结论

第一阶段新增一个 `application-service`，不要同时拆出 `menu-service`。应用目录、菜单定义、菜单发布版本和租户应用授权具有同一生命周期，放在一个服务中能保持事务一致性，也不会引入低价值的跨服务 CRUD。

`application-service` 不拥有租户、成员、角色和权限规则：

- `tenant-service` 是租户、组织、成员关系的事实源。
- `authorization-service` 是 RBAC/ABAC、资源权限和授权决策的事实源。
- `application-service` 是应用目录、应用版本、导航菜单，以及租户是否获准使用某应用的事实源。
- 网关/BFF 负责组合“租户拥有的应用”和“当前成员通过授权决策的菜单”，不直接跨库查询。

## 核心模型

所有可变表使用平台统一的 `version`、`created_at`、`updated_at`、`created_by`、`updated_by` 字段；更新和删除携带期望版本。

### applications

- `id`、稳定 `code`、名称、说明、图标
- 状态：draft / active / disabled / archived
- 默认入口、排序、可见性和元数据
- 应用不是部署实例，不保存服务地址或密钥

### application_releases

- 应用的可发布配置版本
- draft / published / retired
- 发布版本冻结菜单树，支持回滚和审计

### menus

- 所属应用和 release
- 父节点、类型（目录/页面/动作/外链）、路由、组件键、图标、排序
- `permission_code` 只引用 authorization-service 的权限码，不复制角色或策略；`permission_scope` 明确该码在租户 membership 或保留平台用户主体上决策
- 国际化使用稳定 `i18n_key`，翻译内容可随发布版本维护
- 使用 materialized path 或 closure table 支持稳定树查询；同级排序建立组合索引

### tenant_application_grants

- `tenant_id + application_id` 唯一
- 状态、有效期、来源（manual/trial/subscription）、可选 entitlement JSON
- 表达“租户能否进入该应用”，不表达某个用户是否有菜单权限
- 高频校验按 tenant_id 分区/索引；到期任务只更新具体 grant，并发布失效事件

## API 与调用链

管理 API：应用 CRUD、菜单草稿编辑、发布/回滚、授予/撤销租户应用、租户授权列表。更新和撤销均使用乐观锁；发布菜单版本使用 `application_id` 粒度的共享 Redsync 锁，避免并发发布，锁内数据库事务尽量短。

运行时 API：

- `ListTenantApplications(tenant_id)`：返回租户当前有效应用。
- `GetPublishedNavigation(application_id, locale)`：返回已发布菜单树、权限码及其 `tenant/platform` 决策作用域。
- `BatchCheckTenantApplications(tenant_id, application_ids)`：供网关或其他内部服务批量判断。
- 控制台在读取上述租户作用域 API 前，必须先调用 tenant-service 的 `POST /api/v1/tenants/select`。tenant-service 从自身事实源验证当前用户的 active membership，再以可信服务身份请求 identity-service 将 `tenant_id + membership_id` 原子写入当前会话并签发新 access token；刷新 Token 继承会话中的作用域。客户端不得自行提交 membership ID 给 identity-service。
- `GetMyNavigation` 不建议直接放在领域服务中；由 BFF 先读取有效应用/菜单，再调用 authorization-service 的批量决策，避免 application-service 逐菜单同步调用形成扇出。

服务间调用必须通过版本化 gRPC 接口。application-service 可通过 tenant-service 校验租户，也应消费租户停用/删除事件维护本地轻量投影，测试时使用进程内 fake 或 bufconn，不依赖 tenant-service 在线。

## 事件

- `application.created|updated|disabled`
- `application.menu_published`
- `tenant.application_granted|revoked|expired`

领域变更和 outbox 在同一事务提交，消费者按 event ID 幂等。tenant-service 的租户停用事件触发 grant 失效；authorization-service 可以消费菜单发布事件预热权限码或校验未知权限，但不能反向修改菜单数据。

## 何时再拆服务

出现以下真实复杂度后，从 application-service 拆出 `entitlement-service`：套餐、试用、续费、席位、功能项、配额消耗、计量或与 billing 的强关联。只有菜单开始拥有独立团队、独立发布频率，或需要支持多种前端编排产品时，才考虑拆 `navigation-service`。现在单独创建 menu-service 会增加分布式事务和发布耦合，收益不足。

## 测试边界

- 单元测试覆盖菜单树约束、循环检测、发布快照、授权有效期和乐观锁。
- Testcontainers 只启动本服务的 PostgreSQL/Redis/NATS，覆盖 Repository、事务 outbox、发布锁竞争和迁移 up/down。
- tenant/authorization 客户端使用 fake 或 bufconn；本服务的单元与集成测试不得要求其他服务运行。
- 平台 system-test 单独覆盖“创建应用 → 发布菜单 → 授权租户 → RBAC 过滤 → 返回我的导航”。
