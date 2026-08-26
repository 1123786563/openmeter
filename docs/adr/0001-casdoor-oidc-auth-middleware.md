# 在 fork 的 Go API 上强制 Casdoor OIDC 认证

上游 OpenMeter 开源版 API 无认证（`openapi3filter.NoopAuthenticationFunc`，v3 规范无 securitySchemes；API token 管理是云版功能）。我们决定在 fork 的 server 路由层加 Casdoor OIDC 中间件：JWKS 本地验签 access token，校验组织与读写两级角色（Viewer 仅 GET，Operator 全量），放行 `/api/v1/portal/*`（走既有 portal token）、healthz、openapi 文档与 metrics；管理端 SPA 走授权码 + PKCE。`auth.oidc.enabled` 默认开启，quickstart、e2e、seed 等非管理端环境显式关闭。

## Considered Options

- SPA + 同源反代门禁：不动 Go 代码，但 API 对任何能触达它的调用方仍然裸奔。
- Node BFF 收口：安全边界好，但引入常驻中间组件，部署复杂度上升。
- 选定改 Go 服务端：内部工具要统一身份且不留旁路，fork 已有持续改造成本。

## Consequences

- 所有直连 API 的存量调用方（脚本、内部服务）开始收到 401，需显式关闭开关或携带 token。
- 上游合并时 `openmeter/server` 与 `app/config` 存在冲突面，需在合并时人工保留。
