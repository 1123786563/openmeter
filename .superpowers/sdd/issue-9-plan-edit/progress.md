# SDD ledger — issue #9 计划编辑（draft 回填与更新）

- Branch codex/admin-config-09 @ base 49d1a760b; worktree /Users/wuyongjun/trea/openmeter-issue-9.
- Plan: docs/superpowers/plans/issue-9-plan-edit.md; prescriptive source = issue #9 comment 1 (verbatim per-file code, saved /tmp/issue-round/issue-9-comments.md).
- Round: 2026-08-29 11:57+08:00 claim (see main-checkout .superpowers/sdd/ round claim file).
- Context compaction at round start: controller session started fresh (this is round-start context; no prior conversation carried over).
- SDD mode: superpowers:subagent-driven-development skill not present in session catalog → controller orchestrates equivalent discipline: per-task fresh implementer subagent + independent spec-compliance review + code-quality review subagents (≤5 fix rounds), final whole-branch adversarial review. subagent tool available this session (verified by spotcheck round precedent 2026-08-29 11:33).

## Pre-implementation anchor verification (controller, 2026-08-29 12:0x+08:00)

All checked first-hand at base 49d1a760b (main checkout):
- `plan-form-schema.ts` exports verified: tierSchema/priceFormSchema(4 variants)/rateCardSchema/planWizardSchema/EMPTY_PLAN/toPlanPhases/toCreatePlanRequest ✓ — prescription's `toUpsertPlanRequest` reuse of `toPlanPhases` is wired, no `toPriceInput` export needed for #9 scope.
- `plan-form-wizard.tsx`: `PlanFormWizardProps = { open, onOpenChange }` (line 59), single `useCreatePlan` path — edit-mode diff applies cleanly.
- `hooks.ts`: `useCreatePlan` (L211), `useClonePlanNext` (L250); `useUpdatePlan` absent ✓ (to add).
- `query-keys.ts`: `plan: (id) => ns('plan', id)` (L26) ✓ (invalidation targets exist).
- `plan-detail.tsx` + #5 action area present (publish/archive/clone buttons) ✓.
- web deps installed via `pnpm install --prefer-offline` (8.4s shared store).

## Tasks

- T1 schema 映射: PENDING
- T2 编辑模式接线: PENDING

## Review rounds

(none yet)

## Final whole-branch review

(pending)

## SDD mode downgrade record (2026-08-29 12:1x+08:00)
- subagent probed first-hand this round: DENIED by unattended allowlist (subagent + subagent_fork + workflow + list_experts all rejected — 4 probe errors on record in controller transcript).
- Standing DOWNGRADE applied (#6/#7/#12/#17/#19/#8/#13/#21/#23/#25/#27 precedent, independently spot-checked 2026-08-29 11:33 with 10/10 PASS):
  controller-executed implementation per prescription + controller-run SEPARATE spec-compliance and code-quality review passes with fresh verification commands, logs on disk; ≤5 fix rounds; final whole-branch adversarial review multi-angle. Attended spot-check of this track remains OPEN for a future session.

- T1 schema 映射 (commit 77be4ef48, amended): COMPLETED — prescription §1/§2 verbatim (fromRateCardToForm/fromPriceToForm/fromPlanToWizardValues/toUpsertPlanRequest + import merge).
  - Gate: pnpm build exit 0 (log /tmp/issue-round/t9-t1-build3.log).
  - Spec review (controller, fresh runs): round-trip property test 6/6 PASS at Node-26 strip-types against real module (/tmp/issue-round/t9-roundtrip-test.ts): schema-valid backfill, cadence convergence (P1D→P1M display, card P1W→null), tier firstUnit 0/101 reconstruction, PUT body omits immutable fields, wire-level phases preservation, blank description→undefined.
  - Quality review (controller, fresh runs): eslint exit 0; anti-pattern grep on added lines = 0; prettier — base file already dirty at line 309 (inherited from #8, repo format:check red pre-existing per ledger); MY added lines made prettier-clean (2 print-width wrappings applied, ruled as formatting-only deviation from verbatim prescription).
- T2 编辑模式接线: IN PROGRESS
