# SDD ledger — issue #25 账单档案：列表与创建

- Branch codex/admin-config-25 @ base 5a4666ec7 (= origin/main, ls-remote
  verified); worktree /Users/wuyongjun/trea/openmeter-issue-25.
- Plan: docs/superpowers/plans/issue-25-billing-profiles.md; prescriptive
  source = issue #25 comment 1 (/tmp/issue-25-plan.md, 860 lines).
- Round claim: main-checkout .superpowers/sdd/issue-25-27-claim.md (03:2x+08).
- SDD mode: subagent/subagent_fork DENIED (unattended allowlist, probed 03:2x
  this round) → standing DOWNGRADE: controller-executed implementation +
  controller-run programmatic spec/quality/final reviews with independent
  verification runs (fresh commands, logs on disk). Attended spot-check remains
  OPEN (waiting item).

## Pre-implementation anchor verification (controller, 03:3x)

- Worktree setup per standing RULING: SDK built in-worktree
  (api/spec/packages/aip-client-javascript pnpm install+build) + web
  pnpm install — logs /tmp/i25-{sdk-install,sdk-build,web-install}.log, all OK.
- SDK: root client has NO `apps` property (dist/sdk/sdk.d.ts getters verified);
  billing domain exposes listProfiles/getProfile/createProfile; request/response
  shapes field-verified (ListBillingProfilesQuery.page={size?,number?};
  CreateBillingProfileRequestInput {name,supplier:Party,workflow:WorkflowInput
  (all-optional → `{}` legal),apps:ProfileAppReferences,default}; Party{
  id?,key?,name?,taxId?:{code?},addresses?:{billingAddress:Address}};
  Address 7 optional fields; AppReference={id}; Profile list-render fields).
