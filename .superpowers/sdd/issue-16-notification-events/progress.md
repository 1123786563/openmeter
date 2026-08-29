# SDD ledger — plan: docs/superpowers/plans/issue-16-notification-events.md

- Issue: https://github.com/1123786563/openmeter/issues/16 `[admin-config 16/29] 通知事件流与 resend`
- Worktree: /Users/wuyongjun/trea/openmeter-issue-16（branch codex/admin-config-16，base 51a2b10b1 = codex/admin-config-15 tip，非 main——Ruling-base）
- Plan commit: 3a3a2f2c6；处方源 .superpowers/sdd/issue-16-notification-events/prescription.md（issue #16 comment 1 逐文件完整代码）
- 轮次：2026-08-29 并行轮（见 .superpowers/sdd/issue-2026-08-29-parallel-round-16-claim.md）

## Pre-flight 冲突扫描（派出 Task 1 前完成）

| 检查对 | 发现 | 裁定 |
|---|---|---|
| T1 ↔ T2 接缝 | T2 消费 T1 的 useNotificationEvents/useResendEvent 与全部 events.* i18n 新键；命名由处方固定；T2 另用 #15 既有 rules.types.* 键（base 已在） | 无冲突：T1 先行产出 T2 所需全部符号 |
| T1 自洽 | query key `ns('notification.events', params)` 与 useResendEvent 失效前缀 nsPrefix('notification.events') 同域（#15 useTestRule 预热的契约续约）；i18n 为子树替换（Ruling-i18n），title 值不变 → 占位路由在 T1 后仍可解析 | Ruling-prefix 已入计划 |
| T2 自洽 | 路由为占位替换（路径不变 → routeTree 零 diff 判据成立）；页面 import getNotificationEvent（T1 产出）+ #15 类型族（base 已在）；DELIVERY_STATE_CLASS 五键 = NotificationEventDeliveryState 五值（实测 L425） | 无缺口 |
| 处方 ↔ 评审红线 | ① events.tsx 直接 import legacy 的 getNotificationEvent（非 hook，处方明示，行内单条刷新用）；② resend 成功 setTimeout(()=>refetch(),1500)（202 异步受理，处方明示）；③ 规则/渠道下拉 pageSize:100（Ruling-3）；④ Record<DeliveryState,string> Tailwind 类映射键完整 | 均为处方明示手法/已核实完整，评审按 Ruling 对待，非缺陷 |
| 处方 ↔ base 现物 | events 路由与 i18n events.title/description 占位已存在（脚手架）→ 处方 Create-or-Modify 落地为替换；NotificationEventType 四值 = rules.types.* 四键一一对应（实测） | Ruling-i18n 已入计划 |
| 本轨 ↔ 其他轨 | #11/#15 分支冻结（本地完成）；本轨唯一活动轨；外部化并集预案入轮次台账 | 无并行冲突 |
| 全局约束 | resend 真实投递 → ConfirmDialog 明示 + toast 受理提示；handleServerError 透出；v1 经 legacy apiFetch | 处方代码符合 |

- 扫描结论：clean，无阻塞性冲突；Ruling-base/i18n/prefix/3 已作为计划级裁定随任务传达。

## SDD 模式

- Standing DOWNGRADE（11+ 轨先例）：本会话三探测全灭——`subagent`（工具不在
  无人值守自动化允许列表）、`subagent_fork`（同上）、bash 后台进程
  （「无人值守运行不允许启动后台进程」）。降级执行：控制器逐任务实施
  （处方逐字 + 计划 Ruling）+ 独立分遍规格符合性审查与代码质量审查（每遍
  重新运行验证命令、日志落盘）+ ≤5 轮修复 + 最终全分支对抗审查。
  attended 独立抽查欠账沿用（见 #11/#15 台账）。

## Tasks

### T1 数据层 + i18n（BASE 3a3a2f2c6）

- 实施（DOWNGRADE 控制器）：legacy.ts events 段 + query-keys notificationEvents
  + hooks events 段（imports 三项字母序插入）+ zh/en events 子树替换
  （Ruling-i18n：占位 description 按处方新文案，title 值不变）。
- 环境事项（非代码）：fresh worktree 缺 aip-client-javascript 构建产物 →
  于 worktree 内 tsc 构建 + 重建 .pnpm file: 依赖条目（wt15 同法先例）；
  pnpm --silent 幂等安装不会重建已删 .pnpm 条目（记录）。locale 结构 diff
  有 3 条 base 既有伪差（single/multiple、'Format: 正则伪命中、
  ready/unauthorized），base 实测同在 → 非本轨引入。
- 实施自查修正 ×3（属实施内修正，非审查发现）：① hooks import 误加
  getNotificationEvent（处方未列，未用）→ 移除；② prettier --write 重排
  8 处既有块（hooks 4 处调用折叠 + zh/en 长行重折）→ 全部回退，仅保留
  新增内容 prettier 合规（Ruling-P1，见下）；③ build 首跑 TS2307/TS7006
  为 SDK 链接缺失的下游症状，环境修复后消失。
