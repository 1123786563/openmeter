# Issue #6 实施计划：计划创建向导（free/flat 价卡）

- **Issue**: https://github.com/1123786563/openmeter/issues/6 `[admin-config 06/29] 计划创建向导：基础（free/flat 价卡）`
- **主计划**: `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 4（前半：free/flat；unit 属 #7，阶梯属 #8，draft 编辑回填属 #9）
- **Worktree**: `/Users/wuyongjun/trea/openmeter-issue-6`（分支 `codex/admin-config-06`，基底 `main` = `a6ff556ef`）
- **依赖**: #4（计划列表与详情）已满足 —— merged `45f89e50d`，issue closed 2026-08-27T07:24:06Z。

## 范围（Scope）

三步「新建计划」向导，完成后经 v3 `POST /openmeter/plans` 创建 draft 计划：

1. **基本信息**：name / key / currency（默认 CNY）/ billingCadence（P1M|P1Y 单选）/ description(可选)。
2. **阶段列表**：增删改（`useFieldArray`）；非最后阶段的 duration 必填，最后阶段可留空=无限期；每阶段 name/key。
3. **价目卡编辑**：每阶段 ≥1 张价目卡（增删）；字段 type（本 Issue 仅 `flat_fee` 选项）/ name / key / billingCadence（P1M|P1Y|null=一次性）/ price（判别联合 `free | flat`，kind 切换重置其余字段）。

**精确文件**（Issue 评论处方，逐字实现，含"清理未用 import"）：

- Create `web/src/features/config/plans/plan-form-schema.ts` — 向导状态 zod schema + 类型 + 提交映射
- Create `web/src/features/config/plans/price-editor.tsx` — `PriceEditor` 组件 + `FieldError` 助手
- Create `web/src/features/config/plans/plan-form-wizard.tsx` — `PlanFormWizard` 三步向导
- Modify `web/src/api/hooks.ts` — `useCreatePlan`（放在 `usePlan` 之后）
- Modify `web/src/features/config/plans/index.tsx` — PageHeader actions 挂「新建计划」按钮 + 挂载向导
- Modify `web/src/i18n/locales/zh-CN.ts`、`en.ts` — `config.plans.wizard` 子树（两 locale 结构一致）

**跨工单契约**（Issue 评论 2 强制）：`plan-form-schema.ts` 必须导出 `PriceFormValue` 类型；该文件是价格表单契约**唯一定义处**（#7/#8/#9/#10/#11 从此 import `priceFormSchema` / `PriceFormValue` / `toPriceInput`，`fromPriceToForm` 由 #9 产出），不得另建平行 schema。zod message 一律为 i18n key，由 `FieldError` 翻译。`PlanFormWizard` / `PriceEditor` 的导出名与 props 是 #7/#8/#9 的稳定契约。

## 非目标（Non-goals）

- ❌ unit 价卡与 feature 选择器（#7）；❌ 阶梯价 graduated/volume（#8）；❌ draft 编辑回填与 PUT（#9）
- ❌ usage_based 价目卡类型 UI（schema 契约含两值，UI 选项只有 flat_fee）
- ❌ 后端/API/spec 改动（纯 web 前端，走既有 v3 SDK）
- ❌ 既有 2 条 e2e 冒烟的行为变更

## 任务拆分（SDD）

- **Task 1（唯一实施任务）**：按 Issue 评论处方实现全部 6 个文件改动 + 清理标注的未用 import；本地验证三连（`pnpm build` / `pnpm lint` / `pnpm test:e2e`，在 `web/` 下）；单 commit `feat(admin): 计划创建向导（free/flat 价卡）`。
  - 实现者：新 implementer subagent（不得自行派生 subagent、不得 push、不得改 GitHub 状态）。
- **Task 1 之后（控制器编排）**：独立规格符合性审查 → 独立代码质量审查（Critical/Important → ≤5 轮修复 + scoped re-review）→ 浏览器 walkthrough（两阶段正向 + 三类反向用例）→ 最强可用模型全分支审查 → delivery gate。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e   # 三连全绿（既有 2 条冒烟不回归）
```

**浏览器手测脚本**（Issue 规定）：

- 正向：向导建两阶段计划（阶段 1 `P1M` 期限 + free 价卡；阶段 2 无期限 + flat 价卡 `99.00`）→ 提交成功 toast「计划已创建（草稿）」→ 列表出现新 draft → 详情页结构与输入一致。
- 反向 1：删除阶段 1 的期限 → 「仅最后一个阶段可以无期限」定位到该行 duration 下。
- 反向 2：清空某阶段价目卡（若可）→ 「每个阶段至少需要一张价目卡」出现在阶段卡片下方。
- 反向 3：flat 金额非法（如 `abc`）→ 行内金额错误。

**验收标准**（Issue 正文）：
- 向导可完整走通并创建含固定费价卡的计划
- 阶段与价目卡数量/期限校验生效（错误定位到具体行）

## 全局约束

- 遵循仓库 `AGENTS.md`（本仓库 Go 约定此处不涉及；web 侧遵循既有 features/config 结构、i18n 双 locale 同步、shadcn 组件用法先例）。
- 处方代码为权威 spec：文件内容、导出名、props、i18n key 逐字对齐；允许的唯一偏差是 lint/构建强制的机械修正（如未用 import 清理——处方明确要求），且必须记录。
- `RateCardInput` 不发送 `type` 字段（wire 无此字段；`type` 仅表单 UI 状态）。`billingCadence: null` → 提交 `undefined`（v3 nil 语义）。
- 金额一律字符串（`PriceFlat.amount: string`）。
- 不 push / merge / close / 改 GitHub 状态——外部化需用户显式批准。
- 提交信息：`feat(admin): 计划创建向导（free/flat 价卡）`（计划文档 commit 先行：`docs(admin): issue #6 计划创建向导实施计划`）。

## 风险与注意

- e2e 环境依赖（mock-idp :9999 / vite preview :4173）——既有端口占用需先清理（issue-1/3 先例）。
- `pnpm test:e2e` 既有 2 条冒烟必须保持绿；向导不做 e2e 自动化（Issue 未要求），走浏览器手测脚本。
- repo-wide prettier 既有红（hooks.ts recharge hunk 等，issue-3 台账已记录）不为本任务修——但本任务触碰 hooks.ts 时不得使 prettier 状况变差（增量 hunk 不新增 format 违规）。
