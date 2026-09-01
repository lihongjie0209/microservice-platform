# APISIX 网关、域名与证书规范

## 结论

平台外部流量统一由 Apache APISIX 承接，不开发自研反向代理。Kubernetes 中使用 APISIX Ingress Controller 管理 `ApisixRoute`，并直接跟踪 Service/EndpointSlice 完成实例自动发现和负载均衡。`service-registry-service` 继续负责应用层元数据、动态 Provider 和非 Kubernetes 调用方发现，两套发现机制不互相替代。

服务必须显式设置 `gateway.enabled=true` 才会生成外部路由；仅注册到服务注册中心、仅创建 Kubernetes Service，均不会自动暴露到公网。

## 域名规范

`baseDomain` 是平台级必填配置，例如 `aaa.com`。生产环境省略环境标签，其他环境把稳定的短环境名放在服务名和主域名之间：

| 环境 | Kubernetes Namespace | Ingress Class | 服务域名示例 | 通配符证书 |
| --- | --- | --- | --- | --- |
| development | `platform-development` | `apisix-dev` | `identity-service.dev.aaa.com` | `*.dev.aaa.com` |
| testing | `platform-testing` | `apisix-test` | `identity-service.test.aaa.com` | `*.test.aaa.com` |
| staging | `platform-staging` | `apisix-staging` | `identity-service.staging.aaa.com` | `*.staging.aaa.com` |
| production | `platform-production` | `apisix-prod` | `identity-service.aaa.com` | `*.aaa.com` |

`*.aaa.com` 只匹配一个 DNS 标签，不能覆盖 `identity-service.dev.aaa.com`，因此证书必须按环境分别签发。环境短名是稳定外部标识，不直接复用可能变化的 Git 分支名。

推荐为每个环境配置一个 DNS wildcard 记录，指向该环境 APISIX LoadBalancer：

```text
*.dev.aaa.com       -> development APISIX
*.test.aaa.com      -> testing APISIX
*.staging.aaa.com   -> staging APISIX
*.aaa.com           -> production APISIX
```

## 环境隔离

- 生产与非生产优先使用独立 Kubernetes 集群；资源有限时至少使用独立 Namespace、APISIX release、IngressClass、LoadBalancer、证书 Secret、DNS 凭据和 NetworkPolicy。
- 每个环境使用独立 ZeroSSL ACME 账号密钥/EAB Secret。生产 EAB 和 DNS 写权限不得挂载到非生产 Namespace。
- APISIX Admin API 仅在集群内开放，Admin Key 来自 ExternalSecret，不进入 Git。每个环境使用不同 Admin Key。
- JWT issuer/audience、PSK、Redis key prefix、NATS stream/consumer、PostgreSQL database/schema 和对象存储 bucket 均按环境隔离。
- 路由只能引用同一 Namespace 的 Service；跨环境调用必须通过明确批准的内部接口和 NetworkPolicy，禁止利用公网域名绕过隔离。

## ZeroSSL 自动证书

使用 cert-manager 的 `ClusterIssuer`/`Issuer` 对接 ZeroSSL ACME 服务。ZeroSSL 要求 External Account Binding（EAB）；EAB key ID、HMAC key 和 DNS Provider 凭据由 External Secrets/Vault 注入。

通配符证书必须使用 DNS-01。每个环境创建一张包含环境入口和 wildcard 的 `Certificate`，例如开发环境：

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: platform-gateway-tls
  namespace: platform-development
spec:
  secretName: platform-gateway-tls
  issuerRef:
    kind: ClusterIssuer
    name: zerossl-development
  dnsNames:
    - dev.aaa.com
    - '*.dev.aaa.com'
```

DNS Provider 必须按实际供应商配置 cert-manager DNS-01 solver。平台不在仓库中保存 ZeroSSL EAB、DNS API Token 或生成后的私钥。

## 路由与安全边界

- 对外 HTTP 业务接口继续使用 POST + JSON；`/live`、`/ready`、JWKS、OpenAPI 等标准端点按各自协议保留 GET。
- APISIX 负责 TLS、Host/Path 路由、安全响应头、外部限流、请求体上限、真实客户端 IP 保留和灰度流量；按服务白名单生成的 CORS 策略继续由后端统一中间件执行。
- 服务自身仍验证 JWT/PSK、权限码、租户上下文和幂等键。网关认证属于第一道防线，不能成为服务绕过认证的理由。
- 默认只暴露前端需要的 HTTP 端口。gRPC 端口保持 ClusterIP 内网访问；确需外部 gRPC 时单独评审 TLS/mTLS、域名和 L4/L7 路由。
- 自动生成的 Host 为 `<service>.<environment-domain>`，允许通过 `gateway.hostname` 显式覆盖，但禁止一个 Host 同时属于多个环境。

共享 `ApisixRoute` 默认启用三项边缘保护：`client-control` 将请求体限制为 1 MiB，`limit-req` 按 APISIX 看到的 `remote_addr` 执行每秒 100 请求、burst 50 的单实例入口限流，`response-rewrite` 写入 `nosniff`、拒绝 iframe、严格 Referrer Policy 和一年 HSTS。服务仍保留自身的 Redis 分布式业务限流、Content-Type/CORS 校验和安全响应头作为纵深防御。生产 LoadBalancer 必须保留真实来源地址，只有明确配置可信代理链后才能改用转发头作为限流键。

## 前端入口

平台控制台默认使用 `console.<environment-domain>`，统一 Swagger 使用 `swagger-service.<environment-domain>`。业务服务可保留独立子域名以便 API 调试和第三方集成；浏览器页面优先通过控制台 BFF/明确的服务 API 调用，避免把内部 gRPC 或管理端口暴露到公网。
