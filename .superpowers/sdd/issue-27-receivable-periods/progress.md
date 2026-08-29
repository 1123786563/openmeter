# SDD ledger — issue #27 客户详情：应收周期 tab 与外部发票登记

- Branch codex/admin-config-27 @ base 5a4666ec7 (= origin/main, ls-remote
  verified); worktree /Users/wuyongjun/trea/openmeter-issue-27.
- Plan: docs/superpowers/plans/issue-27-receivable-periods.md; prescriptive
  source = issue #27 comment 1 (/tmp/issue-27-plan.md, 649 lines).
- Round claim: main-checkout .superpowers/sdd/issue-25-27-claim.md (03:2x+08).
- SDD mode: subagent/subagent_fork DENIED (unattended allowlist, probed 03:2x
  this round) → standing DOWNGRADE: controller-executed implementation +
  controller-run programmatic spec/quality/final reviews with independent
  verification runs (fresh commands, logs on disk). Attended spot-check remains
  OPEN (waiting item).

## Pre-implementation anchor verification (controller, 03:3x)

- Worktree setup per standing RULING: SDK built in-worktree + web pnpm
  install — logs /tmp/i27-{sdk-install,sdk-build,web-install}.log, all OK.
- SDK: commerce.listReceivablePeriods / updateExternalInvoice verified;
  ListReceivablePeriodsQuery.page=CursorPaginationQueryPage{size?,after?,before?};
  ReceivablePeriodPaginatedResponse={data:CommerceReceivablePeriod[],
  meta:CursorMeta}; CursorMetaPage.next: string|null; CommerceExternalInvoiceUpdate
  ={idempotencyKey,invoiceNumber,invoiceUrl?,issuer?,issuedAt?:Date} — all
  field-identical to prescription (zero deviations anticipated).
- web mounts verified: customer-detail.tsx tabs block (entitlements trigger
  last, L480+); events/index.tsx loadMore cursor pattern L54-83 (prescription
  mirrors it); status-badge.tsx tones refund domain L69-75 (receivablePeriod
  inserts after refund); formatFen/formatNumber/formatShortDateTime exist;
  common.cancel/confirm/submitting exist.

## Tasks

- T1 API 层+idempotency+tones (commit ec061235e): COMPLETED — key
  receivablePeriods(customerId, params)；两 hooks；lib/idempotency.ts（D-1
  修复：`in` 收窄 never → typeof 检查，首跑编译失败→修复→绿，/tmp/i27-t1-*2.log）；
  tones 追加 receivablePeriod 五态（refund 后）。门禁 build ✓ lint ✓。
- T2 页面层 (commit 0d6e5d875): COMPLETED — receivable-periods-tab（事件页
  loadMore 同构累积分页；PAGE_SIZE 死常量剔除，D-3）+ external-invoice-dialog
  （per-open UUID 重置）+ customer-detail 三处最小接入。门禁 build ✓ lint ✓
  （/tmp/i27-t2-*2.log）。
- T3 i18n (commit 919f2bfc3): COMPLETED — customers.receivablePeriods 子树
  （首版误插 subscriptions 段尾，自检 parity 脚本当场抓出后修复）+ tabs 键 +
  顶层 receivablePeriod.status.*（D-2）。门禁 build ✓ lint ✓；parity 616=616；
  26 键 = 21 静态 + 5 模板字面量。

## Final gates & walkthrough (final HEAD 919f2bfc3)

- 三连：build ✓ /tmp/i27-f-build.log、lint ✓ /tmp/i27-f-lint.log、test:e2e
  sign-in+customers 冒烟失败 → 环境性裁定（三仓同时刻对照：base 5a4666ec7、
  #25、本轨登录探针同型 TypeError —— shim 对未 mock /namespaces 载荷致
  namespace-switcher 500；非回归，详见 quality-review-report）。
- 验收走查（全端点 mock，temp spec 已删）：/tmp/i27-walkthrough2.log 1 passed —
  tab 位于权益后 → 页 1 两行（逾期/已付清徽章）→ 加载更多（page[after]=
  CURSOR_1 raw cursor）追加部分支付行且页 1 保留 → 登记外部发票：空提交拦截
  （0 PUT）+ UUID 预填断言 → 提交恰 1 次 PUT /api/v3/customers/.../rp-1/
  external-invoice wire 体 deep-equal {idempotency_key,invoice_number,
  invoice_url} → toast → dialog 关。测试侧迭代 1 次（mock 路径：commerce 域
  基路径为 /api/v3/ 无 openmeter 段——请求捕获实证，产品零改动）。
