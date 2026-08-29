# Spec review — issue #21 (apps installed / uninstall / Stripe key)

Reviewer: controller (programmatic, unattended DOWNGRADE mode). Target:
9b0d34d73. Result: **17/17 PASS**, 0 fix rounds.

| # | Spec item (issue #21 comment contract) | Verdict | Evidence |
|---|---|---|---|
| S1 | Route /config/apps renders AppsPage | PASS | routes/_authenticated/config/apps.tsx → AppsPage |
| S2 | useApps list via SDK, page size 100 | PASS | hooks.ts `api.internal.apps.list({page:{number:1,size:100}},{signal})` |
| S3 | Table columns: 应用名称/类型/状态/能力/Stripe 信息/操作 | PASS | index.tsx header set matches plan |
| S4 | type+status badge variants (secondary/outline/ready/unauthorized) | PASS | Badge variant mapping per status |
| S5 | capability badges 用量上报/支付收款/客户开票 via definition.capabilities | PASS | capability label map, one Badge per item |
| S6 | stripe row shows accountId, maskedApiKey, livemode 正式模式/测试模式 | PASS | stripeInfo cell |
| S7 | empty state row colSpan=6 | PASS | apps.length === 0 branch |
| S8 | 卸载 → ConfirmDialog destructive with app name in text | PASS | uninstallConfirm locale + dialog |
| S9 | uninstall mutation invalidates nsPrefix('apps') | PASS | useUninstallApp onSuccess |
| S10 | 换 Key button only for type==='stripe' (union narrowing, no casts) | PASS | app.type === 'stripe' branch |
| S11 | stripe-key-dialog zod secretApiKey min(1) | PASS | schema + zodResolver |
| S12 | PUT body {type:'stripe', name, description?, labels?, secretApiKey} | PASS | wire-verified deep-equal incl. snake_case secret_api_key (walkthrough) |
| S13 | success toast + dialog close; server error via handleServerError | PASS | dialog component |
| S14 | query-keys apps key + ns prefix invalidation | PASS | query-keys.ts |
| S15 | zh/en config.apps complete subtree, parity | PASS | 622=622 leaves, 28/28 static keys resolve |
| S16 | e2e specs untouched | PASS | git diff scope excludes e2e/ |
| S17 | no `api.apps.` residue (D-1 applied everywhere) | PASS | grep 0 matches outside comments |

Deviations: D-1 (api.apps → api.internal.apps), D-2 (hooks anchor section)
— both pre-verified against SDK/types and recorded in the ledger.
