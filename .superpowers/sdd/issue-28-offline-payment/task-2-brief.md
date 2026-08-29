# Task 2 brief — issue #28 T2 线下支付 dialog + tab 页首挂载 + i18n

Worktree: /Users/wuyongjun/trea/openmeter-issue-28（分支 codex/admin-config-28）
T1 已提供（勿改）：`useCreateOfflinePayment(customerId)`（body 类型 SDK
`CommerceOfflinePaymentCreate` = `{idempotencyKey, amountFen: bigint, currency,
receivablePeriodId?, externalReference, receivedAt: Date, note?}`；成功后自动失效
receivablePeriods 查询）。

## 必读参考（worktree 内先读再写）

- `docs/superpowers/plans/issue-28-offline-payment.md`
- `web/src/features/customers/external-invoice-dialog.tsx`（**首要模板**：reset-on-open
  幂等键生成 L74-84、datetime-local→Date、Form/zod/toast/handleServerError）
- `web/src/features/customers/receivable-periods-tab.tsx`（挂载点 + allPeriods 聚合）
- `web/src/features/commerce/recharge-product-form-dialog.tsx:169`（元→fen：
  `BigInt(Math.round(Number(yuan) * 100))`）
- `web/src/lib/idempotency.ts`（generateIdempotencyKey）、`web/src/lib/format.ts`
  （formatFen/formatShortDateTime）
- shadcn Select 组件：`web/src/components/ui/select.tsx`（存在则用；不存在则用
  原生 select 或既有项目内下拉模式，报告里说明选择）

## 交付物

1. 新建 `web/src/features/customers/offline-payment-dialog.tsx`：
   - Props `{ open, onOpenChange, customerId, periods: CommerceReceivablePeriod[] }`。
   - 打开 reset：idempotencyKey=`generateIdempotencyKey()`（可改，mono 展示）；
     其余空；currency 若 periods[0] 存在可留空待选周期预填（不强制）。
   - 字段与校验（zod）：
     - idempotencyKey min(1)
     - amountYuan：字符串，正则 `^\d+(\.\d{1,2})?$`（正数、最多两位小数），
       错误键 offlinePayment.validation.amount
     - currency：min(1)（文本；选中周期时 setValue(period.currency)——用
       watch/联动或 onChange 手动回填，报告说明实现）
     - externalReference：min(1)（银行流水/汇款附言）
     - receivedAt：datetime-local min(1)
     - receivablePeriodId：可选；下拉选项 = periods.map → label
       `${formatShortDateTime(periodStart)} ~ ${formatShortDateTime(periodEnd)} ·
       ${formatFen(amountDueFen, currency)}`，value=id；含「不关联周期」占位项（值空）
     - note：可选
   - 提交体（**逐字段对齐 SDK**）：
     `{ idempotencyKey: v.trim(), amountFen: BigInt(Math.round(Number(v.amountYuan)*100)),
     currency: v.currency.trim(), receivablePeriodId: v.receivablePeriodId || undefined,
     externalReference: v.externalReference.trim(), receivedAt: new Date(v.receivedAt),
     note: v.note?.trim() || undefined }` → `useCreateOfflinePayment(customerId).mutate(
     body, { onSuccess: toast + 关闭, onError: handleServerError })`。
2. 修改 `web/src/features/customers/receivable-periods-tab.tsx`：
   - Table 上方插入页首行（flex justify-between items-center）：左侧
     `<p className='text-xs text-muted-foreground'>{t('customers.receivablePeriods.offlinePayment.note')}</p>`
     （文案：无列表端点，已缴情况见各周期已付金额——Issue 明示）；右侧 Button
     （HandCoins 或 Banknote 图标）`offlinePayment.action` → setOfflineOpen(true)。
   - `<OfflinePaymentDialog open={offlineOpen} onOpenChange={setOfflineOpen}
     customerId={customerId} periods={allPeriods} />`。
   - 周期列表列/分页/外部发票 dialog **零触碰**。
3. i18n：`customers.receivablePeriods` 子树内**只追加** `offlinePayment` 子树：
   `{ action, note, title, description, idempotencyKey, idempotencyHint, amount,
   amountHint, currency, externalReference, receivedAt, period, periodPlaceholder,
   noteLabel, validation: { amount, currency, externalReference, receivedAt },
   toast: { created } }`。zh/en 同构；不改既有键。

## 约束

- 只动上述 3 个文件（2 改 1 新建）；不碰 hooks（T1 已就绪）。
- 金额换算仅元↔fen 一次性 BigInt 转换；币种照传不换算。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-28/web && pnpm build && pnpm lint
```

双 exit 0；无 routeTree 改动；locale 新子树 zh/en 键一致。

## 提交

```
git add src/features/customers/ src/i18n/locales/zh-CN.ts src/i18n/locales/en.ts
git commit -m "feat(admin): 客户线下支付登记 dialog (issue #28)"
```

## 报告

完整报告写入 `.superpowers/sdd/issue-28-offline-payment/task-2-report.md`，
回复只给四行：状态/提交哈希/一行测试摘要/疑虑。
