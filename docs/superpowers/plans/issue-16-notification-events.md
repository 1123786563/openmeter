# Plan — issue #16 通知事件流与 resend

- Issue: https://github.com/1123786563/openmeter/issues/16
  ([admin-config 16/29] 通知事件流与 resend)
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 8
  （#14 规则列表/hooks、#15 事件类型族+test 触发均已本地完成于 codex/admin-config-15）。
- 处方源（权威）：issue #16 comment 1 全文，已存
  `.superpowers/sdd/issue-16-notification-events/prescription.md`（主检出目录，
  逐文件完整代码）。
- Branch codex/admin-config-16 @ base **51a2b10b1（codex/admin-config-15 tip，非 main）**；
  worktree /Users/wuyongjun/trea/openmeter-issue-16。
- Ledger: .superpowers/sdd/issue-16-notification-events/progress.md（主检出目录）。
- 串行链说明：#16 硬依赖 #15 产出的 `NotificationEvent` 类型族（main 上不存在，
  仅在 codex/admin-config-15）→ 本轨基于 #15 tip 实施；外部化时按 #11→#15→#16
  升序合并，本轨分支已含 #15 提交，#15 先合则本轨为快进式追加。

## 范围（Scope）

- `web/src/api/legacy.ts`：通知段末（`testNotificationRule` 之后）追加 events 段
  （处方步骤 1 逐字）：`NotificationEventListParams` /
  `NotificationEventPaginatedResponse` / `NotificationEventResendRequest` +
  `listNotificationEvents(params)` / `getNotificationEvent(eventId)` /
  `resendNotificationEvent(eventId, body?)`（POST resend 202 无内容 → `Promise<void>`）。
- `web/src/api/query-keys.ts`：`notificationRules` 行后追加
  `notificationEvents: (params: object = {}) => ns('notification.events', params)`。
- `web/src/api/hooks.ts`：rules 段（含 useTestRule）后追加 events 段（处方步骤 3
  逐字）：`NotificationEventsParams`（from?/to?/rule?/channel? 单选 string，page/
  pageSize 必填）+ `useNotificationEvents(params)`（单选升数组）+ `useResendEvent()`
  （空选择=省略 channels=原渠道；onSuccess 失效 `nsPrefix('notification.events')`）；
  import 区补 `listNotificationEvents, resendNotificationEvent,
  type NotificationEventListParams`。
- `web/src/features/config/notification/events.tsx`：新建（处方步骤 4 逐字）——
  过滤草稿（from/to datetime-local、rule/channel 单选 Select）+「应用过滤」重置页码、
  分页表格（PAGE_SIZE 20）、行展开投递明细（每渠道 state/reason/nextAttempt/attempts
  含 HTTP 状态码/耗时/响应体 + 载荷 JSON）、行内单条刷新（getNotificationEvent +
  eventOverrides 覆盖显示）、resend ConfirmDialog（MultiSelect 可选渠道子集，留空=
  原渠道；成功 toast + 1.5s 后 refetch）。
- `web/src/routes/_authenticated/config/notification/events/index.tsx`：**占位路由
  替换**为真实路由（处方步骤 5 逐字，component: NotificationEventsPage）。
- i18n：`config.notification.events` 子树**以处方全块替换现有占位块**（处方步骤 6，
  zh-CN.ts 与 en.ts 同构）；现有占位 `title` 值与处方一致保留原值，
  `description` 按处方文案更新；其余键全部为新增。

## 非目标（Non-goals）

- 规则/渠道页面与其 hooks（#14/#15/#12/#13/#30 产物）零改动原样复用。
- 不重复定义事件实体类型族（#15 已产出，直接 import）。
- 无后端/TypeSpec/SDK/Go 改动（纯 web 前端轨，无 go 门禁）。
- 不新增 e2e 用例（浏览器真实走查归 #29 全链路验收）。
- events 页规则/渠道下拉 `pageSize: 100` 上限（与 #14/#15 页面同限，见 Ruling-3）。

## 已核实的契约事实（base 51a2b10b1 实测，2026-08-29）

- legacy.ts L425-474：`NotificationEventDeliveryState` =
  `'SUCCESS'|'FAILED'|'SENDING'|'PENDING'|'RESENDING'` —— 与处方
  `DELIVERY_STATE_CLASS` 五键一一对应（无缺漏，Record 类型完整）。
