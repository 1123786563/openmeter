# Spec Review Brief — Issue #6 计划创建向导（free/flat 价卡）

你是独立规格符合性审查员（spec reviewer）。你的唯一问题：**实现是否逐字符合 Issue #6 的处方，不多不少？**

## 输入

- Worktree：`/Users/wuyongjun/trea/openmeter-issue-6`（分支 `codex/admin-config-06`）。实现 commit = `git log` 中 subject 为 `feat(admin): 计划创建向导（free/flat 价卡）` 的那个（基底 = 计划 commit a319e1b38，其父 = a6ff556ef = main）。
- 权威 spec：`gh issue view 6 --repo 1123786563/openmeter --json body,comments`（评论 1 = 逐文件完整代码处方；评论 2 = PriceFormValue 导出契约）。
- 实施计划（范围/非目标）：`docs/superpowers/plans/issue-6-plan-create-wizard.md`。
- 实现者报告：`.superpowers/sdd/issue-6-plan-create-wizard/task-1-report.md`。

## 审查清单（逐项给证据）

1. **文件集精确**：`git diff --stat a319e1b38..HEAD`（实现 commit）= 恰好 6 个文件（3 create + 3 modify），无范围蔓延、无遗漏。
2. **plan-form-schema.ts**：导出全集（priceFormSchema/PriceFormValue/rateCardSchema/RateCardFormValues/phaseSchema/PhaseFormValues/phasesSchema/planWizardSchema/PlanWizardValues/defaultRateCard/defaultPhase/EMPTY_PLAN/toPlanPhases/toCreatePlanRequest）；正则与 zod 结构、superRefine 的 path 定位（`[index,'duration']`）、i18n-key message、null→undefined 映射、trim 语义逐字对齐处方。
3. **price-editor.tsx**：PriceEditorProps 契约（control/phaseIndex/cardIndex/currency）、useWatch kind、kind 切换重置（free→{kind:'free'}，flat→{kind:'flat',amount:''}）、FieldError 翻译行为、flat 才渲染 amount。
4. **plan-form-wizard.tsx**：STEPS 三步、zodResolver+EMPTY_PLAN+mode:'onChange'、open→reset+回 basics、每步 trigger 的字段集（basics: name/key/currency/billingCadence；phases: 'phases'）、PhaseRateCardsSection 抽组件（hooks 不进循环）、最后阶段 durationHint、增删禁用条件（单阶段/单价卡不可删）、提交按钮 disabled=isPending、RATE_CARD_TYPES=['flat_fee']。
5. **hooks.ts**：useCreatePlan 在 usePlan 之后；mutationFn 签名、三处 invalidation（plans/plans-page/plan(plan.id)）逐字对齐。
6. **index.tsx**：PageHeader actions=新建按钮（Plus 图标 + wizard.createTitle 文案）、createOpen state、向导挂载在 JSX 末尾；import 精确（Plus/useState/PlanFormWizard/Button）。
7. **i18n**：zh/en 两 locale 的 wizard 子树键集**完全一致**（结构对照，程序化验证如 AST/键计数），文案与处方逐字一致（zh 中文/en 英文）；插入位置在 config.plans 对象内。
8. **验收标准映射**（Issue 正文）：向导可完整走通创建含固定费价卡计划（代码路径可静态证实：createPlan.mutate → toast → 关闭）；数量/期限校验错误定位到具体行（superRefine path + FieldError 行内渲染）。
9. **偏差裁定**：实现者报告的每条偏差 vs 处方允许范围（未用 import 清理 / lint 机械修正）；处方内代码引用的既有键（plan.priceType.free/flat、config.plans.detail.cadence、common.back/next/submitting）真实存在。
10. **非目标遵守**：无 unit/阶梯/usage_based UI、无 PUT、无后端改动、e2e 冒烟无行为改动。

## 复跑（廉价验证）

`cd web && pnpm lint`（exit 0）；抽查 /tmp/issue6-*.log 三份日志真实且绿（时间戳/内容与报告一致）。

## 输出

报告写到 `.superpowers/sdd/issue-6-plan-create-wizard/spec-review-report.md`：清单 10 项逐项 PASS/FAIL + 证据（文件:行 或命令输出摘录）；发现分级 Critical（处方违背/验收不满足）/ Important（契约破坏/明显功能缺陷）/ Minor（风格/文档）。结尾一行裁决：`SPEC REVIEW: PASS|FAIL (0C/0I/nM)`。不改任何代码、不改 progress.md、不派生 subagent、不做 GitHub 写操作。
