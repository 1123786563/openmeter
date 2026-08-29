# Issue #9 — 计划编辑（draft 回填与更新）实施计划

- **Issue**: https://github.com/1123786563/openmeter/issues/9
- **处方源**: Issue #9 评论 1（逐文件完整代码，2026-08-26）；总计划 `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 4 编辑模式部分
- **Branch / Worktree**: `codex/admin-config-09` @ base `49d1a760b`；`/Users/wuyongjun/trea/openmeter-issue-9`
- **Ledger**: `.superpowers/sdd/issue-9-plan-edit/progress.md`

## 范围

对 draft 计划提供编辑：向导回填既有结构（`fromPlanToWizardValues`），PUT 提交（`toUpsertPlanRequest` → `api.plans.update`）；非 draft 计划不显示编辑入口（仅保留 #5 的「克隆新版本」）。

- Modify: `web/src/features/config/plans/plan-form-schema.ts`（`fromRateCardToForm`/`fromPriceToForm`/`fromPlanToWizardValues`/`toUpsertPlanRequest` + import 合并）
- Modify: `web/src/features/config/plans/plan-form-wizard.tsx`（`plan?: Plan` 编辑模式：reset 回填、PUT 分支提交、不可变字段禁用态、标题/按钮文案、immutableHint 提示行）
- Modify: `web/src/api/hooks.ts`（`useUpdatePlan`，invalidates `plans`/`plans-page`/`plan(id)`）
- Modify: `web/src/features/config/plans/plan-detail.tsx`（draft 编辑按钮 + 挂载 `<PlanFormWizard plan={plan}/>`）
- Modify: `web/src/i18n/locales/zh-CN.ts`、`en.ts`（`config.plans.wizard.{editTitle,editSubmit,immutableHint,toast.updated}`）

## 非目标

- 阈值/重置型等无关表单；计划内附加组件 tab（#11）；`toPriceInput` 的对外导出改造（#9 评论处方仅复用已导出的 `toPlanPhases`，不扩大导出面）
- 非 draft 计划的任何编辑入口（「已发布计划仅显示克隆入口」由 #5 既有按钮显隐规则保证）
- 后端改动（无）

## 关键契约（来自处方，字段名以 v3 SDK 类型为准）

1. **PUT 语义**：`PUT /openmeter/plans/{planId}` body=`UpsertPlanRequestInput`，仅 `name`+`phases` 必填、`description?` 可选；`key`/`currency`/`billing_cadence` 不可变（不在 schema 中）。
2. **回填映射**：`fromPlanToWizardValues(plan)`——billingCadence 非 P1M/P1Y 收敛显示为 P1M（不回传）；价卡 `featureId = card.feature?.id ?? ''`；卡周期非 P1M/P1Y 收敛为 null；阶梯价 `firstUnit` 由上一档 `upToAmount+1` 显式化（起点 -1）、`lastUnit` 空串=末档无限；`unitAmount` 缺省 '0'、`flatAmount` 缺省 ''。
3. **提交映射**：`toUpsertPlanRequest(values)` = `{ name: trim, description: trim||undefined, phases: toPlanPhases(values) }`。
4. **编辑态校验**：不拆 editSchema——不可变三字段编辑态仍渲染（disabled）且值来自服务端必过既有 zod（处方已论证无「隐藏字段占位值」路径）。
5. **hook**：`useUpdatePlan` onSuccess invalidate `nsPrefix('plans')` + `nsPrefix('plans-page')` + `queryKeys.plan(plan.id)`。

## 任务拆分

- **T1 schema 映射**：`plan-form-schema.ts` 新增四个函数 + import 合并（处方第 1/2 节完整代码）。门禁：`pnpm build`。
- **T2 编辑模式接线**：`plan-form-wizard.tsx`（处方 4a–4d）+ `hooks.ts` `useUpdatePlan` + `plan-detail.tsx` 编辑入口 + 双语 i18n。门禁：`pnpm build && pnpm lint && pnpm test:e2e`。

每任务：新 implementer subagent 实现 → 独立规格符合性审查 + 代码质量审查（Critical/Important 进入 ≤5 轮修复与 scoped re-review）。全部任务后：最强可用模型全分支审查。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- 三连门禁（e2e 冒烟须与 pristine base 同签名对照，:8888 用户 shim 环境性失败基线沿用既有裁定，非回归）
- 验收（Issue AC）：draft 计划可编辑保存、详情即时更新；已发布计划仅显示克隆入口
- 程序化走查证据：编辑模式 reset 回填路径、PUT 分支、禁用态渲染、i18n 键两份对称（en/zh 键数一致）

## 全局约束

- 文案全部 i18n（zh-CN + en 同步维护）；无后端改动；API 请求经 `@openmeter/client` SDK 单例
- 遵循仓库 Go/TS 约定与 AGENTS.md；不引入 `any`/`@ts-ignore`/`eslint-disable`
- 每任务完成即提交（commit message 见处方）；台账 `.superpowers/sdd/issue-9-plan-edit/progress.md` 逐任务记录
