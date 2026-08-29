# Quality review — issue #23

Reviewer: controller (programmatic checks + careful pass). Target: 540072fc8.
Result: **PASS**.

- Diff scope: exactly the 9 planned files (legacy.ts, query-keys.ts, hooks.ts,
  token-once-dialog.tsx, issue-token-dialog.tsx, index.tsx, route, zh-CN.ts,
  en.ts). e2e untouched; no new dependencies.
- Type safety: zero `as` casts, zero `any`.
- v1-legacy pattern followed: apiFetch + typed helpers, no SDK mixing for
  the v1 surface.
- D-3 adaptation audited: observable behavior equivalent to the prescriptive
  comment code (fresh form per open; indicators reset per token), while
  satisfying react-hooks/set-state-in-effect. No hidden state leaks between
  opens (verified by walkthrough scenario 3 issuing a second token).
- Comments capture domain intent: 一次性明文语义 (server returns token only at
  creation), subject 语义 (customer.key not display name), clipboard 降级路径.
- i18n: full config.portalTokens subtree both locales; parity 608=608;
  23/23 static keys resolve.
- Build/lint gates green per task commit (logs /tmp/i23-t*.log).
- Risks accepted: meter multi-select reads one page of meters (size 100) —
  same pagination shape as neighbors; #24 will add the list table reusing
  the portalTokens key.
