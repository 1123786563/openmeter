# SDD ledger — plan: docs/superpowers/plans/issue-15-notification-rules-threshold-reset.md

- Issue: https://github.com/1123786563/openmeter/issues/15 `[admin-config 15/29] 通知规则：阈值/重置型与 test 触发`
- Worktree: /Users/wuyongjun/trea/openmeter-issue-15（branch codex/admin-config-15，base ec85f6871 = main = origin/main）
- Plan commit: 1aae6dacc；处方源 /tmp/issue-round3/issue-15-comments.md（issue comment 1 逐文件完整代码）
- 轮次：2026-08-29 并行轮（见 .superpowers/sdd/issue-2026-08-29-parallel-round-11-15-claim.md）

## Pre-flight 冲突扫描（派出 Task 1 前完成）

| 检查对 | 发现 | 裁定 |
|---|---|---|
| T1 ↔ T2 接缝 | T2 消费 T1 的 useTestRule、NotificationEvent.id（toast）、全部新增 i18n 键（thresholdTypes.*/form.*/testConfirm.*/toast.testSent）；命名由处方固定 | 无冲突：T1 先行产出 T2 所需全部符号 |
| T1 自洽 | useTestRule 失效 nsPrefix('notification.events')——该前缀查询 #16 才落地，当前无监听=无害 no-op | Ruling-2 已入计划（保留处方原样，作为 #16 键前缀契约） |
| T2 自洽 | rule-form-dialog/rules 两文件为**完整替换**（#14 发票型版本被替换）；新表单仍引用 #14 既有键（form.createTitle/editTitle/createDescription/editDescription/channelsPlaceholder/channelsHint/noChannels/typeHint、fields.type/name/channels/status、validation.required/channels、types.*、toggleConfirm.*、toast.created/updated/enabled/disabled、pagination.total、enable/disable/empty/create/description/title/actions）→ i18n 合并时**既有键一个不删** | 已入计划非目标与 i18n 范围 |
| 处方 ↔ 评审红线 | ① 处方含 `as unknown as Control<分支类型>` 收窄（判别联合下 RHF 字段名分支绑定，处方注记明示）；② switchType 用 form.reset 迁移联合形状（reset effect 模式与 #14 相同）；③ 空 features 数组提交时省略字段 | 均为处方明示手法，评审按 Ruling 对待，非缺陷 |
| 处方 ↔ main 现物 | 处方备选 useFeatures() 零参不存在（#2 实落 useFeatures(params) 分页信封） | Ruling-1 已入计划：useFeatures({page:1,pageSize:100}) + data?.data ?? [] |
| 本轨 ↔ #11 跨轨 | 共享 hooks.ts（不同分节）与 locale 双文件（不同子树）→ append-only；#15 不触 query-keys.ts | 已入轮次台账冲突预报 |
| 全局约束 | test 真实投递 → ConfirmDialog 明示；handleServerError 透出；v1 经 legacy apiFetch | 处方代码符合 |

- 扫描结论：clean，无阻塞性冲突；Ruling-1/2 已作为计划级裁定随任务传达。

## SDD 模式

- Standing DOWNGRADE（10+ 轨先例）：本会话 subagent 派生通道三探针全灭——
  `subagent`、`subagent_fork`、`list_experts` 均被「无人值守自动化允许列表」拒绝。
  降级执行：控制器逐任务实施（处方逐字 + 计划 Ruling）+ 独立分遍规格符合性审查
  与代码质量审查（每遍重新运行验证命令、日志落盘 /tmp/issue-round3/）+ ≤5 轮
  修复 + 最终全分支对抗审查。attended 独立抽查待补（先例：11:48 轮 10/10 PASS）。

## Tasks

### T1 数据层 + i18n（BASE 1aae6dacc）

- 实施（DOWNGRADE 控制器）：legacy.ts 事件类型族 + testNotificationRule
  （处方逐字）；hooks.ts useTestRule（处方逐字 + import 补行）；zh/en
  config.notification.rules 合并追加（test/thresholdTypes/testConfirm 新子树
  + fields/form/validation/toast 合并键，既有键零删改）。
