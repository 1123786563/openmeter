# SDD ledger — issue #19 税码：CRUD 与 app 映射

- Branch codex/admin-config-19 @ base f6e767dc3; worktree /Users/wuyongjun/trea/openmeter-issue-19.
- Plan: docs/superpowers/plans/issue-19-tax-codes.md (commit 0a2d2c253); prescriptive source = issue #19 comment 1 (verbatim per-file code).
- Round: 2026-08-29 00:26+08:00 claim (see main-checkout .superpowers/sdd/issue-17-19-claim.md).
- SDD mode: superpowers:subagent-driven-development skill unavailable in session catalog; subagent + subagent_fork probes DENIED by unattended allowlist (same as #6/#7/#12 rounds) → DOWNGRADE: controller-executed implementation + controller-run programmatic reviews with independent verification runs (fresh commands, logs on disk). Attended spot-check remains OPEN.

## Pre-implementation anchor verification (controller, 2026-08-29 00:3x+08:00)

All checked first-hand at base f6e767dc3:
- SDK `api.tax.{listCodes,createCode,upsertCode,deleteCode}` exist (dist/sdk/tax.d.ts) ✓
- `ListTaxCodesQuery { page{size,number}, includeDeleted?:boolean }` ✓; `UpsertTaxCodeRequest = { taxCodeId, body }` with body = { name, description?, labels?, appMappings } — NO key ✓; `CreateTaxCodeRequest` HAS key + appMappings ✓; `TaxCode { id,name,description?,labels?,createdAt,updatedAt,deletedAt?,key,appMappings }` ✓; `TaxCodeAppMapping { appType:'sandbox'|'stripe'|'external_invoicing', taxCode:string }` ✓ (types.d.ts)
- ConfirmDialog props incl. isLoading/destructive/confirmText/cancelBtnText/handleConfirm ✓
- PageHeader has actions slot ✓; Switch/Label/Textarea/Select present ✓
- locales: common.edit/cancel/confirm/submitting exist ✓; `config.taxCodes` currently = #1's {title, description} only ✓; common.optional absent at base — #17 (parallel track) adds it; this track adds it too if absent at implementation time (append-only, trivial merge).
- hooks.ts: Features section ends before `/* Helpers */` ✓; tax-codes route is #1 placeholder ✓
- `pnpm install --prefer-offline` done (5.5s).

## Task 1 — implementation

- Status: in progress.
- Status: completed (commit 50701da53, 7 files +782/−11, per plan verbatim).

## Acceptance trio (initial, feat HEAD 50701da53)

- build ✓ / lint ✓ (after escaping fix) / e2e 2 smoke ✓

## Browser walkthrough (temporary spec, mock-backed, wire-level)

- 6-step flow all green: list + mapping badges (Sandbox/Stripe) → create with 2 mappings
  (POST body exact per issue example incl. app_mappings snake_case) → duplicate appType
  blocked client-side (message rendered, zero POST) → edit (key disabled + hint; PUT body
  WITHOUT key, asserted via JSON contains '"key"' false) → delete ConfirmDialog → 204 →
  include_deleted switch flips wire param. 6 screenshots /tmp/issue19-shots
  (pixel-verified). Debug probes (debug-19 / debug-switch specs) used for the two
  findings below, then deleted; walkthrough spec deleted after green run.
- Found C-1 (Critical): FormMessage-children defect (same as #17 C-1). Fixed in 1761e9e2f.
- Found C-2 (Critical): array-level refine error lands at formState.errors.appMappings
  .root.message (RHF nests array-container errors under root) — plan-verbatim condition
  read .message → rendered nothing. Fixed in 1761e9e2f; verified by re-run.
- Engineering note: SDK query params serialize camelCase→snake_case on the wire
  (include_deleted=true) — relevant to future hooks tasks' mocks/assertions.

## Reviews (controller-executed programmatic; reports in this dir)

- Spec review: PASS after fix round 1 (spec-review-report.md; all 3 acceptance groups
  evidenced wire-level; C-1/C-2 fixed; M-1/M-2 minors accepted).
- Quality review: PASS (quality-review-report.md; trio at final HEAD 1761e9e2f: build
  248ms / eslint 0 / e2e 2 passed; locale parity zh 589 = en 589; 31 added keys all
  referenced (3 via template literals); anti-pattern scan clean; scope == plan).
- Final whole-branch review: PASS (final-review-report.md; 3 commits; no open findings;
  fix rounds used 1/5; array-root + wire-name notes recorded for #15/#20).

## Track status: LOCALLY COMPLETE — awaiting user approval to externalize
(push codex/admin-config-19, merge in order #7 → #12 → #17 → #19, close issue #19).
- Review round 2026-08-29 04:33 +0800: drift check PASS (tip/commits/tree unchanged since green runs; only untracked SDD artifacts). No externalization this round — ask_user_question blocked by unattended allowlist, approval still awaited. NEW: merge-chain forecast done (claim file issue-2026-08-29-review-round-claim.md): this track merges with union-resolvable conflicts [+DEDUPE for #21/#25 useApps·apps keys]; acceptance trio required after each merge.

## 外化记录（2026-08-29 11:07–11:5x +0800，外化轮运行锁 acquiredAt=2026-08-29T03:07:28Z）

- 批准依据：用户在对话中回复「1」＝批准外发（对应前轮汇报等待项 1「外发批准」；台账既定批准消费协议）。
- 外化链按 Issue 编号序 #8→#13→#17→#19→#21→#23→#25→#27 执行，本轨道为其中一环。
- 分支 codex/admin-config-19 @ 1761e9e2f 推送至 origin → merge --no-ff 进 main（merge commit f47e3fc74）→ push main → Issue #19 以证据评论关闭（已验证 CLOSED）。
- 合并冲突按 claim.md 预报并集解决（本轨道详情见外化轮 claim：issue-2026-08-29-externalization-round-claim.md）；合并后门禁 build 0 / lint 0 / e2e 与 pristine base 同基准（:8888 用户 dev-shim 环境签名一致，非回归）。
- 全链完成后 main @ 49d1a760b：locale 权威奇偶校验（真实模块求值）en=786 zh=786 零漂移。
- STATUS: EXTERNALIZED & CLOSED。轨道终态。

## 独立抽查记录（2026-08-29 11:33–12:1x +0800，抽查轮运行锁 acquiredAt=2026-08-29T03:33:51Z）

- 本会话 subagent 能力首次恢复（此前 15+ 轮被无人值守允许列表禁用）；以全新上下文独立 reviewer subagent 对全部 10 条降级轨道并行抽查。
- VERDICT: PASS（2 Low 均处方层：key 无 ≤64 长度上限〔后端 400 兜底〕；description max(1024) 无 FormMessage）。C-2 数组级错误 root 迁移经 @hookform/resolvers 5.9.1 源码级证实。
- 汇总与更正评论见 issue-2026-08-29-spotcheck-round-claim.md；跨切面门禁（build 0/lint 0/locale 786=786 真实求值/反模式 0/重复键 0）全过。
