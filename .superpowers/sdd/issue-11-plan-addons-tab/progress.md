# SDD ledger — plan: docs/superpowers/plans/issue-11-plan-addons-tab.md

- Issue: https://github.com/1123786563/openmeter/issues/11 `[admin-config 11/29] 附加组件：计划内管理 tab`
- Worktree: /Users/wuyongjun/trea/openmeter-issue-11（branch codex/admin-config-11，base ec85f6871 = main = origin/main）
- Plan commit: 6aaa5b916；处方源 /tmp/issue-round3/issue-11-comments.md（issue comment 1 逐文件完整代码）
- 轮次：2026-08-29 并行轮（见 .superpowers/sdd/issue-2026-08-29-parallel-round-11-15-claim.md）

## Pre-flight 冲突扫描（派出 Task 1 前完成）

| 检查对 | 发现 | 裁定 |
|---|---|---|
| T1 ↔ T2 接缝 | T2 组件消费 T1 的四 hooks 与 i18n 键；命名由处方固定（usePlanAddons/useCreatePlanAddon/useUpdatePlanAddon/useDeletePlanAddon；config.planAddons.*、config.planDetail.tabs.*） | 无冲突：T1 先行产出 T2 所需全部符号 |
| T1 自洽 | hooks 引用 queryKeys.planAddons——main 缺失，T1 内一并补入（处方「若缺失则补」条件成立） | Ruling-2 已入计划 |
| T2 自洽 | ServerTable props（server-table.tsx L25-40）与处方用法逐项吻合；plan-detail 现无 Tabs（#4 产物），处方「包 Tabs + overview 原样移入」适用 | 无冲突 |
| 处方 ↔ 评审红线 | 处方 Consumes 的 useAllAddons 在 main 不存在（#10 实落 useAddons，分页信封） | Ruling-1 已入计划：消费 useAddons() + data?.data ?? [] 扁平化，不新增 useAllAddons |
| 本轨 ↔ #15 跨轨 | 共享 hooks.ts（不同分节追加）与 locale 双文件（不同子树追加）→ append-only | 已入轮次台账冲突预报，外部化并集解决 |
| 全局约束 | 写操作 ConfirmDialog、i18n 双语、SDK 直连、无臆造字段——处方代码均符合 | 无冲突 |

- 扫描结论：clean，无阻塞性冲突；Ruling-1/2 已作为计划级裁定随任务传达。

## SDD 模式

- Standing DOWNGRADE（10+ 轨先例，见 issue-2026-08-29-parallel-round-10-18-20-30-claim.md）：
  本会话 subagent 派生通道三探针全灭——`subagent`、`subagent_fork`、`list_experts`
  均被「无人值守自动化允许列表」拒绝（错误原文见控制器转录）。降级执行：控制器
  逐任务实施（处方逐字 + 计划 Ruling）+ 独立分遍规格符合性审查与代码质量审查
  （每遍重新运行验证命令、日志落盘 /tmp/issue-round3/）+ ≤5 轮修复 + 最终全分支
  对抗审查。attended 独立抽查待未来 subagent 可用会话补做（先例：2026-08-29
  11:48 抽查轮 10/10 PASS）。

## Tasks

### T1 数据层 + i18n（BASE 6aaa5b916）

- 实施（DOWNGRADE 控制器）：query-keys 补 planAddons（Ruling-2）；hooks.ts 追加
  Plan addons 分节（处方逐字）；zh/en 追加 config.planDetail.tabs +
  config.planAddons（处方键集，zh 逐字/en 同构直译）。
- 验证：build exit 0（t11-t1-build.log）；lint exit 0（t11-t1-lint.log）；
  routeTree 零 diff。
- 分遍 1 规格（新鲜命令）：PASS 无发现——hooks 与处方逐字一致（仅节尾 1 空行
  差）；i18n zh 块逐字（去缩进）；zh=en=993 键零差集；planAddons 27=27；
  4 文件纯追加 0 删除。详见 spec-review-report.md。
- 分遍 2 质量（新鲜命令）：PASS 无发现——定向 eslint 0；新增行反模式 0；
  prettier 基线对照 +65/+2/+44/+47 全为插入、删除 0。详见 quality-review-report.md。
- Task 1: complete (commits 6aaa5b916..bd942b64c, review clean)

### T2 组件层（BASE bd942b64c）

- 实施（DOWNGRADE 控制器）：新建 plan-addon-form-dialog.tsx（325 行）+
  plan-addons-tab.tsx（195 行）= 处方逐字 + Ruling-1 适配（useAddons 信封）；
  plan-detail.tsx 包 Tabs（overview 原样移入 + addons tab 挂 PlanAddonsTab）。
- 实施自查：新文件 prettier 不 clean → prettier --write + 重验 build/lint 双 0
  → amend（26eeb9e5a）。
- 验证：t11-t2-build2/lint2 exit 0；routeTree 零 diff；e2e sign-in ✓ +
  customers ✘ 与 pristine 基线同签名（环境性非回归）。
- 分遍 1 规格（新鲜命令）：PASS 无发现——两组件与处方 diff 仅 3 处 Ruling-1
  适配+窗口行；plan-detail 内容等价（prettier 两侧对照 +114/-70 恰为包裹+重
  缩进）；34/34 t() 键覆盖。详见 spec-review-report.md。
- 分遍 2 质量（新鲜命令）：PASS 无发现——定向 eslint 0；反模式 0；新文件
  prettier clean。详见 quality-review-report.md。
- Task 2: complete (commits bd942b64c..26eeb9e5a, review clean, 零修复轮)

## Final whole-branch review

- 五角度对抗审查（新鲜命令）：规格追溯 AC 全锚定；回归面 #10 零触碰 +
  plan-detail 等价性 prettier 证明；证据审计日志全落盘（build/lint 双 0 ×
  2 轮、e2e 分支 tip 与基线同签名、eslint 定向 0、locale 993=993）；约定
  全符；对抗边界六项核验（single 省略 maxQuantity、编辑锁 addon、空阶段
  拦截、id 退化、分页重置、非法数量拦截）。ConfirmDialog props 契约实测
  吻合。详见 final-review-report.md。
- RULING: PASS（本地完成，等待外部化批准）。

## 剩余风险

- e2e customers 冒烟环境性失败（基线同签名，非回归，跨轨共性）
- 后端对 plan-addon 状态机/重复挂载的裁决由服务端执行，UI 不预判
- addons 首页 size=100 上限（Ruling-1 代价，与 #10 页同限）

## 2026-08-29T11:27Z attended 抽查轮

- 控制器执行 attended 抽查（subagent/subagent_fork/后台进程三通道均被无人值守允许列表拒，非全新上下文）：门禁全绿（build/routeTree/eslint/locale 奇偶新鲜重跑）、全部 Ruling 第一手证实、无新发现 → SPOT-CHECK PASS。详见 spot-check-report.md。#15 另记：浏览器真实走查欠账保留（:5173 现役服务经 lsof 核实服务主工作树；自起 dev server 被会话策略禁）。

## 2026-08-29T11:47Z 外化轮（用户批准：批准外发全部 3 条）

- 分支已推 origin；合入 main 合并提交见 wake-log（升序 #11→#15→#16，零冲突，#16 为 #15 先合后的干净追加）；合并后 main 门禁全绿（build 0/lint 0 errors/locale 1039=1039/routeTree 零 diff/e2e sign-in ✓ customers 与基线同签名=环境性）；Issue 已附证据评论关闭。
