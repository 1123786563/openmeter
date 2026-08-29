# Spec review — issue #27 (controller-executed programmatic, downgrade mode)

Reviewed against: issue #27 body + acceptance, issue #27 comment 1 (prescriptive
plan), master plan Task 14 first half. Evidence: /tmp/i27-walkthrough2.log
(acceptance walkthrough, 1 passed), request-capture debug run, SDK dist field
checks, final diff base 5a4666ec7..919f2bfc3.

## Acceptance item 1: 周期列表正确分页展示 — PASS (wire-level)

- 新 tab「应收周期」位于「权益」之后（tablist 顺序断言 + 激活）。
- 第一页 2 行渲染：期间（formatShortDateTime 起止）、credits 消耗（formatNumber
  bigint）、应付/已付（formatFen 元单位）、五态徽章（逾期/已付清 实测渲染）、
  关闭时间、行操作。
- cursor 分页：meta.page.next 存在时「加载更多」可见；点击后请求
  page[after]=CURSOR_1（raw cursor 直传，与事件页模式同构），第二页行（部分支付）
  追加且第一页行不丢失；末页 next=null 按钮消失。

## Acceptance item 2: 可为周期登记外部发票号 — PASS (wire-level)

- 行内「登记外部发票」→ dialog（描述含期间范围插值）。
- 发票号留空提交被 zod 前端拦截（PUT 计数 0，dialog 不关）。
- 幂等键预填 UUID（regex 断言）可编辑。
- 填号+可选 URL 提交：恰好 1 次 PUT
  /api/v3/customers/cust-001/receivable-periods/rp-1/external-invoice，
  wire 体 deep-equal {idempotency_key:<UUID>, invoice_number:'INV-2026-0001',
  invoice_url:'https://...'}（空可选字段省略；SDK camelCase→snake_case）。
- toast「外部发票已登记」→ dialog 关闭。

## Contract checks (12/12 vs SDK dist + prescriptive plan)

1. listReceivablePeriods {customerId, page{after,size:20}} ✓
2. ReceivablePeriodPaginatedResponse{data,meta} ✓ 3. CursorMetaPage.next:
   string|null ✓ 4. CommerceReceivablePeriod 11 字段（bigint×3 wire 为 JSON
   number，SDK 注释实证）✓ 5. 五状态枚举值 ✓ 6. StatusBadge tones 五态 ✓
7. updateExternalInvoice {customerId,periodId,body} ✓
8. CommerceExternalInvoiceUpdate 必填 idempotencyKey/invoiceNumber ✓
9. 可选 invoiceUrl/issuer/issuedAt(RFC3339 via Date) ✓
10. 失效用 queryKeys.receivablePeriods(customerId) 前缀吞全部 cursor 页 ✓
11. enabled=Boolean(customerId) ✓ 12. datetime-local→Date 转换 ✓

## Deviations (accepted)

- D-1: plan 的 `if (typeof crypto !== 'undefined' && 'randomUUID' in crypto)`
  在本仓 TS lib 下 `in` 收窄使 else 分支 crypto:never（TS2339/TS18046 编译失败，
  T1 门禁抓到）→ 改 `typeof globalThis.crypto?.randomUUID === 'function'` 检查，
  运行时语义不变（安全上下文 randomUUID，否则 getRandomValues 手工 v4）。
- D-2: 五个状态文案置于顶层 `receivablePeriod.status.*`（StatusBadge 组件契约
  `${domain}.status.${value}` 顶层解析，plan 第 4 步注释为权威）而非 plan 第 8
  步 i18n 块的 customers.receivablePeriods.status 嵌套位（该位不会被组件读取，
  且死键违反本仓引用检查）。
- D-3: plan tab 组件中 PAGE_SIZE 常量未使用（size 已固化在 hook 契约）→ 删除
  （T2 lint 门禁抓到）。
- 发现记录：commerce 域 wire 基路径为 /api/v3/（无 openmeter 段）——请求捕获
  实证；issue/plan 未写全路径，SDK 为契约源，无产品偏差。

## Walkthrough test-side iterations (product unchanged)

run1: mock 路径多写了 openmeter 段（请求捕获发现真实路径）→ 修 mock；run2 PASS。

## Verdict: PASS — 0 fix rounds required.
