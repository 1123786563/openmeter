# Quality Review Brief — Issue #5 Task 1

You are an independent CODE-QUALITY reviewer. Judge ONLY engineering quality and repo-convention compliance (not spec compliance — that's another reviewer).

Repo/worktree: /Users/wuyongjun/trea/openmeter-issue-5 (branch codex/admin-config-05)
Read first: /Users/wuyongjun/trea/openmeter-issue-5/AGENTS.md (repo conventions — Go parts not applicable here; frontend follows existing patterns), the plan docs/superpowers/plans/issue-5-plan-lifecycle.md.
Diff to review: `git diff c4119ca32..a6ff556ef` (skip plan-commit content).

Review axes:
1. TypeScript: no any / as-casts / @ts-ignore / eslint-disable in ADDED lines; Parameters<> typing of mutationFn inputs; LegacyPlan field types match v1 Plan (version: number, not string); no unused imports.
2. React/TanStack patterns: the three mutation hooks mirror useCancelSubscription's invalidation style exactly; invalidations are fire-and-forget `void` prefixed; setConfirming(null) on onSuccess and dialog onOpenChange cancel path both handled; no stale-closure hazards; statusBusy correctly derives from three isPending flags; dialogs not rendered in the !plan early-return branch (no null deref on plan.name).
3. Repo conventions (#2–#4 precedents): prettier formatting of added code (printWidth 80, single quotes, import order); i18n trees in zh-CN.ts and en.ts structurally identical — verify programmatically (node script comparing full key paths of the config.plans subtree AND whole-file key-set equality); no hardcoded UI copy outside t() in added lines (font-mono/`v{plan.version}` display join is allowed — it's a value prefix not copy); toasts via sonner consistent with sibling pages.
4. Query hygiene: invalidation trio (plans / plans-page / plan detail) sufficient for list+detail refresh after publish/archive/clone? Consider: detail page listens on queryKeys.plan(id) — publish returns plan with SAME id, invalidation correct; clone returns NEW id — navigating to the new detail fires a fresh query anyway; list page uses plansPage key. Any missed key that would show stale data (e.g. any other page-keyed query on plans)? Check query-keys.ts and the plans list page implementation for the exact keys used.
5. UX/edge: double-click protection (ConfirmDialog isLoading while pending + buttons hidden via statusBusy); error path — handleServerError shows toast, dialog stays open (confirming not reset on error): is that consistent with the sibling features page precedent (config.features toggleConfirm)? If sibling behaves the same, classify Minor at most.
6. Regressions: only the 5 files in the stat touched; no route changes; existing smoke e2e unaffected (run them).
7. Runs you must do yourself: `cd web && pnpm build && pnpm lint` (exit codes), `pnpm test:e2e` (2 passing), node-based locale key-tree comparison zh vs en (paste the script + output into the report).

Verdict: PASS or FAIL, findings Critical / Important / Minor with file:line evidence. Run-only claims must carry actual exit codes. No drive-by fixes — report only. Write your report to /Users/wuyongjun/trea/openmeter-issue-5/.superpowers/sdd/issue-5-plan-lifecycle/quality-review-report.md with heading `# Quality Review Report — Issue #5 Task 1` and a final line `Ruling: PASS|FAIL`.
