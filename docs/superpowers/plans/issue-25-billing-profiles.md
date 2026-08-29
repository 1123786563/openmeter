# Plan — issue #25 账单档案：列表与创建

- Issue: https://github.com/1123786563/openmeter/issues/25
  ([admin-config 25/29] 账单档案：列表与创建)
- Prescriptive source of truth: issue #25 comment 1（860 行处方级计划，本轮已
  全文核对；/tmp/issue-25-plan.md 为转录副本）。
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 13 第一半。
- Branch codex/admin-config-25 @ base 5a4666ec7；worktree
  /Users/wuyongjun/trea/openmeter-issue-25。
- Ledger: .superpowers/sdd/issue-25-billing-profiles/progress.md（worktree-local）。

## 范围（Scope）

- `web/src/api/query-keys.ts`：追加 `billingProfiles` / `billingProfile` / `apps` 三键。
- `web/src/api/hooks.ts`：追加 `useBillingProfiles` / `useBillingProfile` /
  `useCreateBillingProfile` / `useApps`（v3 SDK）。
- 新建 `web/src/features/config/billing-profiles/billing-profile-form-dialog.tsx`
  （创建表单：name/description/supplier 嵌套〔法定名称/key/税号/账单地址 7 字段〕/
  apps 三槽位必选下拉/default 开关；提交 `workflow: {}`）。
- 新建 `web/src/features/config/billing-profiles/index.tsx`（列表页：名称/供应商/
  apps 显示名/默认徽章/创建时间；空态）。
- 替换占位路由 `web/src/routes/_authenticated/config/billing-profiles/index.tsx`。
- i18n：`config.billingProfiles` 完整子树（zh-CN/en 同构，替换 #1 的 title/description 桩）。

## 非目标（Non-goals）

- 编辑（apps 不可变禁用态）与删除 → #26。
- workflow 各组设置的编辑 UI（创建只读，提交空对象）。
- app 槽位类型过滤规则（spec 未约束；全部已装 app 入下拉，后端校验错误 toast 原文）。
- 后端/API spec 改动；`useBillingProfile` 仅随单交付供 #26 回填（本工单页面不消费）。

## 已核实的契约事实（SDK dist + 主仓源码，2026-08-29）

- `api.billing.listProfiles({ page: { number: 1, size: 100 } })` →
  `ProfilePagePaginatedResponse`（ListBillingProfilesQuery.page={size?,number?}）。
- `api.billing.getProfile({ id })` / `api.billing.createProfile(body)`；
  `CreateBillingProfileRequestInput` = { name(1-256), description?≤1024, labels?,
  supplier: Party, workflow: WorkflowInput(四组全可选→`{}` 合法), apps:
  ProfileAppReferences{tax,invoicing,payment: AppReference{id}}, default: boolean }。
- `Party` = { id?, key?, name?, taxId?: PartyTaxIdentity{code?}, addresses?:
  PartyAddresses{ billingAddress: Address } }；`Address` 七字段全可选。
- `Profile`（列表渲染源）= { id, name, description?, createdAt, updatedAt, deletedAt?,
  supplier: Party, workflow, apps: ProfileAppReferences, default }。
- **DEVIATION D-1（同 #21 D-1）**：处方中 `api.apps.list` 不存在——根 client 无
  `apps` 属性；已装应用在 `api.internal.apps.list({ page: { number: 1, size: 100 } })`
  （GET /openmeter/apps）。#21 未合入 → 本轨自行定义 `useApps`（与 #21 实现逐字
  同形，键 `apps: () => ns('apps')`，后续合并零冲突）。
- `formatDateTime`（@/lib/format）已存在；`common.cancel/confirm/submitting` 已存在。

## 任务拆分（每任务独立 commit + build/lint 门禁）

- T1 API 层：query-keys 三键 + hooks 四个（useApps 走 api.internal.apps，D-1）。
- T2 页面层：billing-profile-form-dialog.tsx + index.tsx + 路由占位替换。
- T3 i18n：zh-CN/en 完整子树（含 form.validation 文案）。

## 测试与验收命令

- 每 commit：`cd web && pnpm build && pnpm lint`。
- 终态：`pnpm build && pnpm lint && pnpm test:e2e`（e2e 与并行轨串行执行端口）。
- locale parity：zh/en 键数一致 + 新增键全部被静态引用（沿用前序轨脚本口径）。
- 浏览器走查（全端点 mock 的临时 spec）：列表渲染 → 创建表单（必填/校验/
  supplier 嵌套映射/apps 三槽位/default）→ POST wire 体 deep-equal → toast →
  列表失效重取出现新行；后端错误经 toast 原文透出。

## 全局约束

- 仓库三 Go 模块不可误改；本轨只动 web/ 下 7 个文件（见范围）。
- 与并行轨 #27 的共享面仅 query-keys.ts/hooks.ts/两 locale 文件的追加段——
  追加式、子树互不相交（billingProfiles+apps vs receivablePeriods）。
- 与未合入 #8/#13/#17/#19/#21/#23 无文件子树交集。
- 未经用户批准不得 push/merge/close/改 GitHub 状态。
- subagent 被禁 → 沿用既定降级：控制者执行实施 + 程序化三审（spec/quality/
  final），独立复跑验证、日志落盘。