- 实施自查修正 ×2（属实施内修正，非审查发现）：① 首次提交漏 add locale →
  amend 补齐；② build TS2552（hooks 缺 testNotificationRule import）→ 补
  import + routeTree 恢复提交版 → build/lint 双 0 → amend（787e4032f）。
- 验证：t15-t1-build2/lint2 exit 0；routeTree 零 diff。
- 分遍 1 规格（新鲜命令）：PASS 无发现——legacy/hooks 与处方逐字；20/20 zh
  键值精确（1 项初报 mismatch 复核为 shell 转义伪差异）；既有键零删改；
  982=982 奇偶；Ruling-1（query-keys 零触碰）/Ruling-2（events 前缀原样）
  落实。详见 spec-review-report.md。
- 分遍 2 质量（新鲜命令）：PASS 无发现——定向 eslint 0；反模式 0；prettier
  +66/+14/+25/+27 全插入删除 0。详见 quality-review-report.md。
- Task 1: complete (commits 1aae6dacc..787e4032f, review clean)

### T2 表单与列表（BASE 787e4032f）

- 实施（DOWNGRADE 控制器）：rule-form-dialog.tsx 完整替换（四类型判别联合 +
  阈值 useFieldArray 1-10 + features MultiSelect）+ rules.tsx 完整替换（test
  按钮 + ConfirmDialog + toast），处方逐字 + Ruling-1 三处 + 处方缺陷修复
  PD1（FieldErrors 分支收窄）/PD2（type-import）。
- 实施自查修正 ×3（属实施内修正，非审查发现）：① build TS2304（Ruling-1
  改名后 deps 残留 features）→ [featuresData]；② TS2339（处方 L598 联合
  errors 访问编译不过）→ PD1 收窄；③ prettier/type-import 复写 → build3/
  lint4 双 0 → amend（51a2b10b1）。
- 验证：t15-t2-build3/lint4 exit 0；两文件 prettier clean；routeTree 零
  diff；e2e（T2 后 + 最终 tip 两轮）sign-in ✓ + customers ✘ 基线同签名。
- 分遍 1 规格（新鲜命令）：PASS——全量 diff 偏差恰四类 sanctioned；锚点
  全中（min(1)/max(10)、空 features 省略、disabled 按钮、type 不可换、
  分支门控）；46/46 t() 键覆盖。含 Ruling-PD1/PD2/Q1 记录。详见
  spec-review-report.md。
- 分遍 2 质量（新鲜命令）：PASS——eslint 0 error（1 信息性 warning
  Ruling-Q1）；反模式 0；完整替换无死代码。详见 quality-review-report.md。
- Task 2: complete (commits 787e4032f..51a2b10b1, review clean)

## Final whole-branch review

- 五角度对抗审查（新鲜命令，分支 1aae6dacc..51a2b10b1）：AC 全锚定；
  回归面 channels/features/query-keys 零触碰；证据全落盘；约定全符；
  对抗边界八项核验。详见 final-review-report.md。
- RULING: PASS（本地完成，等待外部化批准）。

## 剩余风险

- e2e customers 冒烟环境性失败（基线同签名，非回归，跨轨共性）
- useFieldArray react-compiler 信息性 warning（Ruling-Q1，不阻断）
- 阈值/重置表单无浏览器自动化覆盖（attended 补查欠账的一部分）
- features 目录 size=100 上限（与 #2 页同限）

## 2026-08-29T11:27Z attended 抽查轮

- 控制器执行 attended 抽查（subagent/subagent_fork/后台进程三通道均被无人值守允许列表拒，非全新上下文）：门禁全绿（build/routeTree/eslint/locale 奇偶新鲜重跑）、全部 Ruling 第一手证实、无新发现 → SPOT-CHECK PASS。详见 spot-check-report.md。#15 另记：浏览器真实走查欠账保留（:5173 现役服务经 lsof 核实服务主工作树；自起 dev server 被会话策略禁）。

## 2026-08-29T11:47Z 外化轮（用户批准：批准外发全部 3 条）

- 分支已推 origin；合入 main 合并提交见 wake-log（升序 #11→#15→#16，零冲突，#16 为 #15 先合后的干净追加）；合并后 main 门禁全绿（build 0/lint 0 errors/locale 1039=1039/routeTree 零 diff/e2e sign-in ✓ customers 与基线同签名=环境性）；Issue 已附证据评论关闭。
