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

- T2 编辑模式接线 (commits 477413166 + 9f71ad1bb): COMPLETED.
  - ANOMALY RECORD: commit 477413166 was created by an EXTERNAL auto-committer (user-side tool; reflog shows plain commit, no force) one minute before the controller's own commit point — it captured the controller's exact T2 working-tree changes plus the then-untracked ledger (worktrees lacked main's untracked .superpowers/sdd/.gitignore). Content verified identical to intent; accepted. Countermeasure applied: .gitignore shields ('*') added to both track worktrees' .superpowers/sdd/, ledgers untracked again (commit b5defd24c).
  - Implementation: useUpdatePlan (mirrors useCreatePlan invalidations plans/plans-page/plan(id)); wizard edit mode (plan?: Plan prop, isEdit, reset(fromPlanToWizardValues), PUT branch {planId, body: toUpsertPlanRequest}, disabled key/currency/cadence + immutableHint, editTitle/editSubmit/editSubmit pending); plan-detail draft-only edit entry + wizard mount; zh/en i18n keys (editTitle/editSubmit/immutableHint/toast.updated).
  - DEVIATION from prescription 4b (ruled, evidence on record): create branch passes body directly (api.plans.create request shape = CreatePlanRequestInput itself, TS2353 with {body} wrapper; original #6 code correct). Update branch {planId, body} confirmed correct vs SDK UpdatePlanRequest + UpdatePlanResponse=Plan unwrap.
  - Gates: pnpm build exit 0, pnpm lint exit 0, e2e sign-in PASS + customers smoke FAIL signature-identical to pristine base run at same time window (base log /tmp/issue-round/base-e2e.log; environmental :8888 shim ruling re-confirmed, non-regression).
  - Spec review (controller, fresh runs): locale real-module parity zh=790=en=790 zero drift; all 4 new keys + common.edit present both locales; draft-only entry condition verified; SDK contract first-hand verified (updatePlan req {planId, body}, response unwrap Plan).
  - Quality review (controller, fresh runs): anti-patterns 0 on added lines (1 eslint-disable = pre-existing moved comment, preserved per AGENTS.md); prettier hunk-count method: wizard/hooks/zh/en unchanged vs base (pre-existing dirty), plan-detail reduced 1→0 via review fix commit 9f71ad1bb (import order + Button wrap).
- FINAL REVIEW: PENDING

## 最终全分支复审 (controller, 2026-08-29, base 49d1a760b)
- 角度1 规格追溯: 处方 T1/T2 全部块落地; 记录在案偏差 2 项（create 分支直传 body——SDK CreatePlanRequest 无包装、TS2353 实证; prettier 强制格式化项），均有证据。
- 角度2 回归: 向导创建模式语义不变（EMPTY_PLAN 回退、disabled 全部 isEdit 门控、plans/index.tsx 无 plan 挂载点零改动兼容）。
- 角度3 证据审计: 全部门禁日志落盘 /tmp/issue-round/（t9-t2-build3/t9-t2-lint2/t9-t2-e2e + base-e2e 同签名）。
- 角度4 约定/反模式: 全分支新增行 0 反模式、0 新增 eslint-disable、locale 790=790。
- 角度5 对抗边界: 空 phases 由服务端约束+向导校验兜底; graduated firstUnit=-1→0 语义经 roundtrip 6/6 覆盖。
- RULING: PASS（可交付，等待外部化批准）。
- 剩余风险: e2e customers 冒烟环境性失败（基线同签名）; 主题内 isEdit 不可变字段禁用但服务端仍为权威校验; 477413166 外部自动提交者事件（已记录+加固）。
