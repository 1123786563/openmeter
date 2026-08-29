# Task 1 简报 — 数据层 + i18n（issue #16 通知事件流与 resend）

## 任务定位（一句）

OpenMeter admin-config 系列issue #16 的第一半：为通知事件流页面产出全部
数据层符号与 i18n 键；页面与路由（处方步骤 4/5）是 Task 2，不在本任务范围。

## 工作目录

- Git worktree：`/Users/wuyongjun/trea/openmeter-issue-16`（branch
  `codex/admin-config-16`，base 51a2b10b1）。所有改动与提交都在这个 worktree。
- 依赖安装：web/ 下 `pnpm install` 可能已在后台进行；若 `node_modules` 未就绪，
  先在 `web/` 运行 `pnpm install`（幂等，pnpm store 已有缓存）。

## 需求（权威处方）

先读处方文件：`/Users/wuyongjun/trea/openmeter/.superpowers/sdd/issue-16-notification-events/prescription.md`
（issue #16 comment 1 全文，含逐文件完整代码）。

**本任务只做处方步骤 1、2、3、6（步骤 4/5 是 Task 2，严禁实施）：**

1. **步骤 1** — `web/src/api/legacy.ts`：在通知段末（`testNotificationRule`
   函数之后）追加 events 段：`NotificationEventListParams` /
   `NotificationEventPaginatedResponse` / `NotificationEventResendRequest`
   接口 + `listNotificationEvents(params)` / `getNotificationEvent(eventId)` /
   `resendNotificationEvent(eventId, body?)` 函数。代码**逐字转写**处方步骤 1
   的代码块（含注释与 URLSearchParams 重复数组参数手法）。
2. **步骤 2** — `web/src/api/query-keys.ts`：在 `notificationRules` 行之后
   追加：`notificationEvents: (params: object = {}) => ns('notification.events', params),`
3. **步骤 3** — `web/src/api/hooks.ts`：在 rules 段（`useTestRule` 之后）追加
   events 段：`NotificationEventsParams` 接口 + `useNotificationEvents(params)` +
   `useResendEvent()`；import 区补
   `listNotificationEvents, resendNotificationEvent, type NotificationEventListParams`。
4. **步骤 6** — i18n 双文件 `web/src/i18n/locales/zh-CN.ts` 与 `en.ts`：
   将 `config.notification` 下的**现有 events 占位子树**（当前仅
   `title`/`description` 两键）**整块替换**为处方步骤 6 的完整 events 块。
   - Ruling-i18n：title 值与处方一致（'通知事件' / 'Notification Events'）；
     description 按处方**新文案**替换（zh：'规则触发的通知事件流（只读），支持按
     时间/规则/渠道过滤与重发。'；en 见处方）；**events 子树以外的任何既有键
     零删改**；zh 与 en 必须同构（键集与嵌套完全一致）。

## 已核实事实（不必重复验证，直接依据）

- base 上 `NotificationEvent` / `NotificationEventPayload` /
  `NotificationEventDeliveryStatus` 等类型族已在 legacy.ts L425-474（#15 产出）
  ——**直接引用，严禁重复定义**。
- hooks.ts `nsPrefix` 为 L1377 模块私有 helper；`useQueryClient` 已在 import 区。
- query-keys.ts `ns` helper 在 L8，`notificationRules` 在 L47。
- i18n events 占位子树：zh `notification:` 在 L357 起，events 子树在其内
  （channels/rules 之后）。

## 约束（全局，违反即返工）

- 处方代码逐字转写；任何偏离必须先在报告里说明理由（不得静默改动）。
- **只允许改这 5 个文件**：`web/src/api/legacy.ts`、`web/src/api/query-keys.ts`、
  `web/src/api/hooks.ts`、`web/src/i18n/locales/zh-CN.ts`、
  `web/src/i18n/locales/en.ts`。其余一律不碰（events.tsx/路由是 Task 2；
  rules/channels 域文件是 #14/#15/#12/#13 产物）。
- zh-CN.ts 与 en.ts 键集奇偶（结构化 diff 必须为空）。
- 不 merge/rebase main；分支只追加提交。
- 不得派生任何 subagent：全部工作自己做，审查由控制器负责。

## 门禁（提交前全部新鲜运行，输出留证到报告）

在 `web/` 下：

```bash
pnpm install   # 幂等，确保依赖就绪
pnpm build     # 必须 exit 0
pnpm lint      # 必须 exit 0
# routeTree 零 diff（build 后）：
git -C /Users/wuyongjun/trea/openmeter-issue-16 diff --exit-code -- web/src/routeTree.gen.ts
# locale 键集奇偶（结构行 diff 必须为空）：
diff <(grep -oE "^[[:space:]]*[\"']?[A-Za-z0-9_.-]+[\"']?[[:space:]]*:" src/i18n/locales/zh-CN.ts) \
     <(grep -oE "^[[:space:]]*[\"']?[A-Za-z0-9_.-]+[\"']?[[:space:]]*:" src/i18n/locales/en.ts) \
  && echo PARITY_OK
# 变更文件 prettier（--check 不过就 --write 后重跑 build）：
npx prettier --check src/api/legacy.ts src/api/query-keys.ts src/api/hooks.ts src/i18n/locales/zh-CN.ts src/i18n/locales/en.ts
```

## 提交

单次提交（可含自查修正的 amend）：

```
feat(admin): 通知事件 API 封装与事件流 i18n（issue #16 task 1）
```

## 报告

完整报告写入：
`/Users/wuyongjun/trea/openmeter/.superpowers/sdd/issue-16-notification-events/task-1-report.md`
（内容：实施了什么、逐条门禁命令与结果摘要、文件清单、自查发现、疑虑）。
回复控制器时只给 ≤15 行：Status（DONE / DONE_WITH_CONCERNS / BLOCKED /
NEEDS_CONTEXT）、commit 短 SHA 与标题、一行门禁摘要、疑虑（如有）、报告路径。
