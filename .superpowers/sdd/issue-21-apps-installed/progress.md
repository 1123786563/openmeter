# SDD ledger — issue #21 应用：已装列表/卸载/Stripe 换 Key

- Branch codex/admin-config-21 @ base 5a4666ec7 (= origin/main, ls-remote
  verified); worktree /Users/wuyongjun/trea/openmeter-issue-21.
- Final HEAD: 9b0d34d73 (3 commits, +483/−9, 7 files).
- Plan: docs/superpowers/plans/issue-21-apps-installed-uninstall-stripekey.md;
  prescriptive source = issue #21 comment 1 (/tmp/issue-21-plan.md).
- Round claim: main-checkout .superpowers/sdd/issue-21-23-claim.md (02:0x+08).
- SDD mode: subagent/subagent_fork DENIED (unattended allowlist, probed 02:0x)
  → standing DOWNGRADE: controller-executed implementation + controller-run
  programmatic reviews with independent verification runs. Attended spot-check
  remains OPEN (waiting item).

## Pre-implementation anchor verification (controller, 02:1x)

- SDK: OpenMeter root client has NO `apps` property; apps ops live under
  `api.internal.apps` (dist/sdk/sdk.d.ts; internal.d.ts L41+ incl. update;
  src internal.ts L220). README: client.internal.apps.update → PUT
  /openmeter/apps/{appId}. Types App/AppStripe/AppCatalogItem/AppCapability/
  UpdateAppStripeRequest field-verified (types.d.ts L4225/L3623/L1988/L1999).
