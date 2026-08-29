# Quality Review Brief — Issue #6 计划创建向导（free/flat 价卡）

你是独立代码质量审查员（quality reviewer）。规格符合性由另一位审查员负责；你审**代码质量、正确性与回归风险**。

## 输入

- Worktree：`/Users/wuyongjun/trea/openmeter-issue-6`（分支 `codex/admin-config-06`）。审查对象 = 实现 commit（`git log` 中 `feat(admin): 计划创建向导（free/flat 价卡）`；diff 基线 a319e1b38）。
- 仓库约定：`AGENTS.md`（web 侧注意：遵循既有 features/config 结构、shadcn 组件用法、i18n 双 locale 同步、`@/` 别名 import 顺序惯例）。
- 处方 spec：`gh issue view 6 --repo 1123786563/openmeter --json body,comments`（注意：处方代码是权威——若你认为处方本身有问题，标为"处方内在缺陷"而非实现缺陷，并评估是否 Critical）。
- 实现者报告：`.superpowers/sdd/issue-6-plan-create-wizard/task-1-report.md`。

## 审查轴（每轴给证据）

1. **类型安全**：tsc 全量（`cd web && pnpm exec tsc -b --no-check? 或项目等价 typecheck`——若项目无独立 typecheck script，用 `pnpm exec tsc --noEmit -p tsconfig.json` 或 build 已含；记录所用命令与退出码）。重点：`as never` 断言点的必要性（处方模板字符串路径）、`Price` 判别联合映射穷尽性。
2. **运行时正确性**：
   - react-hook-form 动态数组：useFieldArray 与 zod superRefine 行级错误路径（`phases.${i}.duration`、`rateCards` 数组级 message）的渲染是否真能落到具体行/卡片下；
   - Select 受控（value/onValueChange）与 RHF field 的绑定正确性；billingCadence null⇄'one_time' 转换；
   - 提交后 invalidation 链（plans/plans-page/plan）能否让列表自动出现新 draft；
   - `useWatch` 的 `${pricePath}.kind` 在字段未挂载时的行为（步骤 3 才渲染 price 编辑器）；
   - 重置语义：open→true reset(EMPTY_PLAN) 是否会因 useFieldArray 产生键闪烁/状态残留。
3. **回归风险**：`git diff a319e1b38..HEAD -- web/src/api/hooks.ts web/src/features/config/plans/index.tsx` 是否纯增量（既有 usePlans/usePlansPage/usePlan/发布归档克隆与列表列/筛选零改动）；i18n 既有键零改动（只增 wizard 子树）。
4. **lint/format 基线**：`cd web && pnpm lint` exit 0；prettier 对 6 个触碰文件无新增违规（`pnpm exec prettier --check <6 files>`；既有 hooks.ts recharge hunk 违规属基线，仅确认未恶化）。
5. **安全**：无 eval/动态 import/secret；用户输入仅经 zod 校验后进 SDK 类型化调用。
6. **可维护性**：命名/结构/注释与仓库惯例一致；无死代码/未用导出（处方要求的导出除外——它们是 #7-#11 的前向契约）。

## 独立复跑（必须亲跑，给退出码）

- `cd web && pnpm lint`
- `pnpm test:e2e`（若端口残留先清理并记录；2 条冒烟通过）
- prettier check 触碰文件（如上）

## 输出

报告写到 `.superpowers/sdd/issue-6-plan-create-wizard/quality-review-report.md`：每轴结论 + 证据；发现分级 Critical / Important / Minor（Important = 会造成用户可见错误行为、类型不安全、或明确的回归风险；风格偏好归 Minor）。结尾一行裁决：`QUALITY REVIEW: PASS|FAIL (nC/nI/nM)`。若发现 Critical/Important，给**最小修复建议**（文件:行 + 改法），供控制器开修复轮。不改代码、不改 progress.md、不派生 subagent、不做 GitHub 写操作。