- **Ruling-P1（prettier 范围）**：base 51a2b10b1 的 hooks/zh-CN/en 三文件
  本就整体非 prettier-clean（--check 失败）。本轨纪律=新增内容 prettier
  合规、既有块零重排；代价：三文件整体 --check 仍失败（base 既有债）。
- 验证：t16-t1-build3/lint3 exit 0；routeTree 零 diff；locale 结构 diff 与
  base 完全同构（零新增伪差）；新增区与 prettier --write 产物逐字节一致。
- 分遍 1 规格（新鲜命令）：PASS——legacy/hooks/qk/i18n 四块与处方空白归一化
  IDENTICAL（偏差恰两类：prettier 折行 ×6、import 插入位；均获授权）；
  删除行恰 2 条；锚点全中；Ruling-prefix/i18n 落实。详见
  spec-review-report-t1.md。
- 分遍 2 质量（新鲜命令）：PASS——定向 eslint 0/0；反模式 0；9 导出全为
  契约产物无死代码；diff 202+/2-。详见 quality-review-report-t1.md。
- Task 1: complete (commits 3a3a2f2c6..eab5a1d67, review clean)

### T2 页面与路由（BASE = eab5a1d67）

- 实施（DOWNGRADE 控制器）：events.tsx 新建（处方步骤 4 逐字 → prettier
  归一）+ 路由占位替换（处方步骤 5 逐字）。
- 处方缺陷修复 **Ruling-PD1**：处方 useState 惰性表达式 Date.now 触发
  react-hooks/purity error → lazy initializer 等价修复（首渲染求值一次）。
- 实施自查修正 ×1（实施内）：首跑 prettier --check 不过 → --write 归一
  （import 排序/tailwind class 排序/尾逗号，皆 formatter 强制）。
- 验证：t16-t2-build2/lint2 exit 0；定向 eslint 0；routeTree 零 diff；
  e2e sign-in ✓ + customers ✘ 与 base 同机新鲜重跑同签名（环境性非回归）。
- 分遍 1 规格（新鲜命令）：PASS——路由归一化 IDENTICAL；events.tsx 全差异
  枚举恰五类（4 类 formatter 强制 + PD1 授权）；静态 27 t() 键 + 动态
  types.* 四值全覆盖；AC 双锚点命中。详见 spec-review-report-t2.md。
- 分遍 2 质量（新鲜命令）：PASS——eslint 0/0；反模式 0；列表键/useMemo/
  四态完整；无死代码。详见 quality-review-report-t2.md。
- Task 2: complete (commits eab5a1d67..5f9e18274, review clean)

## Final whole-branch review

- 五角度对抗审查（新鲜命令，分支 3a3a2f2c6..5f9e18274，7 文件 779+/9-）：
  AC 全锚定；回归面 rules/channels/addons/plans 零触碰（删除行恰 9 行全
  授权）；证据全落盘（final build/lint 双 0、locale 伪差 md5 与 base 一致
  50b63752、events 子树 zh/en 同 36 行）；约定全符（处方逐字纪律 + 提交
  规范）；对抗边界八项核验。routeTree 瞬态重排事故一次（#15 同款）已恢复
  并双检。详见 final-review-report.md。
- RULING: PASS（本地完成，等待外部化批准）。

## 剩余风险

- e2e customers 冒烟环境性失败（基线同签名，非回归，跨轨共性）
- 浏览器真实走查（过滤收敛/行展开/重发状态流转/后端拒绝 toast）无自动化
  覆盖——归 #29 与 attended 补查欠账（与 #11/#15 同）
- 规则/渠道下拉 >100 条不全（Ruling-3 代价，与 #14/#15 页同限）
- 外部化待用户批准；顺序 #11→#15→#16（本轨含 #15 提交）；#11↔#16
  hooks/query-keys/locale 同锚点 append 并集预案已入轮次台账

## 2026-08-29T11:27Z attended 抽查轮

- 控制器执行 attended 抽查（subagent/subagent_fork/后台进程三通道均被无人值守允许列表拒，非全新上下文）：门禁全绿（build/routeTree/eslint/locale 奇偶新鲜重跑）、全部 Ruling 第一手证实、无新发现 → SPOT-CHECK PASS。详见 spot-check-report.md。#15 另记：浏览器真实走查欠账保留（:5173 现役服务经 lsof 核实服务主工作树；自起 dev server 被会话策略禁）。

## 2026-08-29T11:47Z 外化轮（用户批准：批准外发全部 3 条）

- 分支已推 origin；合入 main 合并提交见 wake-log（升序 #11→#15→#16，零冲突，#16 为 #15 先合后的干净追加）；合并后 main 门禁全绿（build 0/lint 0 errors/locale 1039=1039/routeTree 零 diff/e2e sign-in ✓ customers 与基线同签名=环境性）；Issue 已附证据评论关闭。
