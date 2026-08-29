# Quality Review Report — Issue #5 Task 1

- Reviewer role: independent CODE-QUALITY reviewer (engineering quality + repo conventions only; spec compliance handled separately)
- Worktree: `/Users/wuyongjun/trea/openmeter-issue-5`, branch `codex/admin-config-05`
- Diff under review: `git diff c4119ca32..a6ff556ef` (plan-doc commit c4119ca32 skipped; implementation commit a6ff556ef is its direct child)
- Files touched (5, +243/−2): `web/src/api/hooks.ts`, `web/src/api/legacy.ts`, `web/src/features/config/plans/plan-detail.tsx`, `web/src/i18n/locales/en.ts`, `web/src/i18n/locales/zh-CN.ts`
- No tracked files were modified by this review; no branches, pushes, or GitHub writes. Worktree `git status --porcelain` after all runs shows only the untracked `.superpowers/sdd/issue-5-plan-lifecycle/` directory (this report's location).

## Verification runs (executed by reviewer, real exit codes)

| Run (cwd `web/`) | Result | Exit code |
| --- | --- | --- |
| `pnpm build` (`tsr generate && tsc -b && vite build`) | ✓ built in 303ms | **0** |
| `pnpm lint` (`eslint .`) | clean, no output | **0** |
| `pnpm test:e2e` (`playwright test`) | 2 passed (6.2s): sign-in OIDC round-trip, customers smoke | **0** |
| `pnpm exec prettier --check` on the 5 changed files | 4/5 clean; `hooks.ts` flagged | 1 (pre-existing, see §3) |
| Locale key-tree comparison (script below) | IDENTICAL | **0** |

Node v26.7.0 / pnpm 11.7.0.

## 1. TypeScript quality

- No `any`, no `as`-casts, no `@ts-ignore`, no `eslint-disable` in added lines — verified by grepping the `^+` lines of the diff for `: any`, `\bas <Type>`, `@ts-ignore`, `eslint-disable` (zero matches, grep exit 1).
- `Parameters<>` typing of `mutationFn` inputs: `usePublishPlan` (hooks.ts:207) and `useArchivePlan` (hooks.ts:220) use `Parameters<typeof api.plans.publish>[0]` / `[...archive][0]`, mirroring `useCancelSubscription` (hooks.ts:134). `useClonePlanNext` (hooks.ts:233) uses an inline `{ planIdOrKey: string }` — exactly equivalent to the function's parameter type and consistent with the existing multi-field inline-typing precedent in the same file (`useCreateCreditGrant` hooks.ts:458-464, `useInvoiceAction` hooks.ts:291-297). Not a violation.
- `LegacyPlan` field types match the v1 `Plan` schema (api/openapi.yaml:22777+): `version: integer` → `number` ✓ (openapi.yaml:22841-22842), `id/name/key/currency/billingCadence/status/createdAt/updatedAt` all string-backed ✓. `status: string` is a wider superset of the v1 `PlanStatus` enum; the code only consumes `id` and `version` of the clone result, so narrowing is unnecessary.
- No unused imports: every added import (`useState`, `useNavigate`, `toast`, `handleServerError`, `ConfirmDialog`, three mutation hooks, `clonePlanNextVersion` in hooks.ts) is consumed; `tsc -b` and `eslint .` both exit 0.

## 2. React / TanStack patterns

- The three mutation hooks mirror `useCancelSubscription` (hooks.ts:131-145) exactly: `useQueryClient()` in the hook, `Parameters<>`-typed `mutationFn`, `onSuccess` invalidating the domain prefix then the entity key — plus the `plans-page` prefix required by this feature area. All 9 invalidation calls are fire-and-forget with `void` prefix (hooks.ts:210-212, 223-225, 236-239). Style matches; no floating promises.
- `setConfirming(null)` in every `onSuccess` (plan-detail.tsx:284, 309, 336) and the cancel path handled via `onOpenChange={(open) => !open && setConfirming(null)}` on all three dialogs (plan-detail.tsx:270, 294, 319).
- No stale-closure hazards: `plan` comes from `usePlan` query data and all `handleConfirm` closures are re-created per render with current props; mutation callbacks capture the `plan` at click time, which is the correct semantics.
- `statusBusy` correctly ORs the three `isPending` flags (plan-detail.tsx:100-103).
- All three `ConfirmDialog`s render only in the main return branch, after the `if (isLoading)` (105-115) and `if (!plan)` (116-127) early returns — `plan.name` / `plan.id` dereferences inside dialog props (273, 280, 297, 305, 322, 330) can never be null.
- Button visibility matches the intended rules: draft→publish (145-149), active→archive (150-158), non-draft→clone (159-167); all hidden while `statusBusy`.

## 3. Repo conventions and formatting

- Prettier (printWidth 80, single quotes, sort-imports plugin): `legacy.ts`, `plan-detail.tsx`, `en.ts`, `zh-CN.ts` all pass `prettier --check`. `hooks.ts` fails, but diffing against prettier's output shows a single hunk at lines 525-529 (`api.commerce.listRechargeProducts(...)` wrapping) that is **byte-identical in base commit c4119ca32** (base lines 485-488) — the pre-existing recharge-hunk violation documented in the plan's non-goals, explicitly out of scope. Added lines in `hooks.ts` are prettier-clean.
- Import order is enforced by `@trivago/prettier-plugin-sort-imports` and passes on all changed files.
- No hardcoded UI copy in added lines: every user-visible string goes through `t()`; `v{plan.version}` (plan-detail.tsx:144) is a value prefix, explicitly allowed by the brief.
- Toasts via `sonner` `toast.success` + `onError: handleServerError` — identical shape to the sibling precedent `web/src/features/config/features/index.tsx:162-187` (deleteConfirm).
- i18n trees structurally identical (see §6 script + output); all 16 added keys are consumed in plan-detail.tsx (no dead keys); `common.cancel` exists in both locales.

## 4. Query hygiene

`api.plans.*` is consumed only through hooks.ts (grep-verified across `web/src`):

- `usePlans` → `queryKeys.plans()` = `['api', ns, 'plans']` (subscription wizard `create-subscription-dialog.tsx:41`, subscriptions list `subscriptions/index.tsx:59`, subscription detail `subscription-detail.tsx:137`) — covered by `nsPrefix('plans')`.
- `usePlansPage` → `queryKeys.plansPage(params)` = `['api', ns, 'plans-page', params]` (config plans list `config/plans/index.tsx:29`) — covered by the `nsPrefix('plans-page')` prefix match for every page/status filter combination.
- `usePlan` → `queryKeys.plan(id)` (plan detail) — publish/archive return the plan with the **same id**, so the detail invalidation is correct; clone returns a **new id** and immediately navigates to it, mounting a fresh query, so no stale cache can be shown. The clone hook's `queryKeys.plan(plan.id)` invalidation of the not-yet-fetched new id is harmless (matches the hook template).

No other plans-keyed or page-keyed query exists; nothing that renders plan data is left stale after publish/archive/clone. The invalidation trio is sufficient.

## 5. UX / edge cases

- Double-click protection is complete on both layers: `ConfirmDialog` disables confirm **and** cancel while `isLoading` (confirm-dialog.tsx:56, 64), and the header action buttons are hidden while `statusBusy` (plan-detail.tsx:145/150/159).
- Error path: `onError: handleServerError` shows the server message as a toast and `confirming` is intentionally not reset, so the dialog stays open for retry. This is byte-for-byte the sibling precedent's behavior (`features/index.tsx:183` — `deleteTarget` not reset on error). Per the brief, sibling-consistent behavior classifies as Minor at most; I record it as consistent-by-precedent, not a defect.
- Clone-success ordering (`setConfirming(null)` then `void navigate(...)`, plan-detail.tsx:336-340) is correct: dialog closes before the route change mounts the new detail's loading skeleton.

## 6. Locale key-tree comparison (script + output, run by reviewer)

Script (run as `node --input-type=module -e` from `web/`; Node 26 native TS type-stripping imports the `as const` locale modules directly):

```js
import en from 'file:///Users/wuyongjun/trea/openmeter-issue-5/web/src/i18n/locales/en.ts'
import zh from 'file:///Users/wuyongjun/trea/openmeter-issue-5/web/src/i18n/locales/zh-CN.ts'

function flatten(obj, prefix = '', out = new Map()) {
  for (const [k, v] of Object.entries(obj)) {
    const path = prefix ? prefix + '.' + k : k
    if (v && typeof v === 'object') flatten(v, path, out)
    else out.set(path, String(v))
  }
  return out
}

const enAll = flatten(en)
const zhAll = flatten(zh)
const onlyEn = [...enAll.keys()].filter((k) => !zhAll.has(k))
const onlyZh = [...zhAll.keys()].filter((k) => !enAll.has(k))

const enPlans = new Map([...enAll].filter(([k]) => k === 'config.plans' || k.startsWith('config.plans.')))
const zhPlans = new Map([...zhAll].filter(([k]) => k === 'config.plans' || k.startsWith('config.plans.')))
const onlyEnPlans = [...enPlans.keys()].filter((k) => !zhPlans.has(k))
const onlyZhPlans = [...zhPlans.keys()].filter((k) => !enPlans.has(k))

console.log('whole-file key counts: en=' + enAll.size + ' zh=' + zhAll.size)
console.log('keys only in en (whole file): ' + JSON.stringify(onlyEn))
console.log('keys only in zh (whole file): ' + JSON.stringify(onlyZh))
console.log('config.plans subtree key counts: en=' + enPlans.size + ' zh=' + zhPlans.size)
console.log('keys only in en (config.plans): ' + JSON.stringify(onlyEnPlans))
console.log('keys only in zh (config.plans): ' + JSON.stringify(onlyZhPlans))

const ph = (s) => [...String(s).matchAll(/\{\{(\w+)\}\}/g)].map((m) => m[1]).sort().join(',')
const mismatch = [...enPlans.keys()].filter((k) => zhPlans.has(k) && ph(enPlans.get(k)) !== ph(zhPlans.get(k)))
console.log('config.plans interpolation mismatches: ' + JSON.stringify(mismatch.map((k) => [k, ph(enPlans.get(k)), ph(zhPlans.get(k))])))

const ok = onlyEn.length === 0 && onlyZh.length === 0 && onlyEnPlans.length === 0 && onlyZhPlans.length === 0 && mismatch.length === 0
console.log('RESULT: ' + (ok ? 'IDENTICAL' : 'MISMATCH'))
process.exit(ok ? 0 : 1)
```

Output:

```
whole-file key counts: en=528 zh=528
keys only in en (whole file): []
keys only in zh (whole file): []
config.plans subtree key counts: en=40 zh=40
keys only in en (config.plans): []
keys only in zh (config.plans): []
config.plans interpolation mismatches: []
RESULT: IDENTICAL
LOCALE_SCRIPT_EXIT:0
```

## 7. Regressions

- Diff touches exactly the 5 expected files; no route files, no generated SDK/spec changes, no backend changes.
- Existing smoke e2e unaffected: 2 passed, exit 0 (§ table).
- `pnpm build` and `pnpm lint` both exit 0.

## Findings

| # | Severity | Location | Description |
| --- | --- | --- | --- |
| 1 | Minor | web/src/api/hooks.ts:238 | Code comment written in Chinese (`// v1 id 是 v3 同一资源 id；…`) inside a file whose comments are otherwise uniformly English (hooks.ts, legacy.ts, query-keys.ts all English). Pure consistency nit; no functional impact. |

Non-finding observations (verified acceptable, no action needed):

- `useClonePlanNext` inline `{ planIdOrKey: string }` typing instead of `Parameters<>` — exact-equivalent and precedented by `useCreateCreditGrant` / `useInvoiceAction` in the same file.
- `LegacyPlan.status: string` wider than the v1 enum — code only consumes `id`/`version`; superset is harmless.
- `hooks.ts` prettier violation — pre-existing recharge hunk, byte-identical in base commit, explicitly out of scope per plan non-goals.

No Critical findings. No Important findings. One Minor finding. All verification runs green with recorded exit codes.

Ruling: PASS
