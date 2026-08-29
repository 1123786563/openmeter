# Final Whole-Branch Adversarial Review Report — Issue #5

- Reviewer role: FINAL adversarial whole-branch reviewer (audits the three task-level reviews; assumes everything prior could be wrong)
- Worktree: `/Users/wuyongjun/trea/openmeter-issue-5`, branch `codex/admin-config-05`, HEAD = `a6ff556ef019746861ff85e3858ca657bc46405d`
- Review range: `45f89e50d..HEAD` = plan commit `c4119ca32` (adds only `docs/superpowers/plans/issue-5-plan-lifecycle.md`, +65) + implementation commit `a6ff556ef` (`feat(admin): 计划发布/归档/克隆新版本`, 5 files, +243/−2)
- Authoritative spec: `gh issue view 5 --repo 1123786563/openmeter --comments` (1 comment, the prescription; fetched and compared hunk-by-hunk)
- Audited reports: spec-review-report.md (PASS), quality-review-report.md (PASS), browser-walkthrough-report.md (PASS, 43/43)
- No fixes applied; no tracked files modified; no commits/branches/pushes; no GitHub writes; no subagents. Only this report file was written.

## Angles run (all 11 — none skipped)

### 1. i18n deep-compare — RUN, PASS

Independent script (own code, not the quality reviewer's): whole-file flatten en=528 keys, zh=528 keys, zero keys present in only one locale; `config.plans` subtree en=40, zh=40, identical key-path sets, zero interpolation mismatches (`{{name}}` / `{{version}}`). All 12 new keys (`actions.publish/archive/cloneNext`, `publishConfirm.title/description`, `archiveConfirm.title/description`, `cloneConfirm.title/description`, `toast.published/archived/cloned`) exist in BOTH locales. Reference audit: every new key is consumed in `plan-detail.tsx` (`actions.*` ×2 each — button + confirmText; others ×1); zero references to these keys anywhere else; zero dead keys. `common.cancel` pre-exists in both ("Cancel"/"取消"). Duplicate keys: none — a name-level textual scan flags `title×4/description×2/status×2/key×2`, all at different nesting paths (path-level sets are strictly equal and `tsc -b` rejects object-literal duplicates via TS1117). Known-context check: `plan.status.active` zh = "已发布" confirmed pre-existing (present in base `45f89e50d` at zh-CN.ts:291; the diff adds only `config.plans.*` keys).

### 2. Button visibility rules vs spec — RUN, PASS

`plan-detail.tsx:145-167` re-read at HEAD: draft→发布 only (145-149); active→归档 (150-158, `variant='outline'`); `plan.status !== 'draft'`→克隆新版本 (159-167, `variant='outline'`); every condition ANDed with `!statusBusy` (hidden, not disabled). SDK typing `Plan.status: 'draft'|'active'|'archived'|'scheduled'` verified in `node_modules/@openmeter/client/dist/models/types.d.ts:5460` — the four states map to: draft→{publish}, active→{archive,clone}, scheduled→{clone}, archived→{clone}, exactly the spec matrix. publish and clone are mutually exclusive (`=== 'draft'` vs `!== 'draft'`); no status shows two wrong buttons; the union type makes `!== 'draft'` exhaustive-correct — nothing slips through unhandled.

### 3. Mutation wiring — RUN, PASS

`usePublishPlan`/`useArchivePlan` call `api.plans.publish/archive` with `{ planId }` via `Parameters<typeof …>[0]` typing (hooks.ts:204-228); SDK signatures `publish(request: PublishPlanRequest): Promise<PublishPlanResponse>` / `archive(...)` with `PublishPlanRequest/ArchivePlanRequest = { planId: string }` verified in dist. Clone: `clonePlanNextVersion(plan.id)` → `apiFetch('/v1/plans/{encodeURIComponent(id)}/next', { method: 'POST' })` (legacy.ts:107-116); `API_BASE='/api'` prefix (lib/api.ts:4,45) yields exactly the v1 spec path. v1 contract verified at `api/openapi.yaml:7362-7382`: 201→v1 `Plan`; description "It returns error if there is already a plan in draft or planId does not reference the latest published version" — so cloning from an archived v1 while a newer published version exists is a documented backend 4xx, and the spec explicitly prescribes no frontend pre-check (「不做前端预判」). Grep confirms zero draft-exists/status pre-check logic anywhere in the diff. Walkthrough wire log corroborates: `POST /api/v3/openmeter/plans/plan-draft-1/publish`, `POST /api/v3/openmeter/plans/plan-active-1/archive`, `POST /api/v1/plans/plan-active-1/next` (201 and 409), all bodyless.

### 4. Invalidation semantics — RUN, PASS

`nsPrefix(domain)` = `['api', ns, domain]` (hooks.ts:779-781). Segment-exact prefix matching means `nsPrefix('plans')` covers `usePlans` (`['api',ns,'plans']` — subscription wizard `create-subscription-dialog.tsx:41`, subscriptions list `index.tsx:59`, subscription detail `subscription-detail.tsx:137`) but NOT `plans-page`; the explicit `nsPrefix('plans-page')` covers the config list page (`config/plans/index.tsx:29`, every filter/page combination), and `queryKeys.plan(id)` covers the detail page. Publish/archive return the same id → detail key invalidated → refetch (wire log shows GET detail + GET list fired right after each POST). Clone returns a NEW id and immediately navigates to it → fresh query mounts (wire: GET `/api/v3/openmeter/plans/plan-draft-2` after the 201). Global `staleTime: 10s` (main.tsx:37) does not block this: `invalidateQueries` marks stale and refetches active queries regardless of staleTime. No plans-data consumer is left stale; no other plans/plans-page/plan-keyed query exists (grep-verified).

### 5. Error paths, dialog state machine, double-submit — RUN, PASS (one Minor finding, pre-existing pattern)

`onError: handleServerError` on all three mutations; `confirming` is never reset on error → dialog stays open for retry (walkthrough scenario d DOM-verified: alertdialog visible, page alive after 409). Single-value state machine `'publish'|'archive'|'clone'|null` → at most one dialog open; close-then-reopen cannot cross-wire mutations (each dialog's closure binds its own mutation). Double-submit: `ConfirmDialog` disables confirm via `disabled={disabled || isLoading}` (confirm-dialog.tsx:64) AND cancel via `disabled={isLoading}` (:56); header buttons hidden while `statusBusy` → no second POST can fire while one is in flight. NEW finding (Minor, pre-existing app-wide, not a branch defect): `main.tsx:40-42` `defaultOptions.mutations.onError` already calls `handleServerError` globally, and query-core 5.99.0 fires BOTH the mutation-level and the per-`mutate()` `onError` (verified in installed `query-core/build/modern/mutation.js:159` + `mutationObserver.js:109`) → a failed mutation shows the error toast twice. The per-mutate `onError: handleServerError` is prescribed verbatim by the issue and mirrors the sibling precedent (`features/index.tsx:183`), so every feature mutation in this app already double-toasts; the branch introduced no regression.

### 6. LegacyPlan type honesty — RUN, PASS

Field-for-field against `api/openapi.yaml:22777+` v1 `Plan`: id/name/key/currency/billingCadence/status/createdAt/updatedAt all string-backed ✓; `version` integer→`number` ✓ (openapi.yaml:22841-22842); camelCase names exact ✓. `status: string` is a deliberate widening of the v1 `PlanStatus` enum (draft/active/archived/scheduled, openapi.yaml:23352-23358); the clone result only consumes `id` and `version`, so the superset is harmless and documented. No drift; the interface is the prescription's exact 9-field subset.

### 7. Regressions — RUN, PASS

Implementation commit touches exactly the 5 planned files (`git show --stat`: hooks.ts, legacy.ts, plan-detail.tsx, en.ts, zh-CN.ts; +243/−2 — the 2 deletions are the two replaced import lines). The plan-detail diff is pure additions (plus those import lines); existing rendering (InfoRows card, phases/rate-cards tables, skeleton/notFound early returns at :105-127) untouched. Plans list page untouched; `routeTree.gen.ts` absent from the diff and no route change is needed (navigation targets the existing `/_authenticated/config/plans/$planId` route id already used by `useParams` at :88); e2e specs untouched. Known-context verification: the only prettier violation in `hooks.ts` is the `api.commerce.listRechargeProducts(...)` wrapping hunk — byte-identical in base (`c4119ca32`, lines 485-488) and HEAD (526-529), confirmed by diffing prettier's output for both revisions under the project config; the other 4 files are prettier-clean.

### 8. Cross-commit consistency — RUN, PASS

Plan commit `c4119ca32` adds only the 65-line plan document, which pre-records exactly 3 deviations (i18n indentation adapted; imports merged into existing statements; dialogs inside the main return branch after the `if (!plan)` early return) — all three verified real in the implementation. Implementation commit message is byte-exact `feat(admin): 计划发布/归档/克隆新版本`, matching the prescription.

### 9. Security — RUN, PASS

Added lines contain no secrets/tokens, no new dependencies (package.json/lockfile untouched), no `eval`/`new Function`/dynamic `import()`/`require`/`innerHTML`/`dangerouslySetInnerHTML` (grep over `^+` diff lines: zero matches). Error toasts: `handleServerError` extracts strings from the error body and calls `toast.error(string)` (handle-server-error.ts:47); sonner renders strings as React text nodes — server-controlled error messages are escaped, no HTML injection path.

### 10. Evidence audit — RUN, PASS

Rerun by this reviewer, real exit codes: `pnpm build` → `✓ built in 356ms`, **exit 0**; `pnpm lint` → no output, **exit 0**. (`pnpm test:e2e` not rerun here — brief sets build+lint as the minimum; two prior independent reviewers ran it with exit 0, 2 passed.) Artifacts: `/tmp/issue5-shots/` holds exactly 9 PNGs, 81,905–118,652 bytes each, sane sizes; `sips` confirms `draft-detail-zh.png` = 1440×900. `/tmp/issue5-walkthrough-results.json` (10,688 bytes): 43 checks / 0 failed (login 1, a 6, b 12, c 7, d 4, e 5, f 6, g 1, h 1); `pageErrors: 0`; `consoleErrors` = exactly the expected 409 fetch line; 20 wire records matching the report's POST/GET table verbatim (paths, bodyless POSTs, `filter[status]=draft` encoding, the 409 second-clone); pixel metrics for all 9 screenshots and all 5 pairwise diffs match the report's table digit-for-digit. Chain of custody: walkthrough `startedAt 2026-08-27T12:01:43Z` postdates the implementation commit (`2026-08-27T07:36:47Z`), the worktree has been clean since (only the untracked `.superpowers/sdd/` dir), and progress.md's dispatch order (implement → commit → reviews) is consistent — the evidence corresponds to this HEAD to the extent /tmp artifacts allow. Ledger consistency: progress.md's three rulings and supporting numbers (build/lint/e2e exit 0, 40=40 locale keys, 9 shots 1440×900, unique colors 1004–1422) match the three reports and my reruns. Two cosmetic discrepancies noted in Findings #3/#4.

### 11. AGENTS.md conventions in new code — RUN, PASS

No `any`, `as`-casts, `@ts-ignore`, `@ts-expect-error`, or `eslint-disable` in any added line (grep over `^+` diff lines: zero matches). No new helpers extracted (nothing to violate the trivial-helper rule); the one new code comment (hooks.ts:238 zh) is verbatim from the issue prescription — known pre-approved context, re-verified identical. Comments carry intent (endpoint doc comment on `LegacyPlan`), not narration.

## Findings

| # | Severity | Location | Description |
|---|---|---|---|
| 1 | Minor (pre-existing pattern, surfaced by this feature) | web/src/main.tsx:40 + plan-detail.tsx:286,311,342 | Global `defaultOptions.mutations.onError` already toasts via `handleServerError`; query-core fires it AND the per-`mutate` `onError: handleServerError` → duplicate error toast on a failed publish/archive/clone. Prescription-verbatim and identical to sibling precedent `features/index.tsx:183`; app-wide pre-existing wiring, not a branch regression. Walkthrough scenario d still passes (toast text assertion is unaffected). |
| 2 | Minor (spec-sanctioned design consequence) | plan-detail.tsx:330; api/openapi.yaml:7368 | Clone passes the viewed plan's `id`; cloning from an archived/scheduled version that is not the latest published version will always backend-4xx ("planId does not reference the latest published version") and surface as a raw toast. The spec explicitly mandates this no-pre-check passthrough; recorded so the controller presents it as intended behavior, not a bug. |
| 3 | Minor (report-side cosmetic) | browser-walkthrough-report.md:3; progress.md:24 | Walkthrough report date reads 2025-08-27 (actual 2026, per JSON `startedAt` and commit dates); progress.md's "关键成对差异 1.2%/99.9%/7.7%" paraphrase doesn't map cleanly onto the raw pairwise diffs (0.675/15.086/106.313 of 255) — the report's own raw table is the accurate one. No ruling impact. |
| 4 | Minor (report-side prose miscounts) | spec-review-report.md:42 ("13 new key values"); quality-review-report.md:43 ("16 added keys") | Actual new-key count is 12 per locale (config.plans 40 = 28 pre-existing + 12). Both reports' programmatic outputs (40=40, IDENTICAL) were correct; only the prose counts are off. |
| 5 | Minor (observation) | plan-detail.tsx:270,294,319 | Radix AlertDialog's Escape key path can still close a dialog mid-flight (`onOpenChange(false)` → `setConfirming(null)`; the disabled cancel button does not gate Escape). The mutation completes regardless and its `onSuccess` toast/navigate still fires, so no success path is lost; matches the app's existing dialog pattern. |

Critical: none. Important: none. All five findings are Minor; none is a defect introduced by this branch's diff, and none blocks presentation.

## Verdict

The branch is a faithful, verbatim realization of the issue #5 prescription across its full range (plan commit + implementation commit): the 3 pre-approved deviations are real and benign, the implementation commit message is exact, all evidence reruns pass with recorded exit codes (build 0, lint 0), the three task-level PASS rulings survive adversarial audit — their claims reproduced independently (locale key sets, invalidation wiring, SDK/openapi contracts, wire log, screenshots, pixel metrics) — and the known pre-existing contexts (recharge prettier hunk, prescription zh comment, pre-existing 「已发布」 badge key) all re-verified as still true. No cross-commit interaction, regression, security, i18n-integrity, or evidence-authenticity issue was found that the task-level reviews missed beyond the Minor items above.

Ruling: PASS
