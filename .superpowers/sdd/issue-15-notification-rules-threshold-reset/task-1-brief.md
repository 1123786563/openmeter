# Task 1 brief — issue #15 通知规则阈值/重置型与 test（数据层 + i18n）

Worktree: /Users/wuyongjun/trea/openmeter-issue-15（branch codex/admin-config-15，
base ec85f6871；实施计划已随 plan commit 1aae6dacc 入分支）。

## 需求源（先读这两个文件）

1. 实施计划：`/Users/wuyongjun/trea/openmeter-issue-15/docs/superpowers/plans/issue-15-notification-rules-threshold-reset.md`
   （范围/非目标/Ruling/验收命令）。
2. 处方（权威，逐字代码）：`/tmp/issue-round3/issue-15-comments.md`。
   本任务只做其中 **步骤 1（legacy.ts 事件类型族 + testNotificationRule）**、
   **步骤 2（useTestRule）**、**步骤 5（i18n 追加）**。步骤 3/4（表单与列表页
   完整替换）属 Task 2，**勿做**。

## 交付（按序）

1. `web/src/api/legacy.ts`：在 rules 段末（现有 `ruleToUpdateBody` 等之后）追加
   处方步骤 1 的全部事件实体类型（`NotificationEventType` /
   `NotificationEventDeliveryState` / `EventDeliveryAttemptResponse` /
   `NotificationEventDeliveryAttempt` / `NotificationEventDeliveryStatus` /
   `NotificationEventPayload` / `NotificationEvent`）与函数
   `testNotificationRule(ruleId)`。逐字采用（`NotificationChannelMeta` /
   `NotificationRule` 已在文件内，勿重复定义）。
2. `web/src/api/hooks.ts`：追加处方步骤 2 第一块 `useTestRule()`（逐字）。
   **不要**添加处方中「若 #2 未合入」的备选 `useFeatures` 块，也**不要**改
   query-keys.ts（#2 已合入，features key 已存在——计划 Ruling-1）。
3. `web/src/i18n/locales/zh-CN.ts` 与 `web/src/i18n/locales/en.ts`：向
   `config.notification.rules` 子树**合并追加**处方步骤 5 键集（zh 全文 + en 全文
   均在处方内，逐字采用）：
   - 新增顶层键 `test`；新增子树 `thresholdTypes`、`testConfirm`
   - `fields` 内合并追加 `thresholds` / `features`（保留既有
     type/name/channels/status/disabled）
   - `form` 内合并追加 `thresholdsHint` / `addThreshold` / `removeThreshold` /
     `featuresPlaceholder` / `featuresHint` / `noFeatures`；
     `form.validation` 内合并追加 `threshold`（保留 required/channels）
   - `toast` 内合并追加 `testSent`（保留既有四键）
   - 既有键一个不删不改；zh 与 en 键集完全同构。

## 验证（全部在 worktree 的 web/ 目录）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-15/web && pnpm build && pnpm lint
```

- `pnpm build` 0 error；`pnpm lint` 0 error 0 warning。
- routeTree.gen.ts 零 diff。

## 提交

- 只 add 上述三类文件；commit message：
  `feat(admin): 通知规则 test 端点封装与阈值/重置 i18n（issue #15 task 1）`

## 硬约束

- 处方代码逐字采用（含注释与命名）；不新增依赖。
- 只在 /Users/wuyongjun/trea/openmeter-issue-15 内工作；绝不动主检出
  /Users/wuyongjun/trea/openmeter 或其他 worktree。
- 禁止派生任何 subagent。
