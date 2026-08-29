# Issue #8 — 计划创建向导：阶梯价（graduated/volume）

- **Issue**: https://github.com/1123786563/openmeter/issues/8
- **分支**: `codex/admin-config-08`（worktree `/Users/wuyongjun/trea/openmeter-issue-8`，base `5a4666ec7` = main）
- **处方来源**: Issue comment 1（575 行精确到文件的代码处方）；总计划 `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 5
- **前置**: #7 已合入（`priceFormSchema` free|flat|unit、`rateCardSchema` refine、`PriceEditor`、`resetPrice`、`toPriceInput`/`toRateCardInput` 均在位，已核对）

## 范围

1. `web/src/features/config/plans/plan-form-schema.ts`
   - 新增 `NON_NEGATIVE_INT`、`tierSchema`（firstUnit/lastUnit/unitAmount/flatAmount 字符串字段，''=开区间末档/无固定费）、`TierFormValues`、`defaultTier()`
   - `priceFormSchema` 增加 `{ kind:'tiered', mode:'graduated'|'volume', tiers[] }` 判别分支（tiers min 1）
   - `rateCardSchema.superRefine`：usage_based 允许 unit|tiered（`usagePriceKind` 消息措辞更新为 i18n key 语义不变）；tiered 区间规则错误路径精确到 `price.tiers.{i}.firstUnit|lastUnit`：首档 firstUnit=0、非末档 lastUnit 必填、末档 lastUnit 必空、lastUnit≥firstUnit、lastUnit+1=下档 firstUnit（连续无重叠）
   - `toPriceInput` tiered 分支：`{ type: mode, tiers: [{ upToAmount: lastUnit||undefined, unitPrice:{type:'unit',amount}, flatPrice?:{type:'flat',amount} }] }`（firstUnit 不进 payload——spec 由 upToAmount 隐式表达区间起点）
2. `web/src/features/config/plans/price-editor.tsx`
   - 价格类型选择器加 `tiered` 项；`resetPrice` 加 tiered 分支（graduated + 单行 defaultTier）
   - `kind==='tiered'` 时渲染模式二选一下拉 + `TierEditor`（useFieldArray 行编辑器：起止量×单价×固定费 + 增删行；首档 firstUnit disabled=0、末档 lastUnit disabled+「无上限」placeholder；addTier 从上一档 lastUnit+1 接续；删行按钮至少保留一行）
3. `web/src/i18n/locales/zh-CN.ts` + `en.ts`
   - `config.plans.wizard` 下追加 `tierMode.{graduated,volume}`、`tierLastPlaceholder`、`addTier`、`fields.{tierMode,tierFirstUnit,tierLastUnit,tierUnitAmount,tierFlatAmount}`、`errors.{tiersRequired,tierBound,tierFirstFromZero,tierLastOpen,tierLastRequired,tierRange,tierGap}`（处方原文）
   - 顶层 `plan.priceType` 追加 `tiered`（zh「阶梯价」/ en「Tiered」；graduated/volume 徽章 key 已在位）

## 非目标

- 不改 `RATE_CARD_TYPES`（阶梯价是 usage_based 卡的价格形态，不是新卡类型）
- 不改 `RateCardRow` 结构（feature 必选与 one-time 限制已由 #7 的 refine/条件渲染覆盖）
- 不做 #9（PUT 编辑回填——tiered 的反向映射 fromPrice 属 #9 范围）
- 不动计划详情页徽章渲染（#4 已含 graduated/volume + tierSummary）

## 已核对的契约事实

- JS SDK（`api/spec/packages/aip-client-javascript/src/models/types.ts`）：`Price = PriceFree|PriceFlat|PriceUnit|PriceGraduated|PriceVolume`；`PriceGraduated/PriceVolume { type, tiers: PriceTier[] }`；`PriceTier { upToAmount?: string, flatPrice?: PriceFlat, unitPrice?: PriceUnit }` —— 与处方映射逐字段吻合。
- Go SDK（`api/v3/client/models_shared.go` L430-470）同构，注释明确「At least one price component (flat_price or unit_price) must be set」→ 每档必带 unitPrice、flatAmount 空则省略 flatPrice。
- 现状文件已读：`plan-form-schema.ts`（239 行）、`price-editor.tsx`（109 行）与处方「前置已提供」清单一致。

## 任务拆分（SDD）

- T1 schema：plan-form-schema.ts 五处扩展（1a-1e）——implementer 提交
- T2 编辑器：price-editor.tsx tiered 分支 + TierEditor——implementer 提交
- T3 i18n：两份 locale 追加——implementer 提交
- 每任务后：规格符合性审查（对照本计划逐条）+ 代码质量审查（AGENTS.md/前端约定）+ 修复轮 ≤5
- 全部任务后：全分支审查（最强可用模型）+ 验证三连

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- 既有 2 条冒烟不得回归；locale 两份 zh/en key 集合零漂移（parity 检查）。
- 浏览器走查（真实后端）：向导建 usage_based 卡 → 选「阶梯」→ 两档（0–99 单价 0.10 固定费 10；100–∞ 单价 0.05）→ 创建成功 → 详情徽章「阶梯（累计）」+「2 档阶梯价」→ 发布成功。
- 反向用例（错误定位到行）：下档 firstUnit=200 → 该行「与上一档不连续」；末档填截止量 → 「末档截止量必须留空」；非末档清空截止量 → 「非末档必须填写截止量」。

## 全局约束

- 文案全部 i18n（zh-CN + en 同步）；zod message 一律 i18n key 由 FieldError 翻译（FormMessage children 缺陷的既定修正模式）。
- 所有 API 经 `@openmeter/client`；字段名以 v3 SDK 为准，禁止臆造。
- 每任务验证三连；e2e 端口与并行轨道共享（4173/9999），验收 e2e 串行执行。
- 无用户批准不 push/merge/close。
