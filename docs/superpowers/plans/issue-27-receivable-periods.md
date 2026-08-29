# Plan — issue #27 客户详情：应收周期 tab 与外部发票登记

- Issue: https://github.com/1123786563/openmeter/issues/27
  ([admin-config 27/29] 客户详情：应收周期 tab 与外部发票登记)
- Prescriptive source of truth: issue #27 comment 1（649 行处方级计划，本轮已
  全文核对；/tmp/issue-27-plan.md 为转录副本）。
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 14 第一半。
- Branch codex/admin-config-27 @ base 5a4666ec7；worktree
  /Users/wuyongjun/trea/openmeter-issue-27。
- Ledger: .superpowers/sdd/issue-27-receivable-periods/progress.md（worktree-local）。

## 范围（Scope）

- `web/src/api/query-keys.ts`：追加 `receivablePeriods(customerId, params)`。
- `web/src/api/hooks.ts`：追加 `useReceivablePeriods(customerId, after?)` /
  `useUpdateExternalInvoice(customerId)`（v3 commerce）。
- 新建 `web/src/lib/idempotency.ts`（UUID 幂等键助手，#28 复用）。
- `web/src/components/status-badge.tsx`：`tones` 追加 `receivablePeriod` 五态域
  （refund 域之后）。
- 新建 `web/src/features/customers/receivable-periods-tab.tsx`（cursor 分页列表
  + 加载更多，镜像事件页模式）与 `external-invoice-dialog.tsx`（登记外部发票）。
- `web/src/features/customers/customer-detail.tsx`：TabsList 末尾增
  `receivable-periods` 触发器 + 新 TabsContent。
- i18n：`customers.detail.tabs.receivablePeriods` + `customers.receivablePeriods`
  完整子树（zh-CN/en 同构）+ StatusBadge 文案命名空间 `receivablePeriod.status.*`。

## 非目标（Non-goals）

- 线下支付登记 dialog（页首入口 + CreateOfflinePayment）→ #28。
- 周期分页以外的过滤/排序；后端改动。
- 外部发票的删除/清空（spec 仅 PUT upsert）。

## 已核实的契约事实（SDK dist + 主仓源码，2026-08-29）

- `api.commerce.listReceivablePeriods({ customerId, page: { after, size: 20 } })` →
  `ReceivablePeriodPaginatedResponse = { data: CommerceReceivablePeriod[],
  meta: CursorMeta }`；`CursorMetaPage.next: string | null`（后端 cursorpagination.go
  已在 issue 中核实为原始 cursor 串，可直接作 page[after]；与事件页
  web/src/features/events/index.tsx loadMore 模式同构——本仓源码已核对 L54-83）。
- `CommerceReceivablePeriod` = { id, customerId, periodStart/periodEnd: Date,
  creditsConsumed: bigint, amountDueFen: bigint, amountPaidFen: bigint, currency,
  status: 'open'|'closed'|'partially_paid'|'paid'|'overdue', createdAt, closedAt? }。
- `api.commerce.updateExternalInvoice({ customerId, periodId, body })`；
  `CommerceExternalInvoiceUpdate` = { idempotencyKey(必填), invoiceNumber(必填),
  invoiceUrl?, issuer?, issuedAt?: Date } —— 与处方逐字段一致。
- `formatFen/formatNumber/formatShortDateTime`（@/lib/format）已存在；
  StatusBadge tones 表 + refund 域位置已核对；customer-detail.tsx tabs 结构
  （entitlements 在末尾）已核对。
- 无偏差预判：处方与 SDK/仓内事实完全一致（含 datetime-local → Date 转换、
  幂等键 crypto.randomUUID 安全上下文回退）。

## 任务拆分（每任务独立 commit + build/lint 门禁）

- T1 API 层：query-keys + hooks + lib/idempotency.ts + status-badge tones。
- T2 页面层：receivable-periods-tab.tsx + external-invoice-dialog.tsx +
  customer-detail.tsx 接入。
- T3 i18n：zh-CN/en 完整子树。

## 测试与验收命令

- 每 commit：`cd web && pnpm build && pnpm lint`。
- 终态：`pnpm build && pnpm lint && pnpm test:e2e`（e2e 与并行轨串行执行端口）。
- locale parity：zh/en 键数一致 + 新增键全部被静态引用。
- 浏览器走查（全端点 mock 的临时 spec）：新 tab 位于既有 tab 之后；列表
  （期间/credits 消耗/应付/已付 formatFen/五态徽章/关闭时间）；cursor 分页
  「加载更多」追加不闪烁；登记 dialog 发票号必填前端拦截 → 提交 PUT wire 体
  deep-equal（idempotencyKey=UUID、invoiceNumber、可选字段缺省正确）→ toast；
  空态。

## 全局约束

- 只动 web/ 下 8 个文件（见范围）；三 Go 模块不可误改。
- 与并行轨 #25 的共享面仅 query-keys.ts/hooks.ts/两 locale 文件的追加段——
  子树互不相交（receivablePeriods vs billingProfiles+apps）。
- 与未合入 #8/#13/#17/#19/#21/#23 无文件子树交集（customers 域未被任何轨触碰）。
- 未经用户批准不得 push/merge/close/改 GitHub 状态。
- subagent 被禁 → 沿用既定降级：控制者执行实施 + 程序化三审（spec/quality/
  final），独立复跑验证、日志落盘。
