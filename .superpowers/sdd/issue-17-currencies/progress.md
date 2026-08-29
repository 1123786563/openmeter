# SDD ledger — issue #17 货币：法币与自定义列表/创建

- Branch codex/admin-config-17 @ base f6e767dc3; worktree /Users/wuyongjun/trea/openmeter-issue-17.
- Plan: docs/superpowers/plans/issue-17-currencies.md (commit 1e043ba2f); prescriptive source = issue #17 comment 1 (verbatim per-file code).
- Round: 2026-08-29 00:26+08:00 claim (see main-checkout .superpowers/sdd/issue-17-19-claim.md).
- SDD mode: superpowers:subagent-driven-development skill unavailable in session catalog; subagent + subagent_fork probes DENIED by unattended allowlist (same as #6/#7/#12 rounds) → DOWNGRADE: controller-executed implementation + controller-run programmatic reviews with independent verification runs (fresh commands, logs on disk). Attended spot-check remains OPEN.

## Pre-implementation anchor verification (controller, 2026-08-29 00:3x+08:00)

All checked first-hand at base f6e767dc3 (worktree issue-17):
- SDK `api.internal.currencies.{list,createCustomCurrency}` exist (dist/sdk/internal.d.ts InternalCurrencies) ✓
- `ListCurrenciesQuery { page{size,number}, filter{type:'fiat'|'custom'}, expand?: 'cost_basis'[] }` ✓ (operations/currencies.d.ts)
- `CreateCurrencyCustomRequest { name, symbol?, precision:number, decimalMark, thousandSeparator, code }` ✓; `CurrencyCustom { ..., id, code, createdAt:Date, costBasis?:CostBasis[] }` ✓ (types.d.ts)
- hooks.ts: `/* Helpers */` section at EOF with local `nsPrefix` ✓; imports from `@/api/legacy` ✓
- legacy.ts: `apiFetch` from `@/lib/api` ✓ (append-at-EOF pattern matches clonePlanNextVersion)
- locales: common has cancel/confirm/edit/submitting; `common.optional` ABSENT → must add both locales (plan note anticipated this) ✓
- `config.currencies` currently = {title, description} from #1 (zh line 273) ✓; `formatShortDateTime` exists (lib/format.ts:42) ✓
- UI: alert/badge/skeleton/table/tabs/switch/label/textarea/select/input/form/dialog/button all present ✓; currencies route is #1 placeholder (PlaceholderPage) ✓
- `pnpm install --prefer-offline` done (4.6s).

## Task 1 — implementation

- Status: in progress.
- Status: completed (commit ccfdacc52, 8 files +679/−9, per plan verbatim).

## Acceptance trio (initial, feat HEAD ccfdacc52)

- build ✓ / lint ✓ / e2e 2 smoke ✓ (walkthrough infra verified separately)

## Browser walkthrough (temporary spec, mock-backed, wire-level)

- Flow: fiat tab rows (USD row: $, 2 subunits) → custom tab (immutable Alert, GOLD_POINTS
  row cost-basis count 2) → create dialog → code 'gold_points' CONFLICT BLOCKED (refine
  vs live lists; message rendered; ZERO POST) → valid 'CREDIT_POINTS' submit → POST body
  EXACT `{name:'Credit Points',code:'CREDIT_POINTS',precision:2,decimal_mark:'.',
  thousand_separator:','}` → toast 自定义货币已创建 → new row appears. 5 screenshots
  /tmp/issue17-shots (pixel-verified non-blank, distinct states).
- Found C-1 (Critical): shadcn FormMessage renders raw zod sentinel & ignores children —
  same defect class as #12 round. Fix round 1 (commit f1ecba377): validation i18n copy
  moved onto zod checks; buildSchema(conflictingCodes, t) rebuilt per locale; plain
  <FormMessage /> everywhere. Walkthrough re-run: green incl. in-DOM translated message
  + zero-POST on conflict. Walkthrough spec DELETED after runs (not in commits).

## Reviews (controller-executed programmatic; reports in this dir)

- Spec review: PASS after fix round 1 (spec-review-report.md; acceptance items evidenced
  wire-level; C-1 fixed; M-1/M-2 minors accepted with rationale).
- Quality review: PASS (quality-review-report.md; trio re-run at final HEAD f1ecba377:
  build 263ms / eslint 0 / e2e 2 passed; locale parity zh 587 = en 587, zero drift, all
  29 added keys referenced, 34 used keys resolve; anti-pattern scan clean; scope == plan).
- Final whole-branch review: PASS (final-review-report.md; full diff re-read; 3 commits;
  no open findings; fix rounds used 1/5).

## Track status: LOCALLY COMPLETE — awaiting user approval to externalize
(push codex/admin-config-17, merge in order #7 → #12 → #17 → #19, close issue #17).
- Review round 2026-08-29 04:33 +0800: drift check PASS (tip/commits/tree unchanged since green runs; only untracked SDD artifacts). No externalization this round — ask_user_question blocked by unattended allowlist, approval still awaited. NEW: merge-chain forecast done (claim file issue-2026-08-29-review-round-claim.md): this track merges with union-resolvable conflicts [+DEDUPE for #21/#25 useApps·apps keys]; acceptance trio required after each merge.

## 外化记录（2026-08-29 11:07–11:5x +0800，外化轮运行锁 acquiredAt=2026-08-29T03:07:28Z）

- 批准依据：用户在对话中回复「1」＝批准外发（对应前轮汇报等待项 1「外发批准」；台账既定批准消费协议）。
- 外化链按 Issue 编号序 #8→#13→#17→#19→#21→#23→#25→#27 执行，本轨道为其中一环。
- 分支 codex/admin-config-17 @ f1ecba377 推送至 origin → merge --no-ff 进 main（merge commit e7bb6834d）→ push main → Issue #17 以证据评论关闭（已验证 CLOSED）。
- 合并冲突按 claim.md 预报并集解决（本轨道详情见外化轮 claim：issue-2026-08-29-externalization-round-claim.md）；合并后门禁 build 0 / lint 0 / e2e 与 pristine base 同基准（:8888 用户 dev-shim 环境签名一致，非回归）。
- 全链完成后 main @ 49d1a760b：locale 权威奇偶校验（真实模块求值）en=786 zh=786 零漂移。
- STATUS: EXTERNALIZED & CLOSED。轨道终态。

## 独立抽查记录（2026-08-29 11:33–12:1x +0800，抽查轮运行锁 acquiredAt=2026-08-29T03:33:51Z）

- 本会话 subagent 能力首次恢复（此前 15+ 轮被无人值守允许列表禁用）；以全新上下文独立 reviewer subagent 对全部 10 条降级轨道并行抽查。
- VERDICT: PASS（1 Low 处方层：symbol max(16) 无 FormMessage）。RHF 7.72.1 resolver 重读语义验证 C-1 修复正确；合并瞬时重复 banner 注释已被 #19 合并自愈，HEAD 干净。
- 汇总与更正评论见 issue-2026-08-29-spotcheck-round-claim.md；跨切面门禁（build 0/lint 0/locale 786=786 真实求值/反模式 0/重复键 0）全过。
