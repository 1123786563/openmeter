# Issue #2 功能目录：列表与 CRUD — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: subagent-driven-development. Implement task-by-task with isolated git worktree, per-task implementer/reviewer subagents, and an SDD ledger at `.superpowers/sdd/issue-2-features-crud/progress.md`.

**Issue:** https://github.com/1123786563/openmeter/issues/2 — `[admin-config 02/29] 功能目录：列表与 CRUD`（label `ready-for-agent`，Blocked by #1 — 已完成合并关闭）
**Authoritative detail:** the issue's comment carries the fully prescriptive implementation plan (exact code for every file: query keys, hooks, form dialog, list page, route swap, i18n); `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 2 is the master-plan section. Where any wording differs, the issue comment wins; this file adds scope control only.
**Selection rationale (recorded per controller protocol):** smallest open issue number; dependency #1 satisfied (merged to main at ea31ef45a, closed 2026-08-27); `ready-for-agent` vertical slice; not a PR/duplicate/blocked/discussion item.

**Goal:** 将 `/config/features` 占位页替换为真实功能目录页：ServerTable 列表（key/名称/描述/创建时间 + 名称搜索 + 分页）、新建 dialog（name/key/description，key 正则校验、创建后不可改）、删除确认；提供跨 issue 复用的 features hooks。

**Tech Stack:** React 19 + TanStack Query + react-hook-form + zod + shadcn/ui + i18next（zh-CN/en）+ `@openmeter/client` v3 SDK。

## Verified spec facts（本计划撰写时已对照 SDK dist 与页面源码核实，实施时直接引用）

- SDK `api.features`：`list`（`ListFeaturesQuery`: `page{number,size}`、`sort{by: 'created_at', order: 'desc'}`、`filter{name:{contains}}`，响应 `FeaturePagePaginatedResponse = { data: Feature[], meta: { page: { number, size, total } } }`）、`listAll`（AsyncIterable）、`create`、`get`、`update`（**PATCH 仅 unit_cost**，SDK 文档明示 "Currently only the unit_cost field can be updated"）、`delete`（204）。
- `Feature`（camelCase）：`id, name(1-256), description?(≤1024), labels?, createdAt: Date, updatedAt: Date, deletedAt?, key, meter?: {…}, unitCost?`。
- key 约束（`ResourceKey`）：`/^[a-z0-9]+(?:_[a-z0-9]+)*$/`，1-64。
- `ConfirmDialog` props：`{ open, onOpenChange, title, desc, confirmText?, cancelBtnText?, destructive?, isLoading?, handleConfirm }`（已核对 discriminated union：无 `form` 时必传 `handleConfirm`）。
- `ServerTable` props：`{ columns, data, page, pageSize, total, onPageChange({pageIndex,pageSize}), isLoading?, isFetching?, toolbar?, emptyMessage? }`（1-based page）。
- `hooks.ts` 内部已有私有 `nsPrefix(domain)` helper（文件底部 Helpers 分节）；分节注释样式 `/* ------ */`。
- 参考模式：`web/src/features/commerce/recharge-product-form-dialog.tsx`（dialog 表单）、`web/src/features/customers/index.tsx` + `commerce/orders.tsx`（列表页）、`usePlans`（listAll 迭代模式）。
- i18n 现状：`config.features.{title,description}` 已由 #1 落位（zh「功能」/「管理功能目录与成本查询。」；en "Features"/"Manage the feature catalog and cost queries."）——本任务按 issue 评论的精确文案**替换**这两个值（zh「功能目录」/「管理可售卖与可授权的功能定义。」；en "Features"/"Manage sellable and grantable feature definitions."）并把其余新 key 合并进该子树；`common.delete` 不存在，需新增（zh「删除」/ en "Delete"）；`common.search`/`common.cancel`/`common.confirm`/`common.submitting` 均已存在。
- 占位路由现状：`web/src/routes/_authenticated/config/features/index.tsx` 渲染 `PlaceholderPage`（titleKey/descriptionKey）。

## Scope

- Modify `web/src/api/query-keys.ts`：追加 `features` / `feature` / `featureCostQuery`（第三个供 #3 落位防冲突）。
- Modify `web/src/api/hooks.ts`：追加「Features」分节（Commerce 之后、Helpers 之前）：`FeatureListParams`、`useFeatures`、`useFeature`、`useAllFeatures`、`useCreateFeature`、`useUpdateFeature`、`useDeleteFeature`（代码按 issue 评论逐字落地）。
- Create `web/src/features/config/features/feature-form-dialog.tsx`（仅创建模式；编辑模式不存在——PATCH 只能改 unit_cost，无 name/description 更新端点）。
- Create `web/src/features/config/features/index.tsx`（列表页 `FeaturesPage`：搜索 toolbar + ServerTable + 删除 ConfirmDialog）。
- Modify `web/src/routes/_authenticated/config/features/index.tsx`：占位 → 挂 `FeaturesPage`。
- Modify `web/src/i18n/locales/zh-CN.ts`、`en.ts`：`common.delete` 新增 + `config.features.*` 全量子树（按 issue 评论文案）。
- `web/src/routeTree.gen.ts` 随 `pnpm build` 的 `tsr generate` 保持不变（路由节点 #1 已建）。

## Non-goals

- 不做功能详情页、`$featureId` 路由、成本查询（`useFeatureCostQuery` 调用方）——属 #3；本任务只落 `featureCostQuery` query key 定义。
- 不做编辑 dialog（spec 无 name/description 更新端点；`useUpdateFeature` 仅按契约提供）。
- 不采集 `meter` / `unit_cost` / `labels` 表单字段。
- 不新增 e2e 用例（既有 2 条冒烟不得回归）。
- 无后端/API/SDK 改动。

## Task split

单实施任务（issue 即单一垂直切片）：步骤 1 query keys + hooks → 步骤 2 新建 dialog → 步骤 3 列表页 + 删除确认 → 步骤 4 路由替换 + i18n → 步骤 5 验证三连 → 步骤 6 commit `feat(admin): 功能目录列表与新建/删除`。

## Testing & acceptance

- `cd web && pnpm build && pnpm lint && pnpm test:e2e` 全部 exit=0（tsc -b 覆盖类型检查；既有 OIDC 登录 + 客户列表两条冒烟无回归）。
- 表单校验：key 输入 `Token-API` 报错（正则 `^[a-z0-9]+(?:_[a-z0-9]+)*$`），`token_api` 通过；name 1-256、description ≤1024。
- 浏览器手测（mock 后端或 dev server）：新建 → 列表出现；搜索生效；删除弹确认 → 确认后行消失、缓存失效刷新。
- zh/en 两份 locale 子树键结构一致。

## Global constraints

- 文案全部 i18n（zh-CN + en 同步）；术语对齐 CONTEXT.md 词表。
- 所有请求经 `@/api/client`（自动注入 Bearer + X-Namespace）；mutation onSuccess 用 `nsPrefix('features')` 失效。
- 写操作带确认弹窗；服务端错误经 `handleServerError` toast 透出。
- 遵循 AGENTS.md Go/前端约定；不引入 panic/context.Background 类反模式（前端无此问题，列出以示确认）。
- implementer 不得派生 subagent；每任务独立规格审查 + 质量审查；Critical/Important 修复 ≤5 轮 scoped 复审；全分支终审后才能宣告本地完成。