- DEVIATION D-1: plan's `api.apps.*` → `api.internal.apps.*` (types identical).
- DEVIATION D-2: plan anchor "Organization default tax codes 段之后" absent
  (belongs to unimplemented #20) → Apps section inserted after "Notification
  channels (v1)", before "Helpers".
- v1 root-openapi PUT /api/v1/apps/{id} (secretAPIKey/metadata casing) NOT
  used — v3 SDK serialization is the plan's contract source.
- Locales: #1 stubs config.apps.{title,description} replaced in place.
- ENV FIX (worktree setup): api/spec/packages/aip-client-javascript/dist is
  gitignored → fresh worktree lacked SDK dist → TS2307 on @openmeter/client.
  Built SDK in worktree (pnpm --filter @openmeter/client run build) +
  clean reinstall of web node_modules. Same treatment for track #23 and the
  pristine base comparison worktree.

## Tasks

- T1 API 层 (commit b3680d948): COMPLETED — query-keys apps key; hooks
  useApps/useUpdateApp/useUninstallApp via api.internal.apps (D-1), Apps
  section before Helpers (D-2); mutations invalidate nsPrefix('apps').
  Gate: build ✓ / lint ✓ (/tmp/i21-t1-*.log).
- T2 页面层 (commit 929cc652b): COMPLETED — stripe-key-dialog.tsx (zod
  min(1), PUT body {type:'stripe', name, description?, labels?,
  secretApiKey}, toast+close on success, handleServerError);
  apps/index.tsx (table: name/type/status/capabilities/stripeInfo/actions;
  ready/unauthorized badge variants; stripe row accountId+maskedApiKey+
  livemode badge; empty row colSpan=6; uninstall ConfirmDialog destructive;
  stripe-only 换 Key button — union narrowing worked, zero casts);
  route → AppsPage. Gate: build ✓ / lint ✓ (/tmp/i21-t2-*.log).
- T3 i18n (commit 9b0d34d73): COMPLETED — zh/en config.apps full subtree
  (fields/type/status/capability ×5/stripe/installed/uninstall/
  uninstallConfirm/stripeKey/toast). Gate: build ✓ / lint ✓; locale parity
  zh 622 = en 622; static-key resolution 28/28 zh+en 0 missing
  (/tmp/i18n-keycheck.mjs).

## Review rounds (controller-run programmatic; 0 fix rounds needed)

- Spec review: 17/17 PASS (S1–S17, see spec-review-report.md). Wire contract,
  PUT body construction, invalidation, badge/status/capability rendering,
  route swap, i18n completeness, no `api.apps.` residue.
- Quality review: PASS (quality-review-report.md) — diff scope exactly 7
  planned files; no casts/any; repo patterns (nsPrefix, handleServerError,
  ConfirmDialog, section comments); no new deps; e2e untouched.
- Final full-branch review: PASS (final-review-report.md) — base..HEAD read;
  D-1/D-2 justified and documented; cross-file consistency verified.

## Verification evidence (final HEAD 9b0d34d73)

- Trio: build ✓ (280ms) / lint 0 ✓ / e2e: sign-in smoke ✓, customers smoke
  FAILED — ENVIRONMENTAL, ruled non-regression via pristine-base comparison:
  /tmp/base-check-56 @ 5a4666ec7 fails the IDENTICAL test with the same
  signature while user-side openmeter_shim.py (PID 21142) squats 127.0.0.1:8888
  (vite /api proxy target). Logs: /tmp/i21-e2e.log, /tmp/base-e2e.log.
- Acceptance walkthrough (temp spec, full endpoint mocks incl. namespaces-
  shaped catch-all, DELETED after run; /tmp/i21-walkthrough4.log): 1 passed —
  (1) list renders Sandbox+Stripe rows, 就绪/未授权 badges, 用量上报/支付收款/
  客户开票 capability badges, acct_test_123, sk_live_****abcd, 正式模式;
  (2) uninstall: confirm dialog text 「卸载「Sandbox App」（Sandbox）吗？」,
  zero DELETE before confirm, exactly 1 DELETE to /api/v3/openmeter/apps/{id}
  after confirm → toast 应用已卸载 → refetch drops row;
  (3) Stripe key: dialog shows masked key, PUT to /api/v3/openmeter/apps/{id}
  body deep-equal {type:'stripe', name:'Stripe Prod', description:
  'main integration', labels:{env:'prod'}, secret_api_key:'sk_test_fake_key_123'}
  (SDK wire casing snake_case confirmed) → toast + dialog closes.
- Walkthrough spec defects found & fixed during run (test-side only, not
  product): catch-all mock shape for /namespaces ({default,namespaces[]} not
  page-shape — otherwise layout renders 500 error boundary); duplicate DELETE
  counting (route handler + request listener); strict-mode regex double-match
  (.first()). No product change required.

## Rulings & remaining risk

- RULING (env): :8888 shim failures environmental via base comparison —
  standing precedent re-applied; shim belongs to user's dev stack, NOT killed.
- RULING (tooling): SDK dist gitignored → per-worktree SDK build is part of
  track setup (also applied to base comparison worktree).
- Remaining risk: LOW — unmerged stacking with #8/#13/#17/#19 shares only
  append-only sections (hooks/locales distinct subtrees); eventual merge
  order #8 → #13 → #17 → #19 → #21 → #23 (issue-number ascending).
- Externalization: NOT performed — awaiting user approval (Step 7 gate).
- Review round 2026-08-29 04:33 +0800: drift check PASS (tip/commits/tree unchanged since green runs; only untracked SDD artifacts). No externalization this round — ask_user_question blocked by unattended allowlist, approval still awaited. NEW: merge-chain forecast done (claim file issue-2026-08-29-review-round-claim.md): this track merges with union-resolvable conflicts [+DEDUPE for #21/#25 useApps·apps keys]; acceptance trio required after each merge.

## 外化记录（2026-08-29 11:07–11:5x +0800，外化轮运行锁 acquiredAt=2026-08-29T03:07:28Z）

- 批准依据：用户在对话中回复「1」＝批准外发（对应前轮汇报等待项 1「外发批准」；台账既定批准消费协议）。
- 外化链按 Issue 编号序 #8→#13→#17→#19→#21→#23→#25→#27 执行，本轨道为其中一环。
- 分支 codex/admin-config-21 @ a0918cb53（原 9b0d34d73 + docs 计划文档提交） 推送至 origin → merge --no-ff 进 main（merge commit 0b9450f24）→ push main → Issue #21 以证据评论关闭（已验证 CLOSED）。
- 合并冲突按 claim.md 预报并集解决（本轨道详情见外化轮 claim：issue-2026-08-29-externalization-round-claim.md）；合并后门禁 build 0 / lint 0 / e2e 与 pristine base 同基准（:8888 用户 dev-shim 环境签名一致，非回归）。
- 全链完成后 main @ 49d1a760b：locale 权威奇偶校验（真实模块求值）en=786 zh=786 零漂移。
- STATUS: EXTERNALIZED & CLOSED。轨道终态。

## 独立抽查记录（2026-08-29 11:33–12:1x +0800，抽查轮运行锁 acquiredAt=2026-08-29T03:33:51Z）

- 本会话 subagent 能力首次恢复（此前 15+ 轮被无人值守允许列表禁用）；以全新上下文独立 reviewer subagent 对全部 10 条降级轨道并行抽查。
- VERDICT: PASS（零实质发现；关闭评论 hook 名笔误 useUpdateStripeKey→useUpdateApp 已补更正评论）。D-1 合法性经 SDK sdk.d.ts 17 getter 无 apps 第一手证实；D-2 良性。
- 汇总与更正评论见 issue-2026-08-29-spotcheck-round-claim.md；跨切面门禁（build 0/lint 0/locale 786=786 真实求值/反模式 0/重复键 0）全过。
