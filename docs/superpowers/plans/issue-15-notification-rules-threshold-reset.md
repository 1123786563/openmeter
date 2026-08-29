# Plan — issue #15 通知规则：阈值/重置型与 test 触发

- Issue: https://github.com/1123786563/openmeter/issues/15
  ([admin-config 15/29] 通知规则：阈值/重置型与 test 触发)
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 8 后半
  （前半 #14 规则列表/发票型表单、#2 功能目录均已合入 main = ec85f6871）。
- 处方源（权威）：issue #15 comment 1 全文，已存
  `/tmp/issue-round3/issue-15-comments.md`（逐文件完整代码）。
- Branch codex/admin-config-15 @ base ec85f6871；worktree
  /Users/wuyongjun/trea/openmeter-issue-15。
- Ledger: .superpowers/sdd/issue-15-notification-rules-threshold-reset/progress.md
  （主检出目录）。
- 串行说明：#16（事件流/resend）与本轨同改 legacy.ts 通知段且直接消费本轨产出
  的 NotificationEvent 类型族——#16 留待本轨外部化后领取。

## 范围（Scope）

- `web/src/api/legacy.ts`（rules 段末追加，处方步骤 1 逐字）：事件实体类型
  `NotificationEventType` / `NotificationEventDeliveryState` /
  `EventDeliveryAttemptResponse` / `NotificationEventDeliveryAttempt` /
  `NotificationEventDeliveryStatus` / `NotificationEventPayload` /
  `NotificationEvent` + 函数 `testNotificationRule(ruleId)`（POST
  `/v1/notification/rules/{ruleId}/test`，201 → NotificationEvent）。
- `web/src/api/hooks.ts`（处方步骤 2）：`useTestRule()`——mutate(ruleId)，
  onSuccess 失效 `nsPrefix('notification.events')`（为 #16 预热的键前缀，
  当前无该键查询=无害 no-op）。
- `web/src/features/config/notification/rule-form-dialog.tsx`：**完整替换**为
  四类型判别联合表单（处方步骤 3 逐字）：threshold 型 thresholds[1..10]
  useFieldArray + features MultiSelect；reset 型 features MultiSelect；类型切换
  switchType 迁移联合形状；空 features 数组提交时省略字段（spec minItems 1）。
- `web/src/features/config/notification/rules.tsx`：**完整替换**（处方步骤 4 逐字）：
  编辑放开全类型 + 每行「发送测试」按钮（禁用规则 disabled；ConfirmDialog 确认；
  toast 展示生成事件 id）。
- i18n：`config.notification.rules` 子树追加（处方步骤 5 全键集，**合并入既有
  #14 键，不删不改既有键**），zh-CN.ts 与 en.ts 同构。

## 非目标（Non-goals）

- 事件列表/详情/resend 页（#16 范围；本轨只产出其复用的类型与 test 端点封装）。
- 渠道管理（#12/#13/#30 已交付，零改动）。
- 不改 useCreateRule/useUpdateRule/useToggleRule（#14 产物原样复用）。
- 无后端/TypeSpec/SDK 改动。

## 已核实的契约事实（main ec85f6871 实测，2026-08-29）

- legacy.ts 已有：`NotificationRule`（thresholds: NotificationRuleThreshold[] /
  features?: FeatureMeta[]，L252-258）、`NotificationRuleCreateRequest` 判别联合
  （L323-353）、`NotificationChannelMeta`（L227）、`ruleToUpdateBody`。
- hooks.ts 已有：`useNotificationChannels(params)` / `useNotificationRules(params)` /
  `useCreateRule` / `useUpdateRule` / `useToggleRule`（L824-923）；
  `nsPrefix` 为 hooks.ts:1363 模块私有 helper。
- spec：thresholds 每项 `{value: number, type: 'PERCENT'|'NUMBER'|'balance_value'|
  'usage_percentage'|'usage_value'}`（PERCENT/NUMBER deprecated）；features 元素
  string（id 或 key），minItems 1、省略=全部功能、空数组非法。
- `MultiSelect` 组件在 web/src/components/multi-select.tsx（props: options/value/
  onChange/placeholder/searchPlaceholder/emptyText，#14 已用同参）。
- zod v4 `z.discriminatedUnion` + RHF 字段名分支收窄（处方注记）成立。

## 接口适配 Ruling（实施与审查均以此为准）

- **Ruling-1（useFeatures 签名）**：处方备选块设想零参 `useFeatures()`，但 #2
  实际落地 `useFeatures(params: FeatureListParams)`（hooks.ts:695，page/pageSize
  必填）。适配：表单内 `useFeatures({ page: 1, pageSize: 100 })`，选项自
  `data?.data ?? []`（features/index.tsx:114 同模式）。**不新增零参变体、不改
  query-keys**（features key 已存在）。代价：>100 条功能时下拉不全——与功能页
  首页同限，可接受。
- **Ruling-2（invalidation 前缀前瞻）**：useTestRule 失效 `nsPrefix('notification.
  events')` 在 #16 落地前无监听查询（无害 no-op），保留处方原样——它正是 #16
  事件的键前缀契约。
- 处方其余代码逐字有效；`Control<ThresholdFormValues>` / `Control<FeaturesAwareFormValues>`
  的 `as unknown as` 收窄为处方明示手法，非类型逃逸缺陷。

## 任务拆分

- Task 1：数据层 + i18n——legacy.ts 事件类型族 + testNotificationRule；hooks.ts
  `useTestRule`；zh-CN/en `config.notification.rules` 追加全键集（合并保留既有键）。
  验证：`cd web && pnpm build && pnpm lint`。
- Task 2：UI 层——rule-form-dialog.tsx 完整替换（四类型表单）；rules.tsx 完整
  替换（test 按钮 + 编辑全类型）。验证：`cd web && pnpm build && pnpm lint &&
  pnpm test:e2e`。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- `pnpm build` 0 error；routeTree 零 diff（无路由变更）。
- `pnpm lint` 0 error 0 warning（新增行）。
- e2e：既有 2 条冒烟（sign-in 应 PASS；customers smoke 环境性失败以与 pristine
  base 同签名为非回归判据——历轮共识）。
- locale 奇偶：zh 键数 == en 键数（config.notification.rules 子树求值同构，
  既有键零删除——用键集双向差集验证）。
- prettier：修改文件 formatted(base) vs formatted(mine) 差异恰为插入块。

验收（Issue AC）：
1. 「可创建阈值型/重置型规则，features 多选来自功能目录」——四类型表单 +
   thresholds 1-10 数组编辑 + features MultiSelect（选项=功能目录）。
2. 「test 按钮触发并展示结果」——test 按钮 + ConfirmDialog + toast(event.id)。
类型面 = 提交体与 NotificationRuleCreateRequest 判别联合逐分支吻合（tsc 证明）。

## 全局约束（Master plan Global Constraints 摘要 + 本轨补充）

- 文案全部 i18n，zh-CN 与 en 同步；术语按 CONTEXT.md（通知规则/通知渠道/功能）。
- test 为真实投递副作用——必须经 ConfirmDialog 明示后果；错误 handleServerError
  原文透出。
- v1 端点经 legacy.ts `apiFetch`（自动注入 Authorization/X-Namespace）。
- 禁止臆造字段；空 features 数组必须省略而非提交（spec minItems 1）。
- 禁用规则 test 按钮 disabled（处方明示）。
