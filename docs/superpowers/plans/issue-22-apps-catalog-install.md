# Plan — issue #22 应用：目录浏览与安装表单

- Issue: https://github.com/1123786563/openmeter/issues/22
  ([admin-config 22/29] 应用：目录浏览与安装表单)
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 11 后半
  （前半 #21 已合入：已装列表/卸载/Stripe 换 Key）。
- Branch codex/admin-config-22 @ base f60cb90b0；worktree
  /Users/wuyongjun/trea/openmeter-issue-22。
- Ledger: .superpowers/sdd/issue-22-apps-catalog-install/progress.md（worktree-local）。

## 范围（Scope）

- `web/src/api/query-keys.ts`：追加 `appCatalog()`。
- `web/src/api/hooks.ts`：追加 `useAppCatalog()`（v3 `api.internal.apps.listCatalog`，
  Apps 段内，模式照抄 useApps）与 `useInstallApp()`（mutation，onSuccess 失效
  `nsPrefix('apps')` 前缀——安装成功后已装列表刷新是验收项）。
- 新建 `web/src/features/config/apps/app-catalog-section.tsx`：应用目录区
  （type 徽章/name/description/capabilities 徽章/installMethods 徽章/安装按钮）。
- 新建 `web/src/features/config/apps/install-app-dialog.tsx`：安装 dialog，
  按目录项 `type` 分支：
  - stripe：名称 + API Key（密码框，zod min(1) 必填——验收项）+
    createBillingProfile 开关（默认开）；
  - sandbox / external_invoicing：仅名称（createBillingProfile 隐式 false，
    见 Ruling）。
- `web/src/features/config/apps/index.tsx`：已装列表下方挂载目录区。
- i18n：`config.apps.catalog.*`（区标题/空态/字段）与 `config.apps.install.*`
  （标题/描述/名称/API Key/开关及后果提示/校验/提交/成功 toast）子树，
  zh-CN 与 en 同构。

## 非目标（Non-goals）

- OAuth 安装流（仓库共识无 OAuth；目录项 installMethods 仅含 with_oauth2 时
  安装按钮禁用并提示不支持，不实现流程）。
- 已装列表/卸载/Stripe 换 Key（#21 已交付，零改动）。
- 安装后 default_for_capability_types 展示（#21 已在页头注释说明仅安装响应携带）。
- 后端/TypeSpec 改动。

## 已核实的契约事实（SDK dist = api/spec/packages/aip-client-javascript，2026-08-29）

- `api.internal.apps.listCatalog()` → `AppCatalogItemPagePaginatedResponse`
  `{ data: AppCatalogItem[], meta }`；`AppCatalogItem = { type: 'sandbox' |
  'stripe' | 'external_invoicing', name, description, capabilities:
  AppCapability[], installMethods: ('with_oauth2' | 'with_api_key' |
  'no_credentials_required')[] }`；`AppCapability = { type: 'report_usage' |
  'report_events' | 'calculate_tax' | 'invoice_customers' | 'collect_payments',
  key, name, description }`。
- `api.internal.apps.install(request)`：oneOf `InstallAppStripeWithApiKey
  {type:'stripe', name, createBillingProfile: boolean, apiKey}` |
  `InstallAppSandbox {type:'sandbox', name, createBillingProfile: boolean}` |
  `InstallAppExternalInvoicing {type:'external_invoicing', name,
  createBillingProfile: boolean}`。
- Ruling（SDK 事实 vs 总纲笔误）：三分支 `createBillingProfile` 均为**必填
  boolean**（总纲 `createBillingProfile?` 为笔误）。Issue 文案「stripe（名称/
  API Key/createBillingProfile 开关）、sandbox / external_invoicing（仅名称）」
  → 非 stripe 分支表单不渲染开关、提交固定 `createBillingProfile: false`。
  代价：若后端对 sandbox 也支持建档案则 UI 未暴露——与 Issue 验收一致，无碍。
- 既有 UI 事实：`config.apps.type.*`、`config.apps.capability.*`、
  `config.apps.status.*` 键已存在（#21），目录区直接复用；模式对齐
  stripe-key-dialog.tsx（Form/zod/toast/handleServerError/一次性提交）。
- 页头注释（index.tsx L25-31）已说明 default_for_capability_types 语义，不改动。

## 任务拆分（每任务独立 commit + `cd web && pnpm build && pnpm lint`）

- T1 API 层：query-keys.appCatalog + useAppCatalog + useInstallApp。
- T2 目录区 + 安装 dialog + 挂载 + i18n（zh/en 同构）。

## 测试与验收命令

- 每 commit：`cd web && pnpm build && pnpm lint`（build 含 tsr generate + tsc -b
  + vite build；routeTree 零 diff）。
- 终态追加：`pnpm test:e2e`（与并行轨串行执行，4173 端口）；
  locale parity 真实求值（zh/en 递归叶键数相等且新增键全被引用）；
  prettier：新文件 clean、修改文件差异恰为插入块。
- 浏览器走查（全端点 mock）：目录区渲染目录项三段信息；sandbox 项安装表单
  仅名称、提交 wire 体 `{type:'sandbox', name, createBillingProfile:false}`；
  stripe 项空 API Key 前端拦截（zod），填 Key 后 wire 体含 apiKey 与开关值；
  安装成功 toast + 已装列表出现新应用；installMethods 仅 with_oauth2 的项
  安装按钮禁用 + 提示。

## 全局约束

- 只动 web/ 下上述文件；三 Go 模块不可误改。
- 与并行轨 #24/#26/#28 共享面仅 hooks.ts/query-keys.ts 追加段与 locale 文件
  互不相交子树（apps vs portalTokens/billingProfiles/receivablePeriods）。
- 文案全部 i18n（zh-CN + en 同步），术语沿 CONTEXT.md 词表（应用/目录/安装）。
- 所有 API 请求经 SDK（v3 单例已注入 Bearer + X-Namespace）。
- 未经用户批准不得 push/merge/close/改 GitHub 状态。
- implementer 不得自行派生 subagent；不得跳过任务审查/修复复审/终审/台账。
