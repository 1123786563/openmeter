# Issue #5 实施计划：计划发布/归档/克隆新版本

- Issue: https://github.com/1123786563/openmeter/issues/5 `[admin-config 05/29] 计划：发布/归档/克隆新版本`
- 规范来源：Issue #5 正文 + 处方化评论计划（完整代码）；SDK/openapi 契约已由控制器逐项核实（见下）
- 基线：45f89e50d（= main = origin/main，含 #1–#4 成果）；worktree `/Users/wuyongjun/trea/openmeter-issue-5`，分支 `codex/admin-config-05`

## 范围（Scope）

计划生命周期状态操作（只读列表/详情由 #4 提供）：

1. **legacy 层**：`web/src/api/legacy.ts` 末尾新增 `LegacyPlan` 接口与 `clonePlanNextVersion(planIdOrKey)`（POST `/v1/plans/{id}/next`，v1 camelCase 响应）。
2. **hooks**：`web/src/api/hooks.ts` 在 `usePlan` 之后新增 `usePublishPlan` / `useArchivePlan` / `useClonePlanNext`，沿用 `useCancelSubscription` 的 invalidation 模式（`nsPrefix('plans')` + `nsPrefix('plans-page')` + `queryKeys.plan(id)`）。
3. **详情页操作区**：`web/src/features/config/plans/plan-detail.tsx` 标题行追加按钮（draft→发布、active→归档、非 draft→克隆新版本；`statusBusy` 时全部隐藏；三个操作均走 `ConfirmDialog`，`isPending` 时禁用/loading），克隆成功后 toast 新版本号并 `navigate` 到新 draft 详情。
4. **i18n**：`zh-CN.ts` / `en.ts` 的 `config.plans` 下新增 `actions` / `publishConfirm` / `archiveConfirm` / `cloneConfirm` / `toast` 键。

## 非目标（Non-goals）

- 不改后端、API spec、v3/v1 SDK 生成物。
- 不实现计划创建向导（#6–#8）、计划编辑（#9）。
- 不处理既有 prettier 违规（hooks.ts recharge hunk 等仓库预存问题，#2–#4 四次独立确认预存）。
- 不做前端预判「已有草稿」禁用逻辑（评论明确：v1 端点 4xx 错误原文 toast 透出，不预判）。

## SDK 契约核实记录（控制器，2026-08-27，基于 worktree dist 与 api/openapi.yaml）

- `api.plans.publish(request)`：`PublishPlanRequest = { planId: string }` → `PublishPlanResponse = Plan`（dist/sdk/plans.d.ts:73、dist/models/operations/plans.d.ts:43-46）。
- `api.plans.archive(request)`：`ArchivePlanRequest = { planId: string }` → `ArchivePlanResponse = Plan`（同上 :39-42）。
- `Plan.status: 'draft' | 'active' | 'archived' | 'scheduled'`（dist/models/types.d.ts:5460）——与评论按钮显隐规则（draft/active/非 draft）完全自洽。
- v1 `POST /api/v1/plans/{planIdOrKey}/next`：operationId `nextPlan`，201 → v1 `Plan`（camelCase）；描述明确「已存在 draft 或非最新发布版时报错」（api/openapi.yaml:7362-7382）——错误透传 toast 的设计依据。
- v1 `Plan` 含 id/name/key/version(int)/currency/billingCadence/status/createdAt/updatedAt（api/openapi.yaml:22777+）——评论的 `LegacyPlan` 子集合法。
- `apiFetch<T>(path, init)`（web/src/lib/api.ts:27）已有 `method` 透传；`/v1/` 路径前缀先例 `listSubjects('/v1/subjects')`。
- `nsPrefix(domain)`（hooks.ts:738）、`queryKeys.plans()/plansPage(params)/plan(id)`（query-keys.ts:24-26）均在位；`useCancelSubscription`（hooks.ts:130）为 invalidation 模板。
- `ConfirmDialog` props：title/desc/confirmText/cancelBtnText/destructive/isLoading/handleConfirm（confirm-dialog.tsx:30-44）；`config.features` 页（features/index.tsx:162-183）为 `toast.success` + `onError: handleServerError` + `ConfirmDialog` 的页面级先例。
- `common.cancel` 两份 locale 均存在（各自行 6）。
- **已知偏差风险**：(1) 评论 i18n 代码块缩进（10/12 空格）与实际文件（6/8 空格，prettier 治理）不符——按实际文件缩进适配，键名/层级不变；(2) 评论 import 清单按其自身假设列出现有导入（如 `useNavigate` 需并入现有 `@tanstack/react-router` import 行），以实际文件为准合并；(3) 三个 ConfirmDialog 须在 `if (!plan)` 早退之后的主返回分支内渲染（评论自身已注明）。

## 任务拆分（SDD Task 1，单 implementer）

按评论步骤 1→4 顺序执行：

1. legacy.ts / hooks.ts 按契约新增（评论「接口契约」节完整代码）。
2. plan-detail.tsx 操作区（import 增补 → 组件体 state/mutations/statusBusy → 标题行按钮 → 末尾三个 ConfirmDialog）。
3. i18n 两份 locale `config.plans` 下新增五组键（缩进适配实际文件）。
4. 验证与提交：`cd web && pnpm build && pnpm lint && pnpm test:e2e` 三连全绿后单提交 `feat(admin): 计划发布/归档/克隆新版本`。

## 测试与验收

- 命令：`cd web && pnpm build && pnpm lint && pnpm test:e2e`（预期三连全绿，既有 2 条冒烟不回归）。
- Issue 验收标准：
  - 对 draft 计划发布后状态变化、列表可见；
  - 克隆新版本后列表出现新 draft 计划。
- 浏览器走查场景（walkthrough 阶段）：draft 详情发布→状态徽章变、列表 draft 筛选不再出现；active 详情归档→状态变；非 draft 克隆→toast 新版本号+跳新 draft 详情+列表出现新 draft；同一 key 连续二次克隆→后端 4xx 以 toast 原文透出。
- i18n 程序化比对：两份 locale 树一致、插值变量（{{name}}/{{version}}）一致、无死键。

## 全局约束

- 遵循仓库 AGENTS.md：Go 侧不涉及；前端遵循现有模式与命名。
- 每步偏差必须记录（#2 SDK 冲突、#4 typing 自洽冲突均有先例：以仓库/SDK 真实契约为准，评论代码冲突时记录并采用可编译的等价形式）。
- implementer 不得派生 subagent；不得自行 push/merge/close 或任何 GitHub 写操作。
- 外部副作用（push/merge/close）默认禁止，完成后单独请示用户。
- 提交信息：`feat(admin): 计划发布/归档/克隆新版本`。

## 环境备注（控制器已就位）

- worktree `web/node_modules` 已 `pnpm install --frozen-lockfile`（node v26.7.0 / pnpm 11.7.0）；SDK dist 已从 #4 worktree（同基线 45f89e50d，内容等效）复制并注入 node_modules；基线 `pnpm build` 已验证绿（280ms），树净。
- 预存风险沿用 #4 台账：hooks.ts recharge hunk prettier 违规预存（勿在本任务修）；web/dist 为 gitignored 构建产物。
