# SDD ledger — plan: docs/superpowers/plans/issue-7-unit-rate-card.md

- Issue: https://github.com/1123786563/openmeter/issues/7 `[admin-config 07/29] 计划创建向导：unit 用量价卡`
- Worktree: /Users/wuyongjun/trea/openmeter-issue-7 (branch codex/admin-config-07, base f6e767dc3 = main post-#6)
- Plan commit: 047f110f6
- Selection: 2026-08-28 22:26+08:00 round, paired with #12 (see main checkout .superpowers/sdd/issue-7-12-claim.md). blocked_by #6 CLOSED ✓. Data-source correction from issue comment 2: feature dropdown must use useAllFeatures() (unpaginated), NOT useFeatures().
- Env notes (controller, pre-dispatch): pnpm 11.7.0/node v26.7.0 direct; node_modules installed; @openmeter/client file:-dep dist injected BOTH into api/spec/packages/aip-client-javascript/ AND web/node_modules/.pnpm/@openmeter+client@file+..*/node_modules/@openmeter/client/ (pnpm copies file: deps at install; issue-6's proven-green dist used as source). Baseline pnpm build GREEN (270ms) at base before any change. tsbuildinfo caches cleaned once after env repair.
- Approval state: NO external side effects (push/merge/close/GitHub) without explicit user approval.

## Task 1 — unit 用量价卡（schema/price-editor/wizard 扩展 + i18n）

- Status: pending dispatch (implementer subagent)
- Downgrade note (2026-08-28 22:3x+08:00): subagent / subagent_fork / workflow ALL denied by
  unattended-session allowlist (first-hand probe this session, same class as issue-6 takeover
  #3/#4). superpowers:subagent-driven-development skill file absent on disk. FALLBACK per
  issue-6 takeover precedent: controller-executed implementation + controller-executed
  programmatic reviews (diff inspection, structural i18n compare, build/lint/e2e, targeted
  probes), full ledger; isolation downgrade flagged for spot-check by an attended session.

## Task 1 rulings (2026-08-28 22:30–23:0x+08:00, controller-executed under documented downgrade)

- Implementation: commit 8da5cc969 feat(admin): 计划向导支持 unit 用量价卡 (5 files +286/−131, amend含 prettier 归一). Plan commit 047f110f6.
- Spec review (programmatic): PASS — priceFormSchema unit 分支与 AMOUNT refine 按评论逐字落地；rateCardSchema superRefine 三规则（flat_fee∈{free,flat}、usage_based 强制 featureId+unit、billingCadence=null 仅 free|flat）错误 path 精确定位 type/featureId/billingCadence；toPriceInput case 'unit'；toRateCardInput feature:{id} 映射（编译级证明 RateCardInput.feature 存在）；RATE_CARD_TYPES 扩展；RateCardRow 抽取解决 hooks-in-map；类型切换重置 price 形态（shouldValidate:true）；one_time 选项按 priceKind 门控；数据源修正已落实 useAllFeatures()（issue 评论 2，非分页 useFeatures）——walkthrough 证明下拉渲染全量 mock 列表。i18n 双 locale 564=564 叶子键结构一致，0 未解析静态 key；rateCardType.usage_based 双语在位（#1 预置）。
- Quality review (programmatic): PASS — tsc -b (vite build) exit 0；eslint exit 0；prettier --check 改动文件全过（amend 修正）；新增行 0 any/0 as any/0 @ts-ignore/0 eslint-disable/0 console.log；routeTree.gen.ts 未触碰；#6 契约（toPlanPhases/toCreatePlanRequest/defaultRateCard/EMPTY_PLAN/PriceFormValue 导出）零改动（diff 证明）；其他域文件零改动。
- Browser walkthrough (temp playwright spec, wire-level mocks): PASS 2/2 — (1) 端到端：flat+usage_based 双卡计划创建，feature 下拉含 'Token API（token_api）'（useAllFeatures 数据源），'单价（CNY/单位）' 标签带币种插值，unit 卡计费周期下拉无 '一次性' 选项，POST /api/v3/openmeter/plans wire 载荷精确断言（flat 卡 price{type:'flat',amount:'9.9'} 无 feature；unit 卡 feature{id:'feat-token-api'}+price{type:'unit',amount:'0.05'}），toast 计划已创建（草稿）；(2) 负向：未选 feature 提交 → 翻译错误 '用量计费价目卡必须选择功能' 渲染于下拉处，POST 0 次。截图 /tmp/issue7-shots/ 2 张（usage-card-filled/feature-required-error）。temp spec 已删，树净。
- Final whole-branch review (vs f6e767dc3): PASS — 6 文件（1 计划文档+5 代码）全在计划范围；i18n PASS；回归面：#6 向导语义经 walkthrough 步骤门禁/冒烟双绿确认；无秘密/无调试残留。
- Acceptance trio at final HEAD 8da5cc969: pnpm build exit 0（341ms）+ pnpm lint exit 0 + pnpm test:e2e 2 passed（6.1s）。
- Ruling: Task 1 CLOSED — 本地完成。0 fix rounds（无 Critical/Important 发现；实现为控制器直接执行，修复与审查同体，独立性降级已记录待补查）。
- Remaining risks (Minor): (1) feature 下拉无搜索/长列表虚拟化（功能目录规模增长后的 UX 债，spec 未要求）；(2) _loading 哨兵 SelectItem 为评论处方模式；(3) 切换类型重置 price 会丢弃已填金额（处方行为，防非法组合残留）；(4) 隔离降级（无独立 subagent 审查者）——建议 attended 会话抽查本轨道 Ruling。

## 完成后唤醒复核 #1（2026-08-28 23:07–23:12+08:00，本轮运行锁 acquiredAt=2026-08-28T15:07:23Z）

- 等待批准状态不变 → 按既定先例本轮仅报告等待事项：不外发、不领取新 Issue。
- Subagent 探测（本会话第一手）：`subagent` 再次被无人值守允许列表拒绝（与 issue-6 各 takeover 同类错误）——隔离降级抽查仍 OPEN，待 attended/具备 subagent 能力的会话执行。
- 一致性复核全绿（第一手）：worktree 干净（仅未跟踪台账目录）、tip 8da5cc969、分支 diff vs f6e767dc3 = 6 文件 +340/−131 与台账一致、Issue #7 在 fork 上仍 OPEN、origin/main=f6e767dc3 未移动、端口 9999/4173/8888 空闲、用户 dev :5173 未受影响。
- 验收三连复跑 GREEN @8da5cc969：pnpm build exit 0 / pnpm lint exit 0 / pnpm test:e2e 2 passed (10.4s)（/tmp/issue7-wake-{build,lint,e2e}.log；:8888 ECONNREFUSED 为已记录的正常绿跑噪音）。
- STATUS UNCHANGED: LOCAL COMPLETE @8da5cc969, AWAITING USER APPROVAL for push/merge/close。

## 完成后唤醒复核 #2（2026-08-28 23:56:50 +0800，本轮运行锁 acquiredAt=2026-08-28T15:48:50Z）

- 等待批准状态不变 → 按既定先例本轮仅报告等待事项：不外发、不领取新 Issue（2 条等待轨道 = 并行上限，进行中轨道数评估于第 1 步）。
- Subagent 探测（本会话第一手）：subagent 与 subagent_fork 均再次被无人值守允许列表拒绝（同 issue-6 各轮同类错误）——隔离降级抽查仍 OPEN，待 attended/具备 subagent 能力的会话执行；superpowers:subagent-driven-development 技能文件本轮依旧不在磁盘/技能目录。
- 发布判断（第 7 步）：剩余风险未达"很小"（隔离降级抽查未闭合 + 台账记录 Approval state：无明确用户批准不外发），GitHub #7 仅 2 条 2026-08-26 处方评论（作者=owner）、无任何批准信号 → 本轮不对该轨道执行外发操作。
- 一致性复核全绿（第一手）：worktree 干净（仅未跟踪台账目录）、tip 8da5cc969、diff vs f6e767dc3 = 6 文件 +340/−131 与台账一致、Issue #7 在 fork 上仍 OPEN、origin/main=f6e767dc3 未移动、端口 9999/4173/8888 空闲、用户 dev :5173（node pid 47701）未受影响。
- 验收三连复跑 GREEN @8da5cc969（/tmp/issue7-wake2-{build,lint,e2e}.log）：pnpm build exit 0 / pnpm lint exit 0 / pnpm test:e2e 2 passed (9.8s)。
- STATUS UNCHANGED: LOCAL COMPLETE @8da5cc969, AWAITING USER APPROVAL for push/merge/close。

## 完成后唤醒复核 #3（2026-08-29 00:07–00:1x +0800，本轮运行锁 acquiredAt=2026-08-28T16:07:30Z）

- 等待批准状态不变 → 按既定先例本轮仅报告等待事项：不外发、不领取新 Issue（2 条等待轨道 = 并行上限）。
- Subagent 探测（本会话第一手）：subagent 与 subagent_fork 均再次被无人值守允许列表拒绝（todo_write 同列表亦拒）——隔离降级抽查仍 OPEN，待 attended/具备 subagent 能力的会话执行；superpowers:subagent-driven-development 技能文件本轮依旧不在磁盘/技能目录。
- 发布判断（第 7 步）：剩余风险未达"很小"（隔离降级抽查未闭合 + 台账记录 Approval state：无明确用户批准不外发），GitHub API 核验 Issue #7 updatedAt=2026-08-26T17:30:31Z 不变、无任何批准信号 → 本轮不对该轨道执行外发操作。
- 环境注意（本轮新发现）：gh 无默认 repo 且本 checkout 同时有 origin(fork)/upstream(openmeterio) 双远端时，裸 `gh issue list` 解析到 upstream——fork 侧操作必须显式 `-R 1123786563/openmeter`（本轮已按此核验）。
- 一致性复核全绿（第一手）：worktree 干净（仅未跟踪台账目录）、tip 8da5cc969、diff vs f6e767dc3 = 6 文件 +340/−131 与台账一致、Issue #7 在 fork 上仍 OPEN、origin/main=f6e767dc3（ls-remote 权威核验）未移动、端口 9999/4173/8888 空闲、用户 dev :5173（node pid 47701）未受影响。
- 验收三连复跑 GREEN @8da5cc969（/tmp/issue7-wake3-{build,lint,e2e}.log）：pnpm build exit 0 / pnpm lint exit 0 / pnpm test:e2e 2 passed (11.6s)；两轨 e2e 串行执行，端口无争用。
- STATUS UNCHANGED: LOCAL COMPLETE @8da5cc969, AWAITING USER APPROVAL for push/merge/close。

## 独立抽查记录（2026-08-29 11:33–12:1x +0800，抽查轮运行锁 acquiredAt=2026-08-29T03:33:51Z）

- 本会话 subagent 能力首次恢复（此前 15+ 轮被无人值守允许列表禁用）；以全新上下文独立 reviewer subagent 对全部 10 条降级轨道并行抽查。
- VERDICT: PASS（零发现）。comment-2 修正 useAllFeatures() 落实；toRateCardInput feature:{{id}} 与 toPriceInput case 'unit' 第一手确认；one_time 门控/类型切换重置均在。
- 汇总与更正评论见 issue-2026-08-29-spotcheck-round-claim.md；跨切面门禁（build 0/lint 0/locale 786=786 真实求值/反模式 0/重复键 0）全过。
