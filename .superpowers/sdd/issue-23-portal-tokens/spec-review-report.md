# Spec review — issue #23 (portal token issuance)

Reviewer: controller (programmatic, unattended DOWNGRADE mode). Target:
540072fc8. Result: **17/17 PASS**, 0 fix rounds.

| # | Spec item (issue #23 comment contract) | Verdict | Evidence |
|---|---|---|---|
| S1 | Route /config/portal-tokens renders PortalTokensPage | PASS | route file |
| S2 | legacy.ts PortalToken interface incl. token 一次性注释 | PASS | api/legacy.ts |
| S3 | createPortalToken POST /v1/portal/tokens | PASS | wire-verified in walkthrough |
| S4 | query-keys portalTokens(params) ready for #24 list | PASS | query-keys.ts |
| S5 | useCreatePortalToken invalidates nsPrefix('portal-tokens') | PASS | hooks.ts |
| S6 | issue dialog: CustomerPicker + meter multi-select (Popover+Command+Checkbox) | PASS | issue-token-dialog.tsx |
| S7 | meter 空选 = 全部（提示文案），非空 = allowedMeterSlugs | PASS | walkthrough body deep-equal both cases |
| S8 | customer required: submit without customer → 请选择客户 + ZERO POST | PASS | walkthrough scenario 1 |
| S9 | subject = customer.key | PASS | wire body subject:'acme-corp' |
| S10 | response.token 缺失 → toast.error noPlaintext（不静默） | PASS | mutation onSuccess guard |
| S11 | once-dialog: om_portal_ 前缀展示、一次性警示 | PASS | token-once-dialog + walkthrough |
| S12 | copy: clipboard 优先 + execCommand 降级，状态归属 token | PASS | copiedToken/failedToken derivation |
| S13 | onPointerDownOutside 防误关 | PASS | DialogContent prop |
| S14 | 关闭后明文不可再现（重载后 count 0） | PASS | walkthrough reload assertion |
| S15 | PageHeader actions 发放令牌按钮 | PASS | index.tsx |
| S16 | 无 effect 内 setState（D-3 适配） | PASS | lint 0 errors |
| S17 | e2e untouched / zh-en parity 608=608 | PASS | git scope + parity script |

Deviations: D-3 (lint adaptation of prescriptive reset-on-open effect —
semantics-equivalent event-handler reset). No other deviations.
