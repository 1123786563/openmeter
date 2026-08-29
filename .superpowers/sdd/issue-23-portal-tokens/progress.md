# SDD ledger — issue #23 门户令牌：发放（一次性明文）

- Branch codex/admin-config-23 @ base 5a4666ec7 (= origin/main, ls-remote
  verified); worktree /Users/wuyongjun/trea/openmeter-issue-23.
- Final HEAD: 540072fc8 (3 commits, +468/−9, 9 files).
- Plan: docs/superpowers/plans/issue-23-portal-tokens-issue.md;
  prescriptive source = issue #23 comment 1 (/tmp/issue-23-plan.md).
- Round claim: main-checkout .superpowers/sdd/issue-21-23-claim.md (02:0x+08).
- SDD mode: subagent/subagent_fork DENIED → standing DOWNGRADE
  (controller-executed + programmatic reviews). Attended spot-check OPEN.

## Pre-implementation anchor verification (controller, 02:1x)

- api/openapi.yaml: POST /api/v1/portal/tokens requestBody = PortalToken
  (subject required; allowedMeterSlugs optional string[]; token readOnly
  "only returned at creation", om_portal_ example). GET listPortalTokens
  exists (#24). ZERO deviations from comment contract.
- legacy.ts EOF append; hooks insert after Notification channels before
  Helpers; query-keys append at object end.
- CustomerPicker {value,onChange}; useMeters paginated {data: Meter[]};
  PageHeader actions; common.cancel/submitting; Checkbox/Command/Popover/
  Label exist; locale stubs config.portalTokens.{title,description} extended
  in place.
- ENV FIX: SDK dist gitignored → per-worktree SDK build + clean web reinstall
  (same as #21 track; details in that ledger).

## Tasks

- T1 API 层 (commit 10af4f901): COMPLETED — legacy.ts PortalToken interface
  (token 注释一次性) + createPortalToken POST /v1/portal/tokens;
  query-keys portalTokens(params) (#24 复用); hooks useCreatePortalToken →
  invalidate nsPrefix('portal-tokens'). Gate: build ✓ / lint ✓.
- T2 页面层 (commit ae66b0eba): COMPLETED — token-once-dialog (om_portal_
  display, copy w/ secure-context + execCommand 降级, onPointerDownOutside
  防误关, amber warning); issue-token-dialog (CustomerPicker + MeterMultiSelect
  Popover/Command/Checkbox, 空=全部; submit validates customer; subject =
  customer.key; meterSlugs.length>0 ? slugs : undefined; token-missing →
  toast.error noPlaintext); index.tsx PageHeader actions 发放按钮; route →
  PortalTokensPage. DEVIATION D-3 (lint adaptation, semantics-equivalent):
  prescriptive useEffect-reset violates react-hooks/set-state-in-effect →
  form fields reset in close event handler; once-dialog copy state derived
  from token value (copiedToken === token). Gate: build ✓ / lint ✓
  (after fix; /tmp/i23-t2-build2.log, i23-t2-lint2.log).
- T3 i18n (commit 540072fc8): COMPLETED — zh/en config.portalTokens full
  subtree (issue/onceTitle/onceDescription/onceWarning/copy/copied/
  copyFailed/copiedClose/form.*/toast.*). Gate: build ✓ / lint ✓; parity
  zh 608 = en 608; static-key resolution 23/23 zh+en 0 missing.

## Review rounds (controller-run programmatic; 0 fix rounds needed)

- Spec review: 17/17 PASS (S1–S17, spec-review-report.md): POST endpoint &
  body shape, subject=customer.key, empty-meter omission, customer-required
  zero-POST guard, noPlaintext honesty path, 防误关, execCommand 降级,
  copy-state ownership, Command+Checkbox multi-select, allMeters 空态,
  PageHeader actions, route swap, no effect-setState, no console.log.
- Quality review: PASS (quality-review-report.md) — 9 planned files only;
  no casts/any; v1-legacy pattern followed; comments capture domain intent
  (一次性明文语义、subject 语义、降级路径); no new deps; e2e untouched.
- Final full-branch review: PASS (final-review-report.md) — D-3 justified
  (repo lint authority over prescriptive comment code; observable behavior
  equivalent: fresh form per open, indicators reset per token).

## Verification evidence (final HEAD 540072fc8)

- Trio: build ✓ (265ms) / lint 0 ✓ / e2e: sign-in smoke ✓, customers smoke
  FAILED — ENVIRONMENTAL via pristine-base comparison (identical failure at
  /tmp/base-check-56 @ 5a4666ec7; :8888 shim PID 21142). Logs:
  /tmp/i23-e2e.log, /tmp/base-e2e.log.
- Acceptance walkthrough (temp spec, full mocks, DELETED; /tmp/
  i23-walkthrough6.log): 1 passed —
  (1) page + dialog render; submit without customer → 请选择客户 + ZERO POST;
  (2) pick customer (combobox → CommandInput → option), toggle Tokens meter,
  submit → POST /api/v1/portal/tokens body deep-equal {subject:'acme-corp',
  allowedMeterSlugs:['tokens_total']} → toast 令牌已发放 → once-dialog shows
  om_portal_TESTTOKEN_plaintext_12345 (textbox toHaveValue), copy click
  no-crash, warning 关闭后无法再次查看…, close → hidden, reload → plaintext
  count 0;
  (3) issue without meters → body deep-equal {subject:'acme-corp'} (field
  omitted = all meters).
- Walkthrough spec defects found & fixed during run (test-side only): picker
  locator — Playwright does NOT compute content-names for role=combobox
  (ARIA name-from-author) → structural [role=combobox]+hasText locators;
  getByDisplayValue unavailable in this Playwright build → getByRole(
  'textbox')+toHaveValue; toast/heading 同文 strict conflict → .first().
  No product change required.

## Rulings & remaining risk

- RULING (env): same as #21 track (:8888 shim, base comparison).
- RULING (a11y/tooling): role=combobox content-name limitation recorded for
  future e2e work (use structural locators).
- Remaining risk: LOW — #24 (list/invalidate) will reuse portalTokens key
  (params-ready) and this page's layout; stacking with other unmerged tracks
  append-only only.
- Externalization: NOT performed — awaiting user approval (Step 7 gate).
- Review round 2026-08-29 04:33 +0800: drift check PASS (tip/commits/tree unchanged since green runs; only untracked SDD artifacts). No externalization this round — ask_user_question blocked by unattended allowlist, approval still awaited. NEW: merge-chain forecast done (claim file issue-2026-08-29-review-round-claim.md): this track merges with union-resolvable conflicts [+DEDUPE for #21/#25 useApps·apps keys]; acceptance trio required after each merge.

## 外化记录（2026-08-29 11:07–11:5x +0800，外化轮运行锁 acquiredAt=2026-08-29T03:07:28Z）

- 批准依据：用户在对话中回复「1」＝批准外发（对应前轮汇报等待项 1「外发批准」；台账既定批准消费协议）。
- 外化链按 Issue 编号序 #8→#13→#17→#19→#21→#23→#25→#27 执行，本轨道为其中一环。
- 分支 codex/admin-config-23 @ 3fbf0df31（原 540072fc8 + docs 计划文档提交） 推送至 origin → merge --no-ff 进 main（merge commit 68ee99eb2）→ push main → Issue #23 以证据评论关闭（已验证 CLOSED）。
- 合并冲突按 claim.md 预报并集解决（本轨道详情见外化轮 claim：issue-2026-08-29-externalization-round-claim.md）；合并后门禁 build 0 / lint 0 / e2e 与 pristine base 同基准（:8888 用户 dev-shim 环境签名一致，非回归）。
- 全链完成后 main @ 49d1a760b：locale 权威奇偶校验（真实模块求值）en=786 zh=786 零漂移。
- STATUS: EXTERNALIZED & CLOSED。轨道终态。

## 独立抽查记录（2026-08-29 11:33–12:1x +0800，抽查轮运行锁 acquiredAt=2026-08-29T03:33:51Z）

- 本会话 subagent 能力首次恢复（此前 15+ 轮被无人值守允许列表禁用）；以全新上下文独立 reviewer subagent 对全部 10 条降级轨道并行抽查。
- VERDICT: PASS（零实质发现；关闭评论 usePortalTokens 表述夸大已补更正评论——该 hook 属 #24）。PortalToken 对照 openapi.yaml 逐字段；一次性明文无持久化证实。
- 汇总与更正评论见 issue-2026-08-29-spotcheck-round-claim.md；跨切面门禁（build 0/lint 0/locale 786=786 真实求值/反模式 0/重复键 0）全过。
