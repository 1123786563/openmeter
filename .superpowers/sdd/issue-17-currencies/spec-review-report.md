# Spec review — issue #17 (controller-executed, 2026-08-29 01:0x+08:00)

Verdict: **PASS** (after fix round 1).

## Issue acceptance criteria vs evidence

1. 「可创建自定义货币（如 CREDIT）并出现在列表」
   ✓ Walkthrough (wire-level): create dialog → POST /api/v3/openmeter/currencies/custom body
   EXACTLY `{name:'Credit Points', code:'CREDIT_POINTS', precision:2 (number),
   decimal_mark:'.', thousand_separator:','}` (snake_case wire via SDK toWire; symbol
   omitted when empty); response → invalidateQueries(nsPrefix('currencies')) → list
   refetch shows new row; toast 自定义货币已创建. Screenshot 05.
2. 「与法币冲突的 code 被前端校验拦截」
   ✓ Plan-comment nuance (verbatim): fiat codes are 3-char ISO so a 4-24 custom code
   can never literally equal one; the conflict check is belt-and-braces on the fiat
   list AND blocks duplicates of existing CUSTOM codes (case-insensitive). Live
   evidence: 'gold_points' vs existing GOLD_POINTS → refine rejects, message rendered
   (与已有货币代码冲突，请更换), ZERO POST fired. Screenshot 04.

## Prescribed plan conformance (issue comment 1, per-file)

- query-keys.ts: 2 keys appended ✓ (anchor "after refunds" drifted — features keys
  landed later in base; appended at object end, same append semantics)
- legacy.ts: FiatCurrency interface + listFiatCurrencies() via apiFetch('/v1/info/currencies') ✓
- hooks.ts: Currencies section before Helpers; useFiatCurrencies (24h staleTime),
  useCurrencies(params) → api.internal.currencies.list with filter/expand/page,
  useCreateCustomCurrency with invalidation ✓ (verified against dist/sdk/internal.d.ts)
- currencies/index.tsx: dual tab, immutable Alert, cost-basis count column (data for
  #18, no action column), skeleton loading, empty states ✓
- custom-currency-dialog.tsx: create-only; dynamic schema vs live lists ✓
- route: placeholder replaced with CurrenciesPage ✓
- locales: full config.currencies.* subtree + common.optional in BOTH locales ✓
- i18n parity 587=587, no zh/en drift, all 29 added keys referenced, all 34 used keys exist ✓
- commit message matches plan (`feat(admin): 货币管理——法币列表与自定义货币创建`) ✓

## Findings

- C-1 (Critical, FIXED in f1ecba377): plan-verbatim FormMessage-children pattern
  renders raw zod sentinel ('conflict') — shadcn FormMessage ignores children when an
  error exists (same defect class as #12 round's C-1, caught here by walkthrough).
  Fix: validation copy moved onto zod checks, schema rebuilt per locale (repo pattern
  from features feature-form-dialog.tsx). Scoped re-review: walkthrough re-run green,
  message text asserted in-DOM.
- M-1 (Minor, accepted): precision '2' default duplicates the placeholder; harmless.
- M-2 (Minor, accepted): dialog fetches custom list without expand for conflict data
  (cache-sharing with the page's expanded query is a separate cache entry; two GETs
  observed in walkthrough evidence — acceptable, #18 may consolidate).

No spec gaps remain. Acceptance criteria fully evidenced.
