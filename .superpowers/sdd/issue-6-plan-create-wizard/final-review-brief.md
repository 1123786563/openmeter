# Final Whole-Branch Adversarial Review Brief — Issue #6 计划创建向导（free/flat 价卡）

你是最强模型全分支对抗性终审员。对象：分支 `codex/admin-config-06` 全范围 `a6ff556ef..HEAD`（基底 = main；提交链 = 计划 commit a319e1b38 + 实现 commit + 可能的修复 commit）。你代表用户做交付前最后一道独立防线——假设前序审查都有漏洞，专找它们漏掉的横切问题。

## 输入

- Worktree：`/Users/wuyongjun/trea/openmeter-issue-6`
- 权威 spec：`gh issue view 6 --repo 1123786563/openmeter --json body,comments`（评论 1 处方 + 评论 2 PriceFormValue 契约）
- 计划：`docs/superpowers/plans/issue-6-plan-create-wizard.md`
- 前序产物（全部读）：`.superpowers/sdd/issue-6-plan-create-wizard/` 下 task-1-report.md、spec-review-report.md、quality-review-report.md、browser-walkthrough-report.md（若有修复轮：修复 diff 与 scoped re-review 记录）

## 审查角度（全部覆盖，各给证据）

1. **契约完备性**：#7/#8/#9/#10/#11 依赖的导出面（priceFormSchema/PriceFormValue/toPriceInput/toPlanPhases/RateCardFormValues/EMPTY_PLAN/defaultRateCard/defaultPhase；PriceEditorProps；PlanFormWizardProps）是否稳定可用、命名与处方一致。
2. **类型与 wire 真值**：SDK d.ts 独立读出（CreatePlanRequestInput/PlanPhaseInput/RateCardInput/Price/PriceFlat）对照 toCreatePlanRequest 输出类型——穷尽、无 any 逃逸。
3. **查询语义**：invalidation 键与 query-keys.ts 实际定义一致（nsPrefix('plans')/nsPrefix('plans-page')/queryKeys.plan）。
4. **回归面**：列表页 diff 仅按钮+挂载增量；既有 hooks 零改动（除追加 useCreatePlan）；i18n 既有键零改动。
5. **跨 commit 一致性**：commit 链 subject 与 Issue 步骤一致；无夹带文件（git diff --stat 全范围 + git status 干净）。
6. **安全**：无 secret/eval/动态 import/XSS 向量；用户输入全部经 zod。
7. **证据审计**：前序报告声称的命令/日志/截图真实存在且时间线自洽（抽查 /tmp/issue6-* 日志与截图、报告与 git 对象一致性）；实现者/审查员报告间的矛盾点逐个裁定。
8. **约定合规**：AGENTS.md web 侧惯例（组件结构、i18n 同步、import 顺序）；与 #1–#5 已合入代码的风格一致性。
9. **验收标准终判**：Issue 正文两条验收标准 + 手测脚本全部有证据支撑。

## 复跑（至少）

`cd web && pnpm lint && pnpm test:e2e`（或说明为何不必/已由证据覆盖）；`git log --oneline a6ff556ef..HEAD`、`git diff --stat a6ff556ef..HEAD`。

## 输出

报告写到 `.superpowers/sdd/issue-6-plan-create-wizard/final-review-report.md`：9 角度逐项结论+证据；发现分级 Critical/Important/Minor；对每个 Minor 给「是否阻塞交付」判断。结尾一行：`FINAL REVIEW: PASS|FAIL — SAFE TO PRESENT|NOT SAFE (nC/nI/nM)`。不改代码、不改 progress.md、不派生 subagent、不做 GitHub 写操作。最终回复：裁决一行 + 最重要的 1-3 个发现。
