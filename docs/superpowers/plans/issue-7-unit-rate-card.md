# Issue #7 实施计划 — 计划创建向导：unit 用量价卡

- Issue: https://github.com/1123786563/openmeter/issues/7 `[admin-config 07/29] 计划创建向导：unit 用量价卡`
- 总计划: `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 4（后半）
- Worktree: `/Users/wuyongjun/trea/openmeter-issue-7`，分支 `codex/admin-config-07`，base `f6e767dc3`（= main，#6 已合并）

## 范围

在 #6 定义的向导契约上做受控扩展：`priceFormSchema` 增加 `{ kind:'unit', amount }`；
价目卡类型 `usage_based` 接入并强制 feature 必选；feature 下拉数据来自功能目录
（`useAllFeatures()`，#2 产出的全量列表 hook —— issue 评论 2 的数据源修正，
**不得**使用分页版 `useFeatures()`）；计费周期「一次性」选项仅对固定费价卡显示。

spec 依据：`BillingRateCard.feature`（`FeatureReference { id }`，必传 id）；
`BillingPriceUnit`（`{ type:'unit', amount: Numeric 字符串 }`）；`billing_cadence`
null=一次性「Only valid for flat prices」。

## 非目标

- 不实现阶梯价（graduated/volume）——#8 的范围。
- 不修改 `toPlanPhases`、`toCreatePlanRequest`、`defaultRateCard`、`EMPTY_PLAN`、phase/plan 层 schema。
- 不改 #6 已有契约的命名与导出（PriceFormValue 等是 #7–#11 的稳定契约）。
- 不做后端/API 改动；不改 `RATE_CARD_TYPES` 之外的 wizard 结构。

## 任务拆分（单任务）

1. `web/src/features/config/plans/plan-form-schema.ts`：
   - `priceFormSchema` 整段替换为三分支 discriminatedUnion（free/flat/unit，unit 与 flat 同样用 `AMOUNT` 正则 refine）。
   - `rateCardSchema` 整段替换：`type: z.enum(['flat_fee','usage_based'])`、新增 `featureId: z.string().optional()`、superRefine 三条规则（flat_fee 价格类型必须 free|flat；usage_based 必须 featureId 且价格必须 unit；billingCadence=null 仅 free|flat 合法），错误 path 分别定位到 `type`/`featureId`/`billingCadence`。
   - `toPriceInput` 增加 `case 'unit'`；`toRateCardInput` 输出 `feature: card.featureId ? { id: card.featureId } : undefined`、`billingCadence: card.billingCadence ?? undefined`。
2. `web/src/features/config/plans/price-editor.tsx`：按 issue 评论的完整新文件实现（kind 选择加 unit 项、unit/flat 共用金额输入、unit 标签用 `unitAmount` key）。
3. `web/src/features/config/plans/plan-form-wizard.tsx`：
   - `RATE_CARD_TYPES = ['flat_fee','usage_based']`。
   - `PhaseRateCardsSection` 拉取 `useAllFeatures()` 并下传。
   - 单卡渲染抽出 `RateCardRow` 子组件（hooks 不能写在 `fields.map` 里）：类型切换时把 price 重置为该类型合法形态（usage_based→`{kind:'unit',amount:''}`，否则 `{kind:'flat',amount:''}`）；usage_based 卡渲染 feature 必选 Select（loading 态/空态处理）；计费周期 Select 的 one_time 选项仅在 priceKind 为 free|flat 时显示。
   - 文件头 `import type { Feature } from '@openmeter/client'`（Feature 类型以 dist 实际导出为准，若命名不同以编译通过且语义正确为准记录偏差）。
4. i18n：两份 locale 的 `config.plans.wizard` 下合并新增 key（fields: unitAmount/feature/featurePlaceholder；errors: featureRequired/usagePriceKind/oneTimeFlatOnly；zh 与 en 按评论给定文案，两份结构逐键对齐）。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

浏览器手测（验收标准）：向导中把价目卡切到「用量计费」→ 出现功能下拉（数据来自功能目录全量）与单价输入；不选功能提交 → 「必须选择功能」错误定位到下拉；创建含 flat+unit 各一张价卡的计划成功；unit 卡上计费周期下拉不显示「一次性」。

## 全局约束

- 遵循仓库 AGENTS.md 与 web 既有代码风格（prettier printWidth 80、sort-imports）；不引入 any/as/@ts-ignore。
- 文案全部 i18n，zh-CN 与 en 同步；不得硬编码用户可见文案。
- 不改 routeTree.gen.ts（生成物）；不动 #6 之外的其他域文件。
- e2e 既有 2 条冒烟不得回归。
- Commit: `feat(admin): 计划向导支持 unit 用量价卡`