- DEVIATION D-1 (same as #21 D-1): plan's `api.apps.list` →
  `api.internal.apps.list({ page: { number: 1, size: 100 } })` (GET /openmeter/apps).
  #21 unmerged → useApps defined in-track, verbatim-identical to #21's
  implementation (key `apps: () => ns('apps')`) for zero-conflict stacking.
- web mounts verified: config/billing-profiles route placeholder from #1
  (PlaceholderPage w/ config.billingProfiles.title/description keys);
  i18n stub config.billingProfiles = {title,description} at zh-CN.ts L329
  (en.ts analogous); formatDateTime exists; common.cancel/confirm/submitting
  exist; nsPrefix local helper at hooks.ts tail; queryKeys object tail =
  featureCostQuery.

## Tasks

- T1 API 层 (commit 402159602): COMPLETED — query-keys 三键（billingProfiles/
  billingProfile/apps）；hooks 四个，useApps 走 api.internal.apps（D-1）；
  createMutation 失效 nsPrefix('billing-profiles')。门禁 build ✓ lint ✓
  （/tmp/i25-t1-*.log，首跑即绿）。
- T2 页面层 (commit 2794be8aa): COMPLETED — form-dialog 484 行（zod 扁平表单→
  v3 嵌套映射、AppSlotSelect、default 开关、Textarea 死导入剔除）+ 列表页 155 行
  （appNameMap 解析、默认徽章、空态）+ 路由占位替换。门禁 build ✓ lint ✓
  （/tmp/i25-t2-*.log，首跑即绿）。
- T3 i18n (commit 106a59bc4): COMPLETED — 双 locale 完整子树 30 键（D-2 省略
  死键 form.validation.*）。门禁 build ✓ lint ✓；parity zh 618 = en 618；新增
  30 键全部静态引用（/tmp/locale-parity.js）。

## Final gates & walkthrough (final HEAD 106a59bc4)

- 三连：build ✓ /tmp/i25-f-build.log、lint ✓ /tmp/i25-f-lint.log、test:e2e
  sign-in+customers 冒烟失败 → 环境性裁定（用户侧 :8888 shim 现对未 mock 的
  GET /api/v3/openmeter/namespaces 返回致崩载荷 → namespace-switcher 500 边界；
  pristine base 5a4666ec7 同时刻同型崩溃 + 本轨 20 分钟前 sign-in 尚 PASS 的
  时序证据；探针日志与三仓对照见 quality-review-report）。非回归。
- 验收走查（全端点 mock，temp spec 已删）：/tmp/i25-walkthrough3.log 1 passed —
  列表渲染（名称/描述/供应商+税号/apps 显示名/默认徽章）→ 空提交拦截（0 POST）
  → 填表提交 → 恰 1 次 POST wire 体 deep-equal（supplier 嵌套 snake_case、
  workflow:{}、三槽位、default:false、空可选省略、cn→CN）→ toast → dialog 关
  → 列表失效重取。测试侧迭代 2 次（断言写法，产品零改动）。
- 复审独立复跑：build ✓ lint ✓（/tmp/i25-rev-*.log）。

## Reviews (controller-executed programmatic; reports in this dir)

- Spec review: PASS — acceptance wire 级取证；14 项 SDK 契约核对；D-1/D-2 附
  理由接受。0 fix rounds。
- Quality review: PASS — 范围==plan（7 files +774/−9）；反模式 0；parity 618=618
  且 30 键全引用；环境性 e2e 三仓对照裁定。0 fix rounds。
- Final whole-branch review: PASS — 3 commits 重读；3 Minor 接受（useBillingProfile
  为 #26 预交付 / 单页 100 / app 槽位不滤类型均 plan 明示口径）。0/5 修复轮。

## Rulings & remaining risk

- RULING (env): :8888 shim 致 e2e 冒烟失败 = 环境性（同时刻 pristine-base 对照
  + 时序证据；shim 归用户 dev 栈，未杀）。本轮新增证据：shim 行为随时间变化。
- RULING (tooling): SDK dist gitignored → 每 worktree SDK 构建属轨道设置（沿
  #21/#23 先例；base 对照 worktree 同样处理，用后已删）。
- Remaining risk: LOW — 与 #21 的 useApps/apps 键逐字同形（自动合并）；与其余
  未合入轨共享面均为追加段不相交子树；合并序 #8→#13→#17→#19→#21→#23→#25→#27。

## Track status: LOCALLY COMPLETE — awaiting user approval to externalize
(push codex/admin-config-25; merge order #8→#13→#17→#19→#21→#23→#25→#27; close issue #25).
- Review round 2026-08-29 04:33 +0800: drift check PASS (tip/commits/tree unchanged since green runs; only untracked SDD artifacts). No externalization this round — ask_user_question blocked by unattended allowlist, approval still awaited. NEW: merge-chain forecast done (claim file issue-2026-08-29-review-round-claim.md): this track merges with union-resolvable conflicts [+DEDUPE for #21/#25 useApps·apps keys]; acceptance trio required after each merge.

## 外化记录（2026-08-29 11:07–11:5x +0800，外化轮运行锁 acquiredAt=2026-08-29T03:07:28Z）

- 批准依据：用户在对话中回复「1」＝批准外发（对应前轮汇报等待项 1「外发批准」；台账既定批准消费协议）。
- 外化链按 Issue 编号序 #8→#13→#17→#19→#21→#23→#25→#27 执行，本轨道为其中一环。
- 分支 codex/admin-config-25 @ 8f510d176（原 106a59bc4 + docs 计划文档提交） 推送至 origin → merge --no-ff 进 main（merge commit 4766954b7）→ push main → Issue #25 以证据评论关闭（已验证 CLOSED）。
- 合并冲突按 claim.md 预报并集解决（本轨道详情见外化轮 claim：issue-2026-08-29-externalization-round-claim.md）；合并后门禁 build 0 / lint 0 / e2e 与 pristine base 同基准（:8888 用户 dev-shim 环境签名一致，非回归）。
- 全链完成后 main @ 49d1a760b：locale 权威奇偶校验（真实模块求值）en=786 zh=786 零漂移。
- STATUS: EXTERNALIZED & CLOSED。轨道终态。

## 独立抽查记录（2026-08-29 11:33–12:1x +0800，抽查轮运行锁 acquiredAt=2026-08-29T03:33:51Z）

- 本会话 subagent 能力首次恢复（此前 15+ 轮被无人值守允许列表禁用）；以全新上下文独立 reviewer subagent 对全部 10 条降级轨道并行抽查。
- VERDICT: PASS（零实质发现；关闭评论「货币/工作流字段」措辞已补更正评论；CJK 标点渗入 en 为处方原文，i18n 打磨候选）。useApps 去重声明第一手证实：main 上 hooks/query-keys 各恰一份，合并首亲 diff=tip 减去重复块。
- 汇总与更正评论见 issue-2026-08-29-spotcheck-round-claim.md；跨切面门禁（build 0/lint 0/locale 786=786 真实求值/反模式 0/重复键 0）全过。