- 复审独立复跑：build ✓ lint ✓（/tmp/i27-rev-*.log）。

## Reviews (controller-executed programmatic; reports in this dir)

- Spec review: PASS — 两项 acceptance wire 级取证；12 项 SDK 契约核对；D-1/D-2/
  D-3 附理由接受。0 fix rounds。
- Quality review: PASS — 范围==plan（9 files +540/−0）；反模式 0；parity 616=616
  （26 键引用口径含模板字面量 5）；T3 误插缺陷自检抓出未流入提交；环境性 e2e
  三仓对照裁定。0 fix rounds。
- Final whole-branch review: PASS — 3 commits 重读；3 Minor 接受（无显式错误
  重试态 / datetime-local 本地时区 / 回退分支假设 crypto 存在——均 plan 同口径
  或目标环境保证）。0/5 修复轮。

## Rulings & remaining risk

- RULING (env): 同 #25 轮（:8888 shim；三仓同时刻对照含时序证据）。
- RULING (tooling): 每 worktree SDK 构建；base 对照 worktree 用后已删。
- RULING (wire): commerce 域基路径 /api/v3/（无 openmeter 段）——本轮请求捕获
  实证并记档，供 #28 及后续 commerce 域工单直接引用。
- Remaining risk: LOW — customers 域独占无堆叠冲突；#28 复用点（idempotency
  助手 / params-ready key / tab 布局）已预留；合并序
  #8→#13→#17→#19→#21→#23→#25→#27。

## Track status: LOCALLY COMPLETE — awaiting user approval to externalize
(push codex/admin-config-27; merge order #8→#13→#17→#19→#21→#23→#25→#27; close issue #27).
- Review round 2026-08-29 04:33 +0800: drift check PASS (tip/commits/tree unchanged since green runs; only untracked SDD artifacts). No externalization this round — ask_user_question blocked by unattended allowlist, approval still awaited. NEW: merge-chain forecast done (claim file issue-2026-08-29-review-round-claim.md): this track merges with union-resolvable conflicts [+DEDUPE for #21/#25 useApps·apps keys]; acceptance trio required after each merge.

## 外化记录（2026-08-29 11:07–11:5x +0800，外化轮运行锁 acquiredAt=2026-08-29T03:07:28Z）

- 批准依据：用户在对话中回复「1」＝批准外发（对应前轮汇报等待项 1「外发批准」；台账既定批准消费协议）。
- 外化链按 Issue 编号序 #8→#13→#17→#19→#21→#23→#25→#27 执行，本轨道为其中一环。
- 分支 codex/admin-config-27 @ 3367d0107（原 919f2bfc3 + docs 计划文档提交） 推送至 origin → merge --no-ff 进 main（merge commit 49d1a760b）→ push main → Issue #27 以证据评论关闭（已验证 CLOSED）。
- 合并冲突按 claim.md 预报并集解决（本轨道详情见外化轮 claim：issue-2026-08-29-externalization-round-claim.md）；合并后门禁 build 0 / lint 0 / e2e 与 pristine base 同基准（:8888 用户 dev-shim 环境签名一致，非回归）。
- 全链完成后 main @ 49d1a760b：locale 权威奇偶校验（真实模块求值）en=786 zh=786 零漂移。
- STATUS: EXTERNALIZED & CLOSED。轨道终态。

## 独立抽查记录（2026-08-29 11:33–12:1x +0800，抽查轮运行锁 acquiredAt=2026-08-29T03:33:51Z）

- 本会话 subagent 能力首次恢复（此前 15+ 轮被无人值守允许列表禁用）；以全新上下文独立 reviewer subagent 对全部 10 条降级轨道并行抽查。
- VERDICT: PASS（零实质发现；关闭评论 POST→PUT 措辞已补更正评论）。commerce /api/v3/ 无 /openmeter 段经 client.ts→SDK→Go r.Put 全链证实；载荷逐字段对照通过。
- 汇总与更正评论见 issue-2026-08-29-spotcheck-round-claim.md；跨切面门禁（build 0/lint 0/locale 786=786 真实求值/反模式 0/重复键 0）全过。
