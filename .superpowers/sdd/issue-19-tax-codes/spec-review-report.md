# Spec review — issue #19 (controller-executed, 2026-08-29 01:0x+08:00)

Verdict: **PASS** (after fix round 1).

## Issue acceptance criteria vs evidence

1. 「税码可创建、编辑（key 创建后不可改）、删除」
   ✓ Create: POST /api/v3/openmeter/tax-codes body EXACTLY
   `{name:'数字服务', key:'digital_services', app_mappings:[{app_type:'sandbox',
   tax_code:'txcd_00000000'},{app_type:'stripe', tax_code:'txcd_10000000'}]}` — the
   issue's own acceptance example, matched key-for-key on the wire. Row appears in
   list; toast 税码已创建.
   ✓ Edit: key input disabled + keyImmutable hint rendered; PUT body captured WITHOUT
   key field (`{name:'标准税率（更新）', app_mappings:[...]}` — JSON.stringify includes
   '"key"' asserted false) — upsert immutability proven at wire level; toast 税码已更新.
   ✓ Delete: ConfirmDialog (title 删除税码, name interpolated 数字服务) → DELETE → 204
   (mock) → row removed; toast 税码已删除.
2. 「app 映射行动态增删」
   ✓ useFieldArray rows: 添加映射 adds, 移除该行 (ghost icon button) removes; Select
   per-row app_type (3 enum options); duplicate appType client-blocked: two sandbox
   rows → 每种应用类型只能有一条映射 rendered, createBodies.length stayed 1 (no POST).
3. 「列表（include_deleted 开关）」
   ✓ Switch 显示已删除 toggles → GET wire param flips include_deleted=false→true
   (SDK serializes camelCase param to snake_case wire name — verified live).

## Prescribed plan conformance (issue comment 1, per-file)

- query-keys.ts: taxCodes key ✓ (anchor "after Currencies section" is advisory
  ordering from the issue author assuming #17 first; parallel track appends before
  Helpers — final ordering emerges at merge, no code dependency).
- hooks.ts: 4 hooks exactly as prescribed (listCodes page size 100; 3 mutations
  invalidate nsPrefix('tax-codes')) ✓ verified against dist/sdk/tax.d.ts.
- tax-code-form-dialog.tsx: create/edit shared, key immutable on edit, zod
  name(1-256)/key ResourceKey regex (identical to #2's FEATURE_KEY)/description
  (≤1024)/mapping rows + duplicate-appType refine ✓.
- tax-codes/index.tsx: list + badges + include_deleted switch + disabled actions on
  deleted rows + destructive badge + ConfirmDialog ✓.
- route: placeholder replaced ✓; locales full config.taxCodes.* + common.optional ✓.
- i18n parity 589=589; 31 added keys all referenced (3 via template literal
  t(`config.taxCodes.appType.${appType}`)); 34 used keys all exist ✓.
- commit message matches plan (`feat(admin): 税码管理（CRUD 与 app 映射）`) ✓.

## Findings

- C-1 (Critical, FIXED in 1761e9e2f): same FormMessage-children defect as #17's C-1
  — raw sentinels ('invalid'/'duplicateAppType') would leak to UI. Fix identical:
  i18n copy onto zod checks, schemas rebuilt per locale via buildSchemas(t).
- C-2 (Critical, FIXED in 1761e9e2f): array-level refine error renders via
  formState.errors.appMappings.ROOT.message (RHF nests array-container errors under
  root), not .message — plan-verbatim condition rendered nothing (walkthrough
  evidence: duplicate error invisible pre-fix). Discovered by debug probe dumping
  the live error object; fix verified by re-run (message visible, still no POST).
- M-1 (Minor, accepted): description max(1024) has no FormMessage (plan-verbatim);
  zod default message unreachable in UI — same as plan.
- M-2 (Minor, accepted): deleted-row edit/delete buttons disabled but visible
  (issue prescribed "操作禁用", matches).

No spec gaps remain. Acceptance criteria fully evidenced.
