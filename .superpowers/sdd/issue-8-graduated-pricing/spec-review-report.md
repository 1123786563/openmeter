# Spec review — issue #8 (controller-executed programmatic, 2026-08-29 01:4x+08:00)

Method: prescription (issue comment 1) vs branch diff 5a4666ec7..fceb8392e,
rule-by-rule; schema rules verified by direct programmatic execution
(node --experimental-strip-types, /tmp/i8-schema-check.mts + range recheck);
UI/payload verified by temporary Playwright walkthrough (wire-level, deleted
after run; logs summarized in progress.md).

## Acceptance criteria (issue body)

- 可创建含阶梯价的计划并发布 → walkthrough: two-tier graduated plan POSTed,
  payload exact: tiers[0] {up_to_amount:'99', unit_price:{type:'unit',amount:'0.10'},
  flat_price:{type:'flat',amount:'10'}}, tiers[1] open-ended {unit_price only};
  toast 计划已创建（草稿）. Publish flow unchanged from #5 (not regressed; badge
  rendering for graduated/volume pre-existing from #4, plan-detail L74-78). PASS.
- 阶梯区间错误（重叠/缺口）在对应行报错 → walkthrough: row2 firstUnit=200 →
  「与上一档不连续（应为上档截止量 + 1）」 rendered at that row, ZERO POST.
  Schema-level: tierGap fires for both gap (99→200) and overlap (150→100) at
  next row's firstUnit path. PASS.

## Prescription contract items

1a NON_NEGATIVE_INT ✓ | 1b tierSchema+TierFormValues+priceFormSchema tiered
variant (mode enum, tiers min1) ✓ | 1c defaultTier ✓ | 1d superRefine: usage_based
unit|tiered ✓, tier rules with exact paths ✓ (14/14 programmatic checks PASS:
tierFirstFromZero, tierLastOpen, tierLastRequired, tierRange [re-verified with
correct trigger case], tierGap, tierOverlap, valid-parse, bound-rejects −1/1.5,
defaultTier shape, flat_fee+tiered, oneTime+tiered, usage+flat) | 1e toPriceInput
tiered → {type:mode, tiers map upToAmount/unitPrice/flatPrice-omitempty} ✓
(wire-proven; firstUnit not in payload ✓).
2 price-editor.tsx: tiered option in kind select ✓; resetPrice tiered branch ✓;
mode select ✓; TierEditor rows (firstUnit locked row0, lastUnit locked last row,
addTier continuation, remove ≥1 row) ✓ (walkthrough drove add-tier + both locks).
3 i18n: all 17 new keys present zh+en (programmatic keycheck), prescription text
verbatim ✓.

## Deviations from prescription (with rationale)

- D-1 (typing, mechanical): prescription's `useWatch({name: \`${pricePath}.kind\`
  as never})` breaks TS inference (converts whole-form type); kept current
  file's typed template-literal path. Semantics identical. ACCEPTED.
- D-2 (message wording): `usagePriceKind` copy updated 「只能选择单价」→「只能选择
  单价或阶梯价」 (en likewise) — key unchanged; old text became false after this
  change (doc-convention: do not leave comments/copy made misleading by the
  change). ACCEPTED.

Verdict: PASS — no Critical/Important findings; 0 fix rounds needed.
