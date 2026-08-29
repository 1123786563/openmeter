# Spec Review Brief — Issue #4 Task 1

你是独立 spec 审查员。审查 worktree `/Users/wuyongjun/trea/openmeter-issue-4` 分支 `codex/admin-config-04` 上 `96e1111c0..45f89e50d` 的实现是否忠实满足 Issue #4 的规范。**只读审查 + 允许重跑验证命令；禁止改任何源文件、禁止派生子代理、禁止 GitHub 写操作。**

## 规范来源（按优先级）

1. Issue 正文+评论（处方化代码）：`gh issue view 4 -R 1123786563/openmeter --json body,comments`（只读）
2. 实施计划：`docs/superpowers/plans/issue-4-plans-list-detail.md`
3. implementer 报告（含偏差清单）：`.superpowers/sdd/issue-4-plans-list-detail/task-1-report.md`

## 审查清单（逐项给证据）

1. 文件清单与评论「文件（精确路径）」节一致（8 文件 + routeTree.gen.ts 再生成；无范围外文件）。
2. hooks：`usePlansPage`/`usePlan` 与评论契约一致（queryKey、queryFn 参数形状 page/sort/filter、enabled、既有 `usePlans` 未动）。
3. query-keys：`plansPage`/`plan` 两行新增；`plans()` 保持。
4. status-badge：`tones.plan` 4 状态与评论一致；StatusBadge/EnumBadge 既有逻辑未动。
5. 列表页：7 列（name 链接/key/version/status/currency/billingCadence/createdAt）、状态筛选 Select（all+4 状态、切换重置页码）、ServerTable 分页接线（pageSize 变化回页 1）、空态。
6. 详情页：loading skeleton / notFound 兜底 / InfoRow 基本信息（key/currency/cadence/created/updated/description?） / 阶段→价目卡两级（phaseIndex·name·duration|noDuration；rate card 表 6 列；RateCardPriceSummary 5 分支 free/flat/unit/graduated|volume 折叠档数；card.currency ?? plan.currency；billingCadence ?? plan.billingCadence；feature?.id ?? '—'）。
7. 路由：index.tsx 替换 + $planId.tsx 新建；routeTree.gen.ts 差异仅 $planId 节点相关。
8. i18n：zh/en 的 config.plans 块与顶层 plan 域与评论逐键对照；两份结构一致。
9. implementer 三条偏差逐条裁定（是否必要、是否最小、是否等价）——特别是偏差 1 的 TS2322 复现验证（node_modules SDK 类型路径 api/spec/packages/aip-client-javascript/dist）。
10. 验证证据真实：自行重跑 `pnpm lint`（web/ 下）exit 0；检查 /tmp/issue4-{build,lint,e2e}.log 三份日志内容与时间线自洽。

## 输出

写入 `.superpowers/sdd/issue-4-plans-list-detail/spec-review-report.md`：每项 PASS/FAIL + 证据（文件:行号/命令输出摘录）；结论按 Critical（规范违背/功能缺失）/Important（明显缺陷）/Minor 分级计数 + 总裁定 PASS/FAIL。只写这个文件，不改 progress.md。
