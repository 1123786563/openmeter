# Issue #16 独立抽查报告（attended 轮 2026-08-29T11:27Z 锁）

- 抽查性质：控制器执行的 attended 抽查（三派生通道被无人值守允许列表拒绝，非全新
  上下文独立审查——欠账部分保留）。
- 工作树 /Users/wuyongjun/trea/openmeter-issue-16 @ 5f9e18274（=台账 tip），受控
  文件改动=0；`merge-base --is-ancestor 51a2b10b1 HEAD` 成立（=Ruling-base 串行链）。

## 门禁证据（本轮新鲜重跑）

| 门禁 | 命令 | 结果 |
|---|---|---|
| build | `pnpm --silent build`（web/） | exit 0 |
| routeTree | build 后 sleep 3 再 `git diff --exit-code -- web/src/routeTree.gen.ts` | exit 0（零 diff，无瞬态重排） |
| eslint（改动面 7 文件） | `pnpm exec eslint <files>` | exit 0，0 problems |
| locale 奇偶（分支 tip=#15+#16） | node 真实求值 | en=1010 zh=1010 onlyEn=0 onlyZh=0（982+28 events 键） |

## Ruling 第一手核实（六项全证）

- Ruling-base ✓：分支基=51a2b10b1（#15 tip）实测祖先关系成立。
- Ruling-prefix ✓：query-keys.ts 追加 `notificationEvents: (params) =>
  ns('notification.events', params)`，与 #15 useTestRule 失效前缀同域，契约续约成立。
- Ruling-i18n ✓：main 占位（zh 473-476 行 title:'通知事件'/description 简句）→
  分支 498 行起完整子树，title 值不变、description 按处方新文案；1147 行另一
  `events:{title:'事件'}` 为 main 既有计量事件页子树（1082 行），本轮零触碰。
- Ruling-3 ✓：events.tsx:81-84 规则/渠道下拉均 `pageSize: 100`（与 #14/#15 同限）。
- Ruling-P1 ✓：diff --stat 实测 zh-CN 42+/en 43+ 均为锚点内追加/子树替换，无既有
  行重排；prettier 范围裁定与改动面一致。
- Ruling-PD1 ✓：events.tsx:95-97 `useState(() => toLocalInputValue(new Date(
  Date.now() - 7*24*60*60*1000)))` 惰性初始化 + purity 注释，处方缺陷修复落实。

## 规格符合性

- 本轨范围 51a2b10b1..5f9e18274：legacy.ts 通知段 events API（65+）、hooks（52+，
  含 resend 失效）、query key、events.tsx 页面（575 行：时间/规则/渠道过滤、分页、
  resend 202 异步 refetch setTimeout 1500ms 处方明示手法）、路由占位替换、i18n 子树。
  与 Issue #16（通知事件流与 resend）范围一致，#15 部分由 #15 抽查轨覆盖。

## 结论

**SPOT-CHECK PASS** —— 门禁全绿、六项 Ruling 全部第一手证实、无新发现。
遗留（非阻断）：下拉 >100 条不全（Ruling-3 代价，与 #14/#15 同限）。