- `NotificationEventType` = `'entitlements.balance.threshold'|'entitlements.reset'|
  'invoice.created'|'invoice.updated'` —— 与 #15 i18n
  `config.notification.rules.types.*` 现有四键一一对应：页面
  `t('config.notification.rules.types.${event.type}')` 全值可解析。
- hooks.ts：`useNotificationChannels(params)` L825 / `useNotificationRules(params)`
  L885（均 `{page, pageSize}`）；`nsPrefix` L1377 模块私有；`useQueryClient` 已在
  import 区（useTestRule 已用）。query-keys.ts：`ns` L8、`notificationRules` L47。
- `apiFetch<void>` 先例：legacy.ts L210、L539（POST 无内容响应）。
- `formatDateTime`：web/src/lib/format.ts L35（`Date | string | undefined | null`）。
- events 路由**已存在为占位**：`PlaceholderPage titleKey='config.notification.events.
  title'`（早期脚手架）→ 处方步骤 5 落地为「替换」而非新建。
- i18n events 子树**已存在为占位**（zh：title '通知事件'/description '查看通知事件流
  并重发。'；en 同构）→ 处方块 title 同值、description 新文案（zh '规则触发的
  通知事件流（只读），支持按时间/规则/渠道过滤与重发。'）。
- `MultiSelect`/`ConfirmDialog`（含 isLoading/handleConfirm）/
  `Header`/`Main`/`PageHeader` 均已存在且 #14/#15 同参使用。

## 接口适配 Ruling（实施与审查均以此为准）

- **Ruling-base（轨基）**：worktree 基于 codex/admin-config-15 tip 51a2b10b1 而非
  main——事件类型族硬依赖；代价：#15 外部化若被否，本轨需 rebase。
- **Ruling-i18n（占位键替换）**：events 子树为**替换**而非纯追加：description 值
  按处方更新，title 值不变；zh/en 同构替换；**子树外既有键零删改**。
- **Ruling-prefix（键前缀契约）**：`ns('notification.events', params)` 正是 #15
  useTestRule 预热的失效前缀（#15 Ruling-2 的续约）；useResendEvent 失效同前缀。
- **Ruling-3（下拉 100 上限）**：规则/渠道 Select 选项取 `pageSize: 100` 首页
  （处方原文如此，与 #14/#15 页面同模式）；代价：>100 条时下拉不全，可接受。
- 处方其余代码逐字有效；发现处方缺陷（PD）须记 ledger Ruling 后方可偏离。

## 任务拆分

- Task 1：数据层 + i18n——legacy.ts events 段 + query-key + hooks + i18n events
  子树替换（处方步骤 1/2/3/6）。门禁：`pnpm build && pnpm lint`（web/）、
  routeTree 零 diff、locale zh==en 键集奇偶。
- Task 2：页面与路由——events.tsx 新建 + 路由占位替换（处方步骤 4/5）。门禁：
  build/lint、e2e 基线签名比对（sign-in ✓ / customers 环境性基线）、变更文件
  prettier + 定向 eslint 0。

## 验证与提交（两任务同规）

- 全部命令在 worktree `web/` 下新鲜运行，日志落盘
  `.superpowers/sdd/issue-16-notification-events/`（主检出）。
- e2e：`pnpm test:e2e` 既有 2 条冒烟；customers 冒烟为已知环境性失败，判据=
  与 pristine 基线**同签名**（先例：#11/#15 两轮比对）。
- 提交规范（循 #15）：`docs(admin): issue #16 通知事件流与 resend 实施计划`；
  `feat(admin): <摘要>（issue #16 task N）`。

## 全局约束

- 处方（comment 1）为权威规格；步骤代码逐字转写；任何偏离须先记 Ruling。
- i18n 既有键零删改（events 子树内 description/title 按处方除外）；zh-CN/en
  同构。
- 不触碰 plans 域（#11 并行轨文件面）、不触碰 rules/channels 页面与组件。
- 分支只基于 #15 tip 追加提交；不 merge/rebase main（外部化统一处理）。
- 不得在主检出（main 工作树）直接改 web 源码；一切改动在 openmeter-issue-16。
