# Quality review — issue #21

Reviewer: controller (programmatic checks + careful pass). Target: 9b0d34d73.
Result: **PASS**.

- Diff scope: exactly the 7 planned files (query-keys.ts, hooks.ts,
  stripe-key-dialog.tsx, features/config/apps/index.tsx, route file, zh-CN.ts,
  en.ts). No stray files; e2e untouched; no new dependencies.
- Type safety: zero `as` casts, zero `any` (grep verified). Union narrowing
  used for stripe-only affordances.
- Repo patterns: nsPrefix invalidation, handleServerError, ConfirmDialog,
  PageHeader, Badge variants, section comments in hooks/locales — consistent
  with neighbors (notification channels section).
- i18n discipline: `as const` default-export shape preserved; leaf parity;
  capability labels centralized in locale, not hardcoded Chinese.
- React Hooks lint: no set-state-in-effect patterns introduced.
- Comments: intent-capturing (why masked key shown, why stripe-only button).
- Build/lint/format gates green at every task commit (logs /tmp/i21-t*.log).
- Risks accepted: SDK wire casing relied on (verified by wire-level
  walkthrough assertion secret_api_key); uninstall without re-fetch of
  catalog is per plan scope (catalog page is #20/#22 territory).
