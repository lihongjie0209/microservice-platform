# Platform system tests

这里存放需要多个真实服务共同运行的端到端业务旅程，例如“登录 → 切换租户 → 权限决策”。

系统测试独立于各服务仓库的单元测试和 Testcontainers 集成测试，不得成为服务独立构建、测试或发布的前置条件。

`platform_test.go` 验证真实的“受限 PSK 引导注册 → JWT 登录 → 创建租户 → 建立权限 → 授权决策 → NATS 事件进入审计投影”旅程。默认访问本地 Compose 暴露的端口，也可通过 `SYSTEM_TEST_*_URL` 指向临时环境；非本地环境使用 `SYSTEM_TEST_BOOTSTRAP_PSK` 注入引导密钥。

```bash
make system-test
```
