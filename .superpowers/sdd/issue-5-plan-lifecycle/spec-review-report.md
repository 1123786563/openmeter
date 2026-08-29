# Spec Review Report — Issue #5 Task 1

- Reviewer role: independent SPEC-COMPLIANCE reviewer (implementation vs. issue prescription only; code style out of scope)
- Worktree: `/Users/wuyongjun/trea/openmeter-issue-5`, branch `codex/admin-config-05`, HEAD = `a6ff556ef`
- Diff reviewed: `git diff c4119ca32..a6ff556ef` (single commit `feat(admin): 计划发布/归档/克隆新版本`)
- Authoritative spec: GitHub issue 5 prescription comment (read via `gh issue view 5 --repo 1123786563/openmeter --comments`, read-only)
- Plan: `docs/superpowers/plans/issue-5-plan-lifecycle.md` (3 pre-approved deviation risks re-verified below)

## Checklist verification

### 1. `web/src/api/legacy.ts` — PASS

- `LegacyPlan` interface with exactly the prescription's field set — id/name/key/version/currency/billingCadence/status/createdAt/updatedAt, all camelCase, same types (`legacy.ts:94-104`). Note: the brief's phrase "exactly the 10 fields" miscounts — it then lists 9 names, and the prescription comment itself defines exactly these 9 fields; the implementation matches the prescription verbatim. (Recorded as Minor observation #2, a brief-side miscount, not an implementation defect.)
- Doc comment present above the interface: `/** POST /api/v1/plans/{planIdOrKey}/next — clone the latest published version into a new draft. */` (`legacy.ts:93`), identical to the prescription.
- `clonePlanNextVersion(planIdOrKey)` POSTs `/v1/plans/${encodeURIComponent(planIdOrKey)}/next` via `apiFetch` with `method: 'POST'` (`legacy.ts:106-115`). Only formatting differs from the comment code (signature and call wrapped across lines) — prettier-governed, benign.

### 2. `web/src/api/hooks.ts` — PASS

- `usePublishPlan` (hooks.ts:204-215), `useArchivePlan` (:217-228), `useClonePlanNext` (:230-243) all placed directly after `usePlan` (:196) and before the Invoices section (:245) — "after `usePlan`" satisfied.
- `mutationFn` typing via `Parameters<typeof api.plans.publish>[0]` / `[0]` of archive, pass-through call — verbatim per prescription (:207-208, :220-221).
- `onSuccess` invalidates exactly the trio `nsPrefix('plans')` + `nsPrefix('plans-page')` + `queryKeys.plan(plan.id)` in all three hooks — no more, no less; matches the prescription and the `useCancelSubscription` pattern (:209-212, :222-225, :234-238).
- `useClonePlanNext` calls `clonePlanNextVersion` with the plan-detail invalidation line carrying the v1-id-is-v3-id rationale comment (`// v1 id 是 v3 同一资源 id；新 draft 详情可直接用它进入。`, :236).
- `clonePlanNextVersion` merged into the existing `'@/api/legacy'` import statement (hooks.ts:9); `LegacyPlan` is not used in hooks.ts and is correctly not imported.

### 3. `web/src/features/config/plans/plan-detail.tsx` — PASS

- Imports: `useState` (line 1); `useNavigate` merged into the existing `@tanstack/react-router` import (line 2); `toast` from `'sonner'` (line 6); the three new hooks merged into one `'@/api/hooks'` import (lines 7-12); `handleServerError` (line 14); `ConfirmDialog` (line 26). All per prescription.
- Component body: `navigate` (line 91); `confirming` state typed `'publish' | 'archive' | 'clone' | null` (lines 92-94); three mutations (lines 96-98); `statusBusy` = OR of the three `isPending` (lines 100-103).
- Title-row buttons with exact visibility rules (lines 145-167):
  - draft → 发布: `<Button size='sm' onClick={...}>` — `size='sm'`, no `variant` prop = default variant (lines 145-149). Matches "no variant spec = default".
  - active → 归档: `size='sm' variant='outline'` (lines 150-158).
  - `plan.status !== 'draft'` → 克隆新版本: `size='sm' variant='outline'` (lines 159-167).
  - All three gated by `!statusBusy &&` — hidden, not disabled, while any mutation is pending, exactly as the prescription's code block and the brief specify.
- Three `ConfirmDialog`s appended at the end of the main return branch (lines 268-346), after the `if (!plan)` early return (lines 116-127) — NOT rendered when plan is null/undefined; `plan.name`/`plan.version` in dialog descs are therefore safe. Pre-approved deviation (3) confirmed real and correctly applied.
- Publish dialog (lines 268-290): `confirmText` = `t('config.plans.actions.publish')`, `cancelBtnText` = `t('common.cancel')` (exists in both locales: zh-CN.ts:6, en.ts:6), `mutate({ planId: plan.id })`, `onSuccess` → `toast.success(t('config.plans.toast.published'))` + `setConfirming(null)`, `onError: handleServerError`. Matches.
- Archive dialog (lines 292-315): same shape plus `destructive` prop (line 301). Matches.
- Clone dialog (lines 317-346): `mutate({ planIdOrKey: plan.id })` (line 330); `onSuccess` → `toast.success(t('config.plans.toast.cloned', { version: draft.version }))` (lines 333-335), `setConfirming(null)` (line 336), `void navigate({ to: '/config/plans/$planId', params: { planId: draft.id } })` (lines 337-340). Matches.

### 4. i18n (`zh-CN.ts` / `en.ts`) — PASS

- Both locales gain, under `config.plans`: `actions{publish,archive,cloneNext}`, `publishConfirm{title,description}`, `archiveConfirm{title,description}`, `cloneConfirm{title,description}`, `toast{published,archived,cloned}` (zh-CN.ts:137-161, en.ts:140-164).
- All 13 new key values are verbatim identical to the prescription code blocks in both locales (verified programmatically by loading both locale modules and comparing the `config.plans` trees).
- Structural identity: both `config.plans` trees have 40 leaf keys in identical order; zero key-path differences; zero interpolation mismatches ({{name}} in all three confirm descriptions, {{version}} additionally in cloneConfirm.description, {{version}} in toast.cloned).
- No other `config.plans` keys touched — the diff hunks are pure insertions after `detail.tierSummary`; context lines (`unit`, `tierSummary`) unchanged. Pre-approved deviation (1) (indentation adapted from the comment's 10/12-space to the actual file's 6/8-space) confirmed real and benign: keys/levels/values unchanged.

### 5. No scope creep — PASS

- Diff touches exactly the 5 prescribed files (git show --stat: hooks.ts, legacy.ts, plan-detail.tsx, en.ts, zh-CN.ts; 243 insertions, 2 deletions — the 2 deletions are the replaced import lines in plan-detail.tsx).
- No backend, api spec, v3/v1 SDK artifact, plans list page, or StatusBadge changes; no draft-exists frontend pre-check (its absence is CORRECT per prescription: backend 4xx passes through via toast); no new e2e specs.

### 6. Commit & worktree — PASS (one Minor observation)

- Single implementation commit `a6ff556ef` with message exactly `feat(admin): 计划发布/归档/克隆新版本`.
- Tracked worktree clean; only dirt is `.superpowers/sdd/issue-5-plan-lifecycle/`. Minor observation #1: the brief calls this directory "gitignored", but it is in fact merely untracked — no `.gitignore`/`.git/info/exclude` entry covers `.superpowers` (`git check-ignore` exits 1). This is a brief-side description inaccuracy; no tracked file is affected and nothing leaks into the commit.

### 7. Recorded deviations + SDK contract — PASS

- Deviation (1) i18n indentation adapted: real (see §4), benign.
- Deviation (2) import merging to actual file: real (see §2/§3 — `useNavigate` merged into the router import; hooks merged into one `@/api/hooks` import; `clonePlanNextVersion` merged into the legacy import), benign.
- Deviation (3) dialogs inside the main return branch after the early return: real (see §3), benign.
- SDK contract re-checked in worktree node_modules: `publish(request: PublishPlanRequest): Promise<PublishPlanResponse>` and `archive(request: ArchivePlanRequest): Promise<ArchivePlanResponse>` (dist/sdk/plans.d.ts:65,73); `PublishPlanRequest = { planId: string }` / `ArchivePlanRequest = { planId: string }` (dist/models/operations/plans.d.ts:39-45); `Plan.status: 'draft' | 'active' | 'archived' | 'scheduled'` (dist/models/types.d.ts:5460). The implemented calls (`{ planId: plan.id }`) and the draft/active/非-draft button rules are fully consistent with these typings; no comment-vs-SDK conflict arose.

### 8. Verification commands (run by reviewer) — PASS

From `/Users/wuyongjun/trea/openmeter-issue-5/web`:

| Command | Result | Exit code |
|---|---|---|
| `pnpm build` | `tsr generate && tsc -b && vite build` — `✓ built in 305ms` | **0** |
| `pnpm lint` | `eslint .` — no output, no warnings | **0** |
| `pnpm test:e2e` | `2 passed (6.2s)` — sign-in smoke + customers smoke (e2e/smoke.spec.ts:26, :38) | **0** |

## Findings

- Critical: none.
- Important: none.
- Minor:
  1. Minor (process observation, not an implementation defect): `.superpowers/sdd/issue-5-plan-lifecycle/` is untracked, not gitignored as the brief states — `git check-ignore` exit 1, no `.gitignore` entry. Tracked tree is clean; commit content unaffected.
  2. Minor (brief-side miscount, not an implementation defect): brief item 1 says `LegacyPlan` has "exactly the 10 fields" but lists 9; the prescription comment defines exactly 9 fields and the implementation matches it field-for-field.

## Conclusion

The implementation is a faithful, verbatim realization of the issue prescription (modulo the three pre-approved, benign formatting/placement deviations recorded in the plan). No spec violations, no scope creep, and all three verification commands pass with exit code 0.

Ruling: PASS
