# Spec review — issue #9 (controller-executed, downgrade mode)

Reviewed against: issue #9 body + acceptance, issue #9 prescription comment
(per-file verbatim code), base 49d1a760b → HEAD 9f71ad1bb. Standing
isolation-downgrade (subagents denied; probes on record in round claim
`issue-2026-08-29-9-14-claim.md`); reviews run as separate controller passes
with fresh commands, logs on disk under /tmp/issue-round/.

## T1 — schema 回填与提交映射 (77be4ef48)

- 处方块逐字比对：`fromRateCardToForm` / `fromPriceToForm`（graduated/volume
  tier firstUnit=上一档 upTo+1、-1 起点映射 0、末档 ''=开放端、unitAmount
  '0' 默认、flatAmount '' 默认）/ `fromPlanToWizardValues`（导出；
  billingCadence 非 P1Y→P1M 展示降级；价目卡 cadence 非 P1M/P1Y→null）/
  `toUpsertPlanRequest`（仅 name/description/phases）全部在场且语义一致。
- SDK 契约一手核验（dist/funcs/plans.js + models/operations/plans.d.ts）：
  `UpsertPlanRequestInput{name,description?,labels?,proRatingEnabled?,phases}`；
  输出结构与之一致。
- 行为级 roundtrip 测试 6/6 PASS（t9-roundtrip-test.ts，Node 26 strip-types，
  真实模块导入；wire 级 JSON 比较规避 explicit-undefined 假阳性）。

## T2 — 向导编辑模式与详情入口 (477413166 + 9f71ad1bb)

- `useUpdatePlan`：镜像 useCreatePlan 失效面（plans / plans-page / plan(id)）；
  `UpdatePlanRequest={planId, body}`、响应解包 `Plan` —— SDK 类型一手证实。
- 向导：`plan?: Plan` 可选 prop（plans/index.tsx 无 plan 挂载点零改动兼容）；
  reset 回填；PUT 分支 `{planId, body: toUpsertPlanRequest(values)}`；
  key/币种/计费周期 disabled + immutableHint；editTitle/editSubmit/pending。
- plan-detail：仅 `plan.status === 'draft'` 且非 statusBusy 显示编辑入口；
  挂载 `<PlanFormWizard open onOpenChange plan />`。
- i18n：zh/en 各 +4 键 + toast.updated；真实模块求值 790=790 零漂移，
  common.edit 两语在场。

## 处方偏差（记录在案，均有证据）

1. create 分支直传 body（处方 4b 示例 `{body: ...}` 包装触发 TS2353；
   SDK `CreatePlanRequest` 即 body 本体；原始 #6 代码正确）。裁定：处方笔误，
   以 SDK 契约为准。
2. prettier 强制格式化项（两处 print-width 换行；plan-detail import 排序 +
   Button 换行于 review fix 提交 9f71ad1bb）。裁定：formatting-mandated。

## RULING: PASS
