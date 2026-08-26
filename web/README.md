# OpenMeter Admin

OpenMeter 内部运营管理后台（中文界面）。基于 [shadcn-admin](https://github.com/satnaing/shadcn-admin) v2.2.1 一次性模板改造（TanStack Router + Vite + Tailwind v4 + shadcn/ui），不自研回应上游，决策背景见仓库 `docs/adr/0002-admin-console-shadcn-base.md`。

- 认证：Casdoor OIDC（授权码 + PKCE，`oidc-client-ts`）
- 国际化：i18next + react-i18next，默认 `zh-CN`，回退 `en`，语言持久化在 localStorage
- API：开发期经 Vite 代理 `/api` → OpenMeter 后端；后续切换为 api/spec 生成的 `@openmeter/client`

## 启动

仓库 Node 版本以根目录 `.nvmrc`（v26.4.0）为准。若本机缺少 pnpm 等工具，使用 Nix CI 环境（必须在仓库根执行，且 `nix develop` 命令不要并发运行）：

```bash
# 在仓库根目录
nix develop --impure .#ci -c bash -c 'cd web && pnpm install'
nix develop --impure .#ci -c bash -c 'cd web && pnpm dev'
```

本机已有匹配的 Node/pnpm 时可直接在 `web/` 下执行：

```bash
pnpm install
pnpm dev        # http://localhost:5173
pnpm build      # tsr generate && tsc -b && vite build
pnpm lint
pnpm format
```

首次运行前复制 `.env.example` 为 `.env` 并填入 Casdoor 配置。

## 环境变量

| 变量                               | 说明                                  | 默认值                                |
| ---------------------------------- | ------------------------------------- | ------------------------------------- |
| `VITE_API_BASE`                    | API 基础路径（`src/lib/api.ts` 使用） | `/api`                                |
| `VITE_CASDOOR_ISSUER`              | Casdoor 地址（OIDC authority）        | 无，必填                              |
| `VITE_CASDOOR_CLIENT_ID`           | Casdoor 应用 client id                | 无，必填                              |
| `VITE_CASDOOR_REDIRECT_URI`        | 登录回调地址                          | `http://localhost:5173/auth/callback` |
| `VITE_CASDOOR_LOGOUT_REDIRECT_URI` | 登出跳转地址                          | `http://localhost:5173/sign-in`       |

## 开发代理

`vite.config.ts` 将 `/api` 代理到 `http://127.0.0.1:8888`（`changeOrigin: true`）。本地启动 OpenMeter API（默认 8888 端口）后，前端请求 `/api/...` 即直达后端，无需为前端单独配置 CORS 或后端地址。

## 部署（Docker + nginx）

`web/Dockerfile` 为多阶段构建：`node:26-alpine` 里用 corepack 启用 pnpm、`pnpm install --frozen-lockfile` 与 `pnpm build` 产出 `dist/`；`nginx:alpine` 阶段拷入静态资源与 `deploy/default.conf.template`（nginx 镜像的 envsubst entrypoint 渲染覆盖为 `/etc/nginx/conf.d/default.conf`）。nginx 负责：

- 托管 `/usr/share/nginx/html` 静态资源，SPA fallback（`try_files $uri /index.html`）
- `location /api/` 反代到 OpenMeter 上游，`/api/...` 原样透传
- gzip、基础安全头（`X-Content-Type-Options` / `X-Frame-Options` / `Referrer-Policy`）；CSP 留给上层入口按需叠加

容器运行时环境变量：

| 变量                | 说明                                              | 默认值           |
| ------------------- | ------------------------------------------------- | ---------------- |
| `OPENMETER_UPSTREAM` | OpenMeter API 上游（`proxy_pass` 目标，host[:port]）。nginx 在启动时解析该地址：默认值需容器与名为 `openmeter` 的服务同网络（如 compose）；独立运行请显式传可解析地址（例如 `host.docker.internal:8888`） | `openmeter:8888` |

构建与运行（注意：`VITE_CASDOOR_*` 是构建期变量，需在 `docker build` 时通过 `--build-arg`/构建环境注入，或在构建前写入 `.env`；`OPENMETER_UPSTREAM` 是运行期变量）：

```bash
# 在 web/ 目录构建镜像（VITE_ 变量按部署环境注入，见下方说明）。
# 应用依赖仓库内 SDK（package.json 的 file:../api/spec/... 依赖），需以
# named build context 一并注入，web/ 仍作为构建上下文。
docker build \
  --build-context openmeter-sdk=../api/spec/packages/aip-client-javascript \
  --build-arg VITE_CASDOOR_ISSUER=https://casdoor.example.com \
  --build-arg VITE_CASDOOR_CLIENT_ID=openmeter-admin \
  --build-arg VITE_CASDOOR_REDIRECT_URI=https://admin.example.com/auth/callback \
  --build-arg VITE_CASDOOR_LOGOUT_REDIRECT_URI=https://admin.example.com/sign-in \
  -t openmeter-admin .

# 运行（上游指向名为 openmeter 的服务的 8888 端口）
docker run -p 8080:80 -e OPENMETER_UPSTREAM=openmeter:8888 openmeter-admin
```

## E2E 冒烟测试（Playwright）

`pnpm test:e2e` 运行 Playwright 冒烟用例（`e2e/smoke.spec.ts`），覆盖登录链路与关键页面渲染，不依赖真实 OpenMeter / Casdoor。`playwright.config.ts` 声明两个 webServer：

- **mock IdP**（`e2e/mock-idp.mjs`，纯 Node 实现的最小 OIDC Provider，`127.0.0.1:9999`）：提供 discovery、`/authorize`（校验 `client_id`/`redirect_uri`/`state`/`code_challenge` 后 302 回 `redirect_uri?code&state`）、`/token`（签发自签 RSA RS256 `id_token`）、`/jwks.json`。nonce 在 authorize 时与 code 建立内存映射、token 时回填进 id_token，使 `oidc-client-ts` 默认的 nonce 校验通过。
- **preview**：带 e2e 专用 `VITE_CASDOOR_*`（issuer 指向 mock IdP、redirect URI 用 preview 端口 4173）重新 `pnpm build` 后 `pnpm preview --port 4173`。

用例：

1. 登录冒烟：访问 `/` → 守卫跳 `/sign-in` → 点击「使用 Casdoor 登录」→ mock IdP 自动 302 回 `/auth/callback` → 落回 dashboard，断言标题与侧边栏。
2. 客户列表冒烟：`page.route` 拦截客户列表 API——`**/api/v3/openmeter/customers*`（v3 SDK 实际调用路径，按 SDK `listCustomersResponseWire` 的 wire 格式）与 `**/api/v1/customers**`（`api/openapi.yaml` 的 `CustomerPaginatedResponse`），返回固定 JSON；导航 `/customers` 断言表格渲染（页面实现前断言占位文案）。

CI 中由 `ci.yaml` 的 `web` job 执行 lint / build / test:e2e（含 `playwright install chromium --with-deps`）。

## 与 OpenMeter API / Casdoor 的关系

- **认证**：登录页仅保留「使用 Casdoor 登录」按钮，跳转 Casdoor 完成授权码 + PKCE 流程；回调路由 `/auth/callback` 完成会话建立后按 `redirect` 参数回跳。`_authenticated` 布局路由的 `beforeLoad` 做登录守卫；登出走 `signoutRedirect`（end-session）。
- **API 调用**：`src/lib/api.ts` 提供带 Bearer token 的 fetch 封装（401 时清除登录态），作为过渡。后续由 api/spec 生成链产出 `@openmeter/client` SDK 替换。

## 目录速览

```
src/
  components/     # 通用组件（ui/、layout/、data-table/ 等）
  features/       # 按域组织的功能模块（auth、settings、errors）
  i18n/locales/   # en / zh-CN 翻译资源
  lib/            # auth（OIDC）、api（fetch 封装）、i18n、工具函数
  routes/         # TanStack Router 文件式路由（routeTree.gen.ts 为生成物）
  stores/         # zustand（auth-store）
```
