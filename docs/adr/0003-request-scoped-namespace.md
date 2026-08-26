# 请求级 namespace 选择（严格 opt-in）

上游 OpenMeter 的 API 忽略请求中的 namespace 参数——所有请求经 `StaticNamespaceDecoder` 解析到静态配置的单个 default namespace，且全仓库无 namespace 列表/注册能力。我们决定在 fork 中以严格 opt-in 方式引入请求级选择：`namespace.allowlist` 为空时行为与上游完全一致；非空时请求可通过 `X-Namespace` 头选择 namespace（取值必须在 allowlist 内，default 恒允许，否则 403），并新增 `GET /api/v3/openmeter/namespaces` 返回 default 与 allowlist 的合并列表。管理端 UI 的全局 namespace 切换器以此端点为数据源、以该头为切换机制。

## Considered Options

- 前端静态配置或「环境切换器」：不动服务端，但切换器要么没有真实数据来源，要么只是切换部署实例，不满足管理端在同一实例内查看多个 namespace 数据的诉求。
- 运行时创建/删除 namespace：上游 Manager 已有跨组件 provision 能力但无注册表，补齐是更大的改造，暂不做。

## Consequences

- allowlist 非空的部署里，绕过管理端直连 API 的调用方也必须携带 `X-Namespace` 或落在 default。
- 上游合并时需保留 `namespacedriver` 的请求级 decoder 与 app 装配改动。
