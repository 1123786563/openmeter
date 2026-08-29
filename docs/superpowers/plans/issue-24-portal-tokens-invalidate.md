# Plan — issue #24 门户令牌：列表与失效

- Issue: https://github.com/1123786563/openmeter/issues/24
  ([admin-config 24/29] 门户令牌：列表与失效)
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 12 后半
  （前半 #23 已合入：发放 dialog + 一次性明文弹窗）。
- Branch codex/admin-config-24 @ base f60cb90b0；worktree
  /Users/wuyongjun/trea/openmeter-issue-24。
- Ledger: .superpowers/sdd/issue-24-portal-tokens-invalidate/progress.md（worktree-local）。

## 范围（Scope）

- `web/src/api/legacy.ts`（Portal tokens 段内追加）：
  - `listPortalTokens(limit = 100)` → `GET /v1/portal/tokens?limit=` →
    `Promise<PortalToken[]>`（**裸数组**，spec 无分页包装）；
  - `invalidatePortalTokens(body: { id?: string; subject?: string })` →
    `POST /v1/portal/tokens/invalidate` → 204 void（二选一：本 UI 按行 id 失效）。
- `web/src/api/hooks.ts`（Portal tokens 段内追加）：
  - `usePortalTokens(limit = 100)`（queryKey `queryKeys.portalTokens({ limit })`，
    已存在）；
  - `useInvalidatePortalToken()`（mutation，onSuccess 失效 `nsPrefix('portal-tokens')`，
    模式照抄 useCreatePortalToken）。
- `web/src/features/config/portal-tokens/index.tsx`：发放按钮下方追加列表
  （Table：subject / 创建时间 formatDateTime / 允许的 meters（Badge 列表，
  空=不限制）/ 状态（expired→徽章）/ 操作（失效，destructive ghost））；
  行失效 ConfirmDialog（确认文案含 subject；确认后按 `{id}` 失效 + toast）。
- i18n：`config.portalTokens.list.*`（标题/空态/字段/不限）+
  `config.portalTokens.invalidateConfirm.*` + `config.portalTokens.toast.invalidated`
  + `config.portalTokens.status.expired`，zh-CN 与 en 同构。

## 非目标（Non-goals）

- 发放/一次性明文（#23 已交付，零改动）。
- 按 subject 批量失效入口（spec 支持但 Issue 仅要求逐令牌失效；UI 按 id）。
- 列表分页（v1 无游标，limit 上限 100 一次取满）。
- 后端/TypeSpec 改动。

## 已核实的契约事实（api/openapi.yaml 8390-8520 + SDK，2026-08-29）

- `GET /api/v1/portal/tokens`：query `limit`（int 1..100，默认 25），响应
  `application/json: array of PortalToken`——**裸数组**，无分页包装。
- `PortalToken = { id, subject, expiresAt?, expired?, createdAt?, token?(仅创建
  响应), allowedMeterSlugs? }`（legacy.ts 已有接口，直接复用）。
- `POST /api/v1/portal/tokens/invalidate`：body `{ id?: string, subject?: string }`
  （description: by ID or by subject），响应 204 No Content。
- 既有 UI 事实：PageHeader actions 已挂发放按钮；issue-token-dialog /
  token-once-dialog 零改动；列表模式对齐 apps/index.tsx（Table/Badge/Skeleton/
  ConfirmDialog/toast/handleServerError）。
- `queryKeys.portalTokens(params)` 已存在（query-keys.ts:56），无需新增 key。

## 任务拆分（每任务独立 commit + `cd web && pnpm build && pnpm lint`）

- T1 API 层：legacy listPortalTokens/invalidatePortalTokens + hooks
  usePortalTokens/useInvalidatePortalToken。
- T2 列表 + 失效确认 + i18n（zh/en 同构）。

## 测试与验收命令

- 每 commit：`cd web && pnpm build && pnpm lint`（routeTree 零 diff）。
- 终态追加：`pnpm test:e2e`（与并行轨串行）；locale parity 真实求值；
  prettier 新文件 clean、修改文件差异恰为插入块。
- 浏览器走查（全端点 mock）：列表渲染 subject/创建时间/meters（空显示
  「不限制」）；失效按钮 → ConfirmDialog（含 subject）→ 确认后 wire 体
  `{id}`、toast 成功、列表刷新（失效项消失或 expired 置位——以 mock 返回为准
  如实断言）；空态文案；列表响应无 token 字段时 UI 不渲染任何明文列。

## 全局约束

- 只动 web/ 下上述文件；三 Go 模块不可误改。
- 与并行轨 #22/#26/#28 共享面仅 hooks.ts/legacy.ts 追加段与 locale 文件
  互不相交子树。
- 文案全部 i18n（zh-CN + en 同步），术语沿 CONTEXT.md 词表（门户令牌/失效）。
- 列表不含明文（Issue 明示）；失效为破坏性操作，必须 ConfirmDialog。
- 未经用户批准不得 push/merge/close/改 GitHub 状态。
- implementer 不得自行派生 subagent；不得跳过任务审查/修复复审/终审/台账。
