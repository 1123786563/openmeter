# Plan — issue #28 客户详情：线下支付登记

- Issue: https://github.com/1123786563/openmeter/issues/28
  ([admin-config 28/29] 客户详情：线下支付登记)
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 14 后半
  （前半 #27 已合入：应收周期 tab + 外部发票登记 + lib/idempotency.ts）。
- Branch codex/admin-config-28 @ base f60cb90b0；worktree
  /Users/wuyongjun/trea/openmeter-issue-28。
- Ledger: .superpowers/sdd/issue-28-offline-payment/progress.md（worktree-local）。

## 范围（Scope）

- `web/src/api/hooks.ts`（Receivable periods 段内追加）：
  `useCreateOfflinePayment(customerId)`（`api.commerce.createOfflinePayment`，
  onSuccess 失效 `queryKeys.receivablePeriods(customerId)`——周期已付金额刷新
  是 Issue 验收项）。
- 新建 `web/src/features/customers/offline-payment-dialog.tsx`：
  - 字段（Issue 处方逐项）：幂等键（打开时 `generateIdempotencyKey()` 预填、
    可改，模式照抄 external-invoice-dialog）/ 金额（元，zod 正数；提交
    `BigInt(Math.round(Number(yuan) * 100))` → amountFen，先例
    recharge-product-form-dialog.tsx:169）/ 币种（必填；选中周期时自动预填
    该周期 currency）/ 外部参照 externalReference（必填，银行流水/汇款附言）/
    received_at（datetime-local 必填 → `new Date(v)`）/ 所属周期下拉
    （可选，选项=当前已加载周期 `期间 ~ 期间 · 应付 formatFen`；未选可提交）/
    note（可选）。
  - 提交体 deep-equal `{ idempotencyKey, amountFen, currency,
    receivablePeriodId?, externalReference, receivedAt, note? }`
    （`CommerceOfflinePaymentCreate`）；成功 toast + 关闭。
- `web/src/features/customers/receivable-periods-tab.tsx`：tab 页首新增一行
  （左侧说明文案「无列表端点，已缴见周期已付金额」——Issue 明示；右侧
  「登记线下支付」按钮），向 dialog 传入 `periods={allPeriods}` 与 customerId。
- i18n：`customers.receivablePeriods.offlinePayment.*` 完整子树（标题/描述/
  字段/提示/toast/周期占位），zh-CN 与 en 同构。

## 非目标（Non-goals）

- 线下支付列表/删除（v3 无列表端点——Issue 明示页面只需说明文案）。
- 外部发票登记、周期 tab 本体（#27 已交付，列表列与分页零改动）。
- 金额跨币种换算（元→fen 仅小数点移位，币种照填照传）。
- 后端/TypeSpec 改动。

## 已核实的契约事实（SDK dist models/operations/commerce.d.ts，2026-08-29）

- `api.commerce.createOfflinePayment({ customerId, body })`；
  `CommerceOfflinePaymentCreate = { idempotencyKey: string, amountFen: bigint,
  currency: string, receivablePeriodId?: string, externalReference: string,
  receivedAt: Date, note?: string }`——externalReference/receivedAt/idempotencyKey/
  amountFen/currency 必填，与 Issue 处方一致。
- `generateIdempotencyKey()`（lib/idempotency.ts，#27 入库）安全上下文回退
  已处理；dialog reset-on-open 模式照抄 external-invoice-dialog.tsx L74-84。
- `allPeriods`（tab 内已聚合的跨页周期数组）即周期下拉数据源；
  `formatFen`/`formatShortDateTime` 已存在。
- 元→fen 先例：`BigInt(Math.round(Number(values.amountYuan) * 100))`。

## 任务拆分（每任务独立 commit + `cd web && pnpm build && pnpm lint`）

- T1 API 层：useCreateOfflinePayment。
- T2 dialog + tab 页首挂载 + i18n（zh/en 同构）。

## 测试与验收命令

- 每 commit：`cd web && pnpm build && pnpm lint`（routeTree 零 diff）。
- 终态追加：`pnpm test:e2e`（与并行轨串行）；locale parity 真实求值；
  prettier 新文件 clean、修改文件差异恰为插入块。
- 浏览器走查（全端点 mock）：页首说明文案 + 登记按钮；打开 dialog 幂等键
  预填 UUID 且可改；空金额/负数/外部参照空/received_at 空 → zod 拦截；
  选中周期后币种预填该周期 currency；提交 wire 体 deep-equal（含
  BigInt(amountFen)、未选周期时无 receivablePeriodId 键、note 空时无键）；
  成功 toast + 关闭 + 周期列表已付金额刷新（query 失效）；无周期时下拉
  占位「不关联周期」仍可提交。

## 全局约束

- 只动 web/ 下上述文件；三 Go 模块不可误改。
- 与并行轨 #22/#24/#26 共享面仅 hooks.ts 追加段与 locale 文件互不相交子树。
- 文案全部 i18n（zh-CN + en 同步），术语沿 CONTEXT.md 词表（应收周期/线下支付）。
- 所有 API 请求经 v3 SDK 单例；金额换算仅元↔fen，不得引入浮点累计。
- 未经用户批准不得 push/merge/close/改 GitHub 状态。
- implementer 不得自行派生 subagent；不得跳过任务审查/修复复审/终审/台账。
