# Spec review — issue #14 (controller-executed, downgrade mode)

Reviewed against: issue #14 body + acceptance, issue #14 prescription comment
(5 per-file sections), base 49d1a760b → HEAD 68826e40f. Standing
isolation-downgrade (subagents denied; probes on record in round claim
`issue-2026-08-29-9-14-claim.md`); reviews run as separate controller passes
with fresh commands, logs on disk under /tmp/issue-round/.

## §1 legacy.ts rules 段 (2a44c97a4) — 逐字在场

- 判别联合四型（invoice.created / invoice.updated /
  entitlements.balance.threshold / entitlements.reset）+ 公共字段；
  NotificationRuleThreshold{value, type: PERCENT|NUMBER …}。
- listNotificationRules：URLSearchParams，feature[]/channel[] append、
  includeDeleted/includeDisabled 仅 true 时序列化。
- ruleToUpdateBody：PUT 全量回传 switch（thresholds/features/metadata 保留）。
- openapi.yaml 字段核验（判别器 type、name 1..256、channels ULID 数组、
  thresholds minItems1/maxItems10、PUT=Create 全替换）全部与实现一致。

## §2 query-keys + hooks (2a44c97a4)

- `notificationRules: (params) => ns('notification.rules', params)`；
  useNotificationRules（includeDisabled: true）/ useCreateRule / useUpdateRule /
  useToggleRule（经 ruleToUpdateBody）在场。

## §4 MultiSelect (9dc876ac2) — 逐字（唯一偏差：import 排序，prettier 强制）

## §5-§8 表单/列表/路由/i18n (68826e40f) — 逐字

- rule-form-dialog：发票型 zod 判别联合（channels min(1)）、MultiSelect 渠道、
  创建/编辑复用、typeHint 明示 threshold/reset 为 follow-up。
- rules 页：类型徽章四色、渠道名解析（Map）、PAGE_SIZE=20 分页、启停
  ConfirmDialog（destructive 随启停方向）、isEditableType 类型守卫收窄。
- 路由替换 PlaceholderPage；routeTree.gen.ts 零 diff。
- i18n：完整 rules 块替换 Task-1 占位子树（两语同步）。

## 行为级证据

- ruleToUpdateBody 3/3 PASS（node strip-types + stub api）：发票型全量回传
  含 metadata、threshold 型保 thresholds+features(id)、reset 型无 features 省键。

## 处方偏差（均为 prettier 强制，记录在案）

两新文件 import 排序（--write）；zh 2 处 / en 5 处插入块内长行换行
（选择性格式化，base 既有脏行 zh 4 / en 2 hunks 保持不动）。

## RULING: PASS
