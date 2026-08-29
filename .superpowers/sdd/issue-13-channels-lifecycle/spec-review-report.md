# Spec review — issue #13 (controller-executed programmatic, 2026-08-29 01:5x+08:00)

Method: prescription (issue comment 1) vs branch diff 5a4666ec7..8086b062b,
item-by-item; UI/wire behavior verified by temporary stateful Playwright
walkthrough (deleted after run).

## Acceptance criteria (issue body)

- 可编辑渠道、切换禁用状态、删除渠道 → walkthrough (stateful wire mocks):
  (1) 编辑：dialog 标题「编辑通知渠道」，signingSecret 输入框回填原值
  (toHaveValue)，改名保存 → PUT /api/v1/notification/channels/chan-1 body 精确
  等于全量 {type:'WEBHOOK', name:'…-v2', url, disabled:false, customHeaders,
  signingSecret:SECRET} → toast 已更新；(2) 禁用：确认弹窗（非 destructive 标题
  「禁用通知渠道」）→ PUT #2 disabled:true 且 signingSecret/customHeaders 保真 →
  toast 已禁用 → 行徽章「已禁用」；(3) 删除：确认弹窗（destructive）→
  DELETE 调用 1 次 → toast 已删除 → 行消失（空态）。PASS。

## Prescription contract items

- legacy.ts updateNotificationChannel/deleteNotificationChannel（PUT 全量替换
  doc 注释 + 软删除注释；encodeURIComponent path）✓ 逐字。
- hooks.ts useUpdateChannel/useDeleteChannel（nsPrefix('notification.channels')
  失效）✓ 逐字；imports 补齐。
- ChannelFormDialog 编辑模式：channel? prop、useEffect 回填（signingSecret ??
  ''、customHeaders entries→rows）、isCreate 分流、editTitle/editDescription、
  isSubmitting 合并 ✓ — 按 RULING 以现状 FieldError/V 模式承载（处方中的
  FormMessage children 三元式是被 #12 修复轮废弃的缺陷模式，未重新引入）。
- channels.tsx：toChannelBody 行级全量重建、操作列（编辑/禁用|启用/删除）、
  两个 ConfirmDialog、表头「操作」列、空态 colSpan 5 ✓。
- i18n：16 个新 key（actions/enable/disable/delete/toggleConfirm×4/
  deleteConfirm×2/form.edit×2/toast×4）zh+en 全部落位（程序化 keycheck；
  keycheck 报告的 form.validation 命中为 V 前缀常量的正则误报，非叶子键）。

## Deviations from prescription (with rationale)

- R-1 (pre-implementation RULING, in plan doc): form dialog keeps current
  i18n-key zod messages + FieldError; prescription's FormMessage children
  pattern not reintroduced (defect class fixed in 5ec73ca05). ACCEPTED.
- R-2 (wording): zh disableDescription 采纳处方语义但明确「本页已显式包含已禁用
  渠道」（与 useNotificationChannels includeDisabled:true 的实际行为一致）；en
  同步。key 集与处方一致。ACCEPTED.

Verdict: PASS — no Critical/Important findings; 0 fix rounds needed.
