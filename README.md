# Microservice Platform

微服务平台的本地工作区与架构入口。业务服务、平台服务、契约和部署配置保持独立 Git 仓库；本目录负责统一文档、本地编排和跨仓库开发约定。

## 建议目录结构

```text
microservice-platform/
├── docs/                    # 架构、规范和服务目录
├── services/                # 独立服务仓库的本地检出目录
├── contracts/               # Proto 契约仓库的本地检出目录
├── libraries/               # 公共 Go SDK 的本地检出目录
├── deploy/                  # Helm/GitOps 仓库的本地检出目录
└── environments/            # 本地开发环境，不保存生产密钥
```

## 第一阶段目标

先建设最小平台闭环：身份认证、权限、租户、审计、配置、通知和文件服务；业务服务通过统一契约和 SDK 接入。工作流、搜索、规则引擎等能力等到出现明确业务需求后再独立建设。

详细规划见 [平台基础服务规划](docs/platform-services.md)，已交付能力和审计证据见 [P0 完成矩阵](docs/completion-matrix.md)。

本地默认使用一个 PostgreSQL `platform` 数据库，并以独立角色和 `identity`、`tenant`、`authorization` schema 隔离服务数据。开发基础设施见 [本地环境](environments/local/README.md)。

## 常用命令

所有跨仓库命令由根目录 Makefile 统一入口管理，工具安装到工作区 `.tools/bin` 并固定版本：

```bash
make help
make bootstrap
make contracts
make build
make test
make test-integration
make lint
make swagger-check
make verify
make infra-up
make infra-down
make dev-up
make system-test

# 只操作一个服务
make service-run SERVICE=tenant-service
make service-test SERVICE=tenant-service
make service-test-integration SERVICE=tenant-service
make service-migrate-up SERVICE=tenant-service
```

每个独立服务仓库仍提供 `make build`、`make test-race`、`make test-integration`、`make migrate-up` 和 `make swagger-check`，离开总工作区也可以独立开发和验证。
