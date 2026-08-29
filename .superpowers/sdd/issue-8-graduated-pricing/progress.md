# SDD ledger — issue #8 计划创建向导：阶梯价（graduated/volume）

- Branch codex/admin-config-08 @ base 5a4666ec7; worktree /Users/wuyongjun/trea/openmeter-issue-8.
- Plan: docs/superpowers/plans/issue-8-graduated-pricing.md; prescriptive source = issue #8 comment 1 (verbatim per-file code, saved /tmp/issue-8-13/issue-8-plan.md).
- Round: 2026-08-29 01:27+08:00 claim (see main-checkout .superpowers/sdd/issue-8-13-claim.md).
- Context compaction at round start: controller session started fresh (this is round-start
  context; no prior conversation carried over).
- SDD mode: superpowers:subagent-driven-development skill unavailable in session catalog;
  subagent probed first-hand this round (01:2x) — DENIED again by unattended allowlist →
  standing DOWNGRADE (#6/#7/#12/#17/#19 precedent): controller-executed implementation +
  controller-run programmatic reviews with independent verification runs (fresh commands,
  logs on disk). Attended spot-check remains OPEN.

## Pre-implementation anchor verification (controller, 2026-08-29 01:3x+08:00)

All checked first-hand at base 5a4666ec7 (worktree issue-8):
- JS SDK `api/spec/packages/aip-client-javascript/src/models/types.ts`:
  `Price = PriceFree|PriceFlat|PriceUnit|PriceGraduated|PriceVolume` ✓;
  `PriceGraduated/PriceVolume { type:'graduated'|'volume', tiers: PriceTier[] }` ✓;
  `PriceTier { upToAmount?: string, flatPrice?: PriceFlat, unitPrice?: PriceUnit }` ✓ —
  prescriptive mapping (upToAmount/unitPrice/flatPrice, firstUnit NOT in payload) matches
  field-for-field.
- Go SDK `api/v3/client/models_shared.go` L430-470 isomorphic; doc comment "At least one
  price component (flat_price or unit_price) must be set" → per-tier unitPrice always,
  flatPrice omitted when flatAmount==='' ✓.
- Current `plan-form-schema.ts` (239 lines): priceFormSchema free|flat|unit
  discriminatedUnion; rateCardSchema superRefine (flatFeePriceKind/featureRequired/
  usagePriceKind/oneTimeFlatOnly); toPriceInput switch 3 cases; defaultRateCard; V-style
  i18n-key messages ✓ — exactly the "#7 provided" contract the comment builds on.
- Current `price-editor.tsx` (109 lines): FieldError export, resetPrice (free/unit/flat),
  PriceEditor props {control,phaseIndex,cardIndex,currency}, kind watch ✓.
- Locales: `config.plans.wizard.{fields,errors,toast,cadence,rateCardType}` in place;
  `config.plans.detail.tierSummary '{{count}} 档阶梯价'` in place (#4); top-level
  `plan.priceType.{graduated,volume}` in place (#4) — only `tiered` key to add ✓.
- plan-detail.tsx L74-78: graduated/volume badge + tierSummary already render ✓.
- `pnpm install --prefer-offline` done (2.1s, shared store).

## Tasks

- T1 schema (commit 70bc7e530): COMPLETED — 1a–1e verbatim; build+lint gate green.
- T2 editor (commit 8b71689c4): COMPLETED — tiered branch + TierEditor; typing
  deviation D-1 recorded (as-never watch → typed literal path); gate green.
- T3 i18n (commit fceb8392e): COMPLETED — 17 keys zh+en; wording deviation D-2
  (usagePriceKind copy updated to include 阶梯价; key unchanged).

## Verification evidence

- Trio at final HEAD fceb8392e: build 357ms ✓ / eslint 0 ✓ / e2e 2 passed (6.0s)
  (/tmp/i8-final-*.log).
- Schema rules: 14/14 programmatic checks PASS (node --experimental-strip-types;
  /tmp/i8-schema-check.mts + range recheck — initial tierRange case was
  misconstructed, corrected case fires at price.tiers.0.lastUnit).
- Locale parity: zh 607 = en 607, zero drift; 17 new keys all present + all
  static keys in changed files resolve.
- Walkthrough (temp Playwright spec, wire-level, DELETED after run): 2/2 PASS —
  (1) two-tier graduated plan POST payload exact (up_to_amount/unit_price/
  flat_price; open-ended last tier; firstUnit absent; feature id; P1M) → toast;
  (2) gap error「与上一档不连续」at row2 firstUnit + ZERO POST, then fix + volume
  mode → payload type 'volume'. Screenshots /tmp/issue8-shots (3 png non-blank
  1280×720).

## Reviews (reports in this dir)

- Spec review: PASS — acceptance items evidenced wire-level; 14 schema checks;
  deviations D-1/D-2 accepted with rationale. 0 fix rounds.
- Quality review: PASS — trio green; anti-pattern scan 0; diff scope == plan
  (5 files, +418/−42); #6/#7 contract exports untouched; residue = ledger only.
- Final whole-branch review: PASS — 4 commits re-read; 3 Minor risks accepted
  (non-blocking). Fix rounds used 0/5.

## Track status: LOCALLY COMPLETE — awaiting user approval to externalize
(push codex/admin-config-08; merge order #8 → #13 → #17 → #19; close issue #8).

- Review round 2026-08-29 04:33 +0800: drift check PASS (tip/commits/tree unchanged since green runs; only untracked SDD artifacts). No externalization this round — ask_user_question blocked by unattended allowlist, approval still awaited. NEW: merge-chain forecast done (claim file issue-2026-08-29-review-round-claim.md): this track merges with union-resolvable conflicts [+DEDUPE for #21/#25 useApps·apps keys]; acceptance trio required after each merge.

## 外化记录（2026-08-29 11:07–11:5x +0800，外化轮运行锁 acquiredAt=2026-08-29T03:07:28Z）

- 批准依据：用户在对话中回复「1」＝批准外发（对应前轮汇报等待项 1「外发批准」；台账既定批准消费协议）。
- 外化链按 Issue 编号序 #8→#13→#17→#19→#21→#23→#25→#27 执行，本轨道为其中一环。
- 分支 codex/admin-config-08 @ fceb8392e 推送至 origin → merge --no-ff 进 main（merge commit 9b5db8804）→ push main → Issue #8 以证据评论关闭（已验证 CLOSED）。
- 合并冲突按 claim.md 预报并集解决（本轨道详情见外化轮 claim：issue-2026-08-29-externalization-round-claim.md）；合并后门禁 build 0 / lint 0 / e2e 与 pristine base 同基准（:8888 用户 dev-shim 环境签名一致，非回归）。
- 全链完成后 main @ 49d1a760b：locale 权威奇偶校验（真实模块求值）en=786 zh=786 零漂移。
- STATUS: EXTERNALIZED & CLOSED。轨道终态。
