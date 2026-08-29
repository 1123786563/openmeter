# Spec Review Brief — Issue #5 Task 1

You are an independent SPEC-COMPLIANCE reviewer. Judge ONLY whether the implementation matches what the issue and its prescriptive comment asked for. Code style is another reviewer's job.

Repo/worktree: /Users/wuyongjun/trea/openmeter-issue-5 (branch codex/admin-config-05)
Plan: docs/superpowers/plans/issue-5-plan-lifecycle.md (read it; its「SDK 契约核实记录」section lists 3 known deviation risks that were pre-approved)
Authoritative sources: `gh issue view 5 --repo 1123786563/openmeter --comments` (the prescription comment with complete code is the spec) — read-only on GitHub. No GitHub writes.
Diff to review: `git diff c4119ca32..a6ff556ef` (c4119ca32 = plan commit, a6ff556ef = the single implementation commit `feat(admin): 计划发布/归档/克隆新版本`).

Checklist (verify EACH against the actual diff, cite file:line):
1. web/src/api/legacy.ts: `LegacyPlan` interface exactly the 10 fields (id/name/key/version/currency/billingCadence/status/createdAt/updatedAt, camelCase); `clonePlanNextVersion(planIdOrKey)` POSTs `/v1/plans/${encodeURIComponent(planIdOrKey)}/next` via apiFetch with method POST; doc comment present.
2. web/src/api/hooks.ts: `usePublishPlan` / `useArchivePlan` added after `usePlan`; mutationFn passes through to `api.plans.publish` / `api.plans.archive` with `Parameters<typeof ...>[0]` typing; onSuccess invalidates exactly nsPrefix('plans') + nsPrefix('plans-page') + queryKeys.plan(plan.id) (same trio as the comment and the useCancelSubscription pattern). `useClonePlanNext` calls clonePlanNextVersion and invalidates the same trio (plan detail invalidation line commented with the v1-id-is-v3-id rationale). Import of clonePlanNextVersion (+ LegacyPlan if used) merged into the existing '@/api/legacy' import statement.
3. web/src/features/config/plans/plan-detail.tsx: 
   - imports added: useState; useNavigate merged into existing @tanstack/react-router import; toast from 'sonner'; the three new hooks from '@/api/hooks'; handleServerError; ConfirmDialog.
   - Component body: navigate, confirming state union 'publish'|'archive'|'clone'|null, three mutations, statusBusy = OR of the three isPending.
   - Title row buttons with EXACT visibility rules: draft → 发布 button (size sm, no variant spec = default); active → 归档 (size sm variant outline); status !== 'draft' → 克隆新版本 (size sm variant outline); ALL three gated by !statusBusy (hidden, not disabled, while any mutation pending).
   - Three ConfirmDialogs appended at end of the main return branch (after the `if (!plan)` early return — NOT rendered when plan is null/undefined; plan.name/plan.version in dialog descs are therefore safe).
   - Publish dialog: confirmText = actions.publish, cancel common.cancel, mutate({planId: plan.id}), onSuccess toast.success(toast.published) + setConfirming(null), onError handleServerError.
   - Archive dialog: same shape + `destructive` prop.
   - Clone dialog: mutate({planIdOrKey: plan.id}), onSuccess toast with draft.version, setConfirming(null), navigate to '/config/plans/$planId' with draft.id.
4. i18n: config.plans gains actions{publish,archive,cloneNext} + publishConfirm{title,description} + archiveConfirm{...} + cloneConfirm{...} + toast{published,archived,cloned} in BOTH zh-CN.ts and en.ts; descriptions interpolate {{name}} (all three) and cloneConfirm also {{version}}; toast.cloned interpolates {{version}}; key trees structurally identical across the two locales; no other config.plans keys touched.
5. No scope creep: no changes to backend, api spec, v3/v1 SDK artifacts, plans list page, StatusBadge, no draft-exists frontend pre-check (comment explicitly says backend 4xx passthrough via toast — absence is CORRECT per spec), no new e2e specs.
6. Commit: single implementation commit with message exactly `feat(admin): 计划发布/归档/克隆新版本`; worktree clean apart from gitignored .superpowers/sdd/.
7. Deviations from comment code: verify each recorded deviation (plan's 偏差风险 3 条: i18n indentation adapted to actual file; import merging; dialogs inside main branch) is real and benign. SDK typing wins over comment code if they conflict (precedent #2/#4) — check web/node_modules/@openmeter/client/dist/sdk/plans.d.ts publish/archive signatures and dist/models/types.d.ts Plan.status union.
8. Cheap re-checks you must run yourself: `cd web && pnpm build` and `pnpm lint` (record exit codes); `pnpm test:e2e` (2 smoke tests must pass).

Verdict: PASS or FAIL, with findings classified Critical (spec violation blocking) / Important (spec gap in a visible behavior) / Minor (cosmetic/spec-nuance). Evidence (file:line) for every claim. No drive-by fixes — report only. Write your report to /Users/wuyongjun/trea/openmeter-issue-5/.superpowers/sdd/issue-5-plan-lifecycle/spec-review-report.md with the heading `# Spec Review Report — Issue #5 Task 1` and a final line `Ruling: PASS|FAIL`.
