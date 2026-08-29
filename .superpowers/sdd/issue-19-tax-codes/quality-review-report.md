# Quality review — issue #19 (controller-executed, 2026-08-29 01:1x+08:00)

Verdict: **PASS** (0 Critical/Important outstanding after fix round 1).

## Programmatic checks (all fresh runs at final HEAD 1761e9e2f)

- `pnpm build` ✓ (248ms)
- `pnpm lint` ✓ (eslint 0 problems)
- `pnpm test:e2e` ✓ 2 smoke tests passed; walkthrough spec passed in the same
  infra before removal (6-step flow incl. create/edit/delete/switch)
- Locale parity: zh 589 = en 589 keys, zero drift (base 558 + 31)
- Anti-pattern scan on branch diff: clean ✓
- Diff scope == plan scope: 7 files, +782/−11 ✓
- Worktree clean besides gitignored ledger ✓

## Repo conventions

- ResourceKey regex identical to #2's FEATURE_KEY (cross-issue consistency noted by
  plan; verbatim).
- buildSchemas(t) mirrors the features dialog's per-locale schema rebuild pattern
  (the repo's settled answer to FormMessage semantics).
- ConfirmDialog props (destructive/isLoading/handleConfirm) match sibling usage.
- Badge/ghost-button row actions match plans/features list pages.

## Notable engineering findings (documented for future rounds)

- RHF array-container errors nest under `errors.appMappings.root` (empirically
  verified via live error-object dump). Future field-array forms with array-level
  refines must render `root.message` — recorded here for #15 (thresholds array) and
  #20.
- SDK query params serialize camelCase→snake_case on the wire (include_deleted) —
  relevant to every future hooks task's mock/walkthrough.

## Regressions / risks

- routeTree untouched; sidebar untouched ✓
- Pre-existing commerce sentinel messages + repo-wide prettier red: same as #17
  ledger (out of scope).
- Merge outlook: appends-only shared files; distinct sections from #7/#12/#17;
  final ordering at merge is cosmetic. Order #7 → #12 → #17 → #19.

## Remaining risks (for report)

1. Visual evidence programmatic (/tmp/issue19-shots 6 files, 1043–2449 unique
   colors); human eyeball recommended (same visual-channel limitation as #17).
2. include_deleted=true list response shape assumed identical (deletedAt badge
   renders from wire deleted_at) — mock covered, real backend per v3 spec.
3. Externalization (push/merge/close #19) NOT approved — awaiting user decision.
