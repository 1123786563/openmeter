# SDD ledger — plan: docs/superpowers/plans/issue-28-offline-payment.md

- 轨道：issue #28 线下支付登记；worktree /Users/wuyongjun/trea/openmeter-issue-28；
  分支 codex/admin-config-28；base f60cb90b0。
- Issue: https://github.com/1123786563/openmeter/issues/28

## Preflight 冲突扫描（dispatch 前）

| 交叠对 | 检查结论 |
|---|---|
| T1 hooks ↔ T2 dialog | T1 产出 useCreateOfflinePayment(customerId)，T2 消费；命名已固定 |
| T2 ↔ receivable-periods-tab 既有结构 | 仅在 Table 上方插入页首行（说明+按钮），列表列/分页/外部发票 dialog 零触碰 |
| T2 dialog ↔ external-invoice-dialog 模式 | reset-on-open（幂等键生成）/datetime-local→Date/handleServerError 全部镜像，不共享代码 |
| T2 i18n ↔ 既有 customers.receivablePeriods 子树 | 只追加 offlinePayment 子树 |
| 本轨 ↔ 并行轨 #22/#24/#26 | 不同 worktree；hooks 追加段与 locale 子树互不相交 |
| 计划文本自洽 | CommerceOfflinePaymentCreate 必填集 {idempotencyKey,amountFen,currency,externalReference,receivedAt} 与 Issue 处方一致；元→fen BigInt 先例已引用 |

Ruling: 币种=必填文本、选中周期时预填该周期 currency（不做全币种下拉——v3 无
币种枚举端点可用且周期内币种一致）。代价：手输错误币种由后端校验报错原文透出。

## SDD 模式

subagent 工具本轮可用 → 完整 SDD（同 #24 ledger 记录）。

## BASE

- T1 BASE: f60cb90b0

## T1 API 层

- Implementer: fresh subagent（f0b30a3c）DONE @ 566c8eb4e
  （+useCreateOfflinePayment；build/lint exit 0）。
- 环境备注：本轨首个发现并修复 SDK dist 缺失（file: 依赖），方法已广播。
- 审查中：review-0924043cf..566c8eb4e.diff。

- 审查（56bb03c0）：SPEC PASS 无发现；QUALITY PASS 无发现。Minor 备忘（可选）：
  hooks.ts:1129 段横幅 "Receivable periods & external invoices" 未含 offline
  payments 字样——语义仍准（同域），T2 若再加 hook 顺带更新。
- T1 complete（566c8eb4e，clean，零修复轮）。

## T2 dialog + tab 页首挂载 + i18n

- Implementer: fresh subagent（af76d549）DONE @ 7551c94c0（恰 4 文件；build/lint
  双 0；offlinePayment 子树求值 IDENTICAL；新文件 prettier clean）。
- 自查备忘：① Radix Select 禁空 value → 哨兵 '__none__' 表示不关联周期，
  wire 体无该键；② locale 保持纯插入。均待审查独立裁定。
- 审查中：review-566c8eb4e..7551c94c0.diff。

- 审查（29aedf32）：SPEC PASS 无发现；QUALITY PASS。独立验证亮点：哨兵
  __none__ 两条 wire 不变量成立（表单态只有 ''/id 两态，|| undefined 兜底，
  序列化丢弃 undefined）；amount 正则+BigInt 边界（Math.round 仅补浮点误差，
  不真实舍入）；period→currency setValue({shouldValidate}) 联动；schema 移入
  useMemo([t]) 是本地化错误键的必要路径（模板为模块级 schema 的既有局限）。
- Minor 备忘（代码零改动）：实现者报告称「Radix Select 禁空 value 抛错」系
  过时信息——安装版 @radix-ui/react-select@2.3.7 无该运行时 guard；哨兵方案
  本身正确且更稳，后续任务勿照搬该「禁令」表述。
- T2 complete（7551c94c0，clean，零修复轮）。

## 终态门禁

- e2e（串行，/tmp/issue-round/e2e-28.log）：sign-in smoke PASS；customers smoke
  FAIL——与原始基线同签名（e2e-base.log）→ 既有环境问题非本轨回归。
- locale parity：offlinePayment 子树求值 IDENTICAL。

- 全分支终审（b4a104b7）：FINAL NEEDS_FIX——1 项：金额校验正则放行 "0"/"0.00"
  → amountFen 0n 可提交（后端 stub 无金额校验 201）；与自身文案「正数」、计划
  处方、先例 recharge-product-form-dialog.tsx:66-69 的 >0 分句三重基准矛盾。
  修复=一行 .refine(v => POSITIVE_AMOUNT.test(v) && Number(v) > 0, msg)。
  观察不计发现：invalidate 默认只 refetch active 查询（#27 同款继承模式，
  计划非目标）。
- 修复轮 1 派发。

- 修复轮 1（a5469f9e）：20e5cc9c7——.refine(test && Number>0) 与先例同构，
  message 沿用原键，build/lint 双 0。
- 修复复审（7b33cee1）：FIX PASS 无发现（零值全拒/正数通过/无 NaN 边界、
  issue code 变化不影响 FormMessage 渲染、BigInt 零改动、恰一文件）。
- 终审唯一发现闭环 → FINAL 判定转 RELEASE_READY。
- 轨道终态：T1 566c8eb4e + T2 7551c94c0 + fix1 20e5cc9c7，门禁全绿。
