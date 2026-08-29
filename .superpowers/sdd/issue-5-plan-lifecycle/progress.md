# SDD ledger — plan: docs/superpowers/plans/issue-5-plan-lifecycle.md

- Issue: https://github.com/1123786563/openmeter/issues/5 `[admin-config 05/29] 计划：发布/归档/克隆新版本`
- Worktree: /Users/wuyongjun/trea/openmeter-issue-5 (branch codex/admin-config-05, base 45f89e50d = main = origin/main，含 #1–#4 成果)
- Plan commit: c4119ca32 (docs(admin): issue #5 计划发布/归档/克隆新版本实施计划)
- Selection rationale: 最小编号 open Issue（#4 已于 2026-08-27T07:24:06Z merge+close 后）；唯一依赖 #4 已满足；ready-for-agent；处方化评论计划（每步完整代码）；非 PR/重复/阻塞/讨论项。#5 是计划域生命周期写操作，与 #6（创建向导）正交切分。
- Controller: 本次唤醒会话（先完成 #4 外部化：分支已 push、main 已 ff-merge 45f89e50d 并 push、issue #4 已 close，用户批准全部三项）。
- Controller SDK/openapi 契约预核实：见 plan 文档「SDK 契约核实记录」节（publish/archive 均 {planId}→Plan；Plan.status 四值联合；v1 next 201→camelCase Plan；LegacyPlan 子集合法；nsPrefix/queryKeys/ConfirmDialog/common.cancel 全部在位；已知偏差风险 3 条写入 plan）。
- Controller env setup: web/node_modules pnpm install --frozen-lockfile done; SDK dist copied from issue-4 worktree (同基线 45f89e50d 等效) + injected; baseline pnpm build green (280ms), tree clean; node v26.7.0 / pnpm 11.7.0.
- Approval state: NO external side effects for issue #5 without separate user approval (push/merge/close stay local).

## Task 1 — plan lifecycle actions: legacy clone + 3 hooks + detail page action row + i18n (issue comment steps 1-4)

- Status: IN PROGRESS — implementation committed; verification + reviews in flight (2026-08-27 resume wake).
- Resume context: 上一唤醒在实现提交后、任务审查前中断。本次唤醒先做控制器静态自检（5 文件逐一对照 Issue 处方与计划——全部一致），再跑验收三连，然后并行派出 3 个独立 subagent（spec 审查 / quality 审查 / 浏览器走查，端口隔离 9989+4183 防冲突）。
- Controller verification (this wake, worktree web/):
  - `pnpm build` → ✓ built in 349ms, exit 0.
  - `pnpm lint` → clean, exit 0.
  - `pnpm test:e2e` → 2 passed (6.1s), exit 0 (sign-in smoke + customers smoke).
- Reviews dispatched: spec (10ddf319), quality (3991f56a), walkthrough (a6dc3bca). Briefs: spec-review-brief.md / quality-review-brief.md / browser-walkthrough-brief.md.
- Spec review Ruling: **PASS** (report: spec-review-report.md)。2 Minor 均非实现缺陷：(1) .superpowers/sdd/ 目录为 untracked 而非 gitignored（简报措辞不准，tracked 树干净、提交不受影响）；(2) 简报写 "10 fields" 实为处方定义的 9 字段，legacy.ts 与之处方逐字段一致。审查员自跑验证：build exit 0 (305ms) / lint exit 0 / test:e2e exit 0 (2 passed)。逐项核对：hooks invalidation 三元组、按钮显隐规则、3 个 ConfirmDialog 位置（if(!plan) 早退之后）、i18n 两树 40 键程序化深比对一致、无范围蔓延、提交信息精确匹配。
- Quality review Ruling: **PASS** (report: quality-review-report.md)。1 Minor：hooks.ts:238 中文注释（一致性 nit）——控制器核实：该注释为 Issue 处方原文逐字要求，保留即符合 spec，不改。审查员自跑验证：build exit 0 (303ms) / lint exit 0 / test:e2e exit 0 (2 passed)；prettier 4/5 文件干净（hooks.ts 唯一违规为基线预存 recharge hunk，新增行干净）；locale 全文件 528=528 键双向相等、插值一致（脚本贴于报告）。query 卫生全量核查：invalidation 三元组覆盖所有 api.plans 消费者；错误路径与 features 页先例一致。
- Browser walkthrough Ruling: **PASS** (report: browser-walkthrough-report.md)。单次完整运行 43/43 DOM 断言通过、pageerror 0、wire 请求全部核实（publish/archive v3 POST、v1 next 201/409、filter[status] 编码）。场景 a–f 全 PASS（发布/归档/克隆成功/二次克隆 4xx 原文透出/i18n en）。按真实产物记录一处 as-built 差异：active 状态 zh-CN 徽章文案为「已发布」而非简报预估「生效」（i18n 预存键，非缺陷）。端口隔离 9989/4183，清理完成，树净。
- 控制器视觉复核：read_image 不可用（当前模型 glm-5.3 不支持图像输入）；describe_image 插件损坏（baseURL 配置错误）——视觉通道与 #1–#4 同况不可用。控制器以 Pillow 独立复核像素：9 张全部 1440x900、unique colors 1004–1422（>500 非空白）、关键成对差异 1.2%/99.9%/7.7%（状态迁移在像素层面可区分），与走查报告一致。
- Final whole-branch review Ruling: **PASS** (report: final-review-report.md)。11 角度全跑无一跳过；证据自复跑：build exit 0 (356ms) / lint exit 0；截图 9 张实测 1440x900；/tmp/issue5-walkthrough-results.json 43/43 checks + 20 wire 记录与报告逐字一致。三项任务级 PASS 独立复现（locale 528=528/40=40、invalidation 全消费者核查、SDK+v1 契约、prettier 基线字节同一、提交信息精确）。5 Minor：(1) main.tsx 全局 mutations.onError + 页面级 onError 双触发 → 失败双 toast（处方逐字 + 与 features 页同模式，非本分支回归）；(2) 克隆传 plan.id 对非最新发布版必 4xx 原文透出（正是处方要求，无前端预判）；(3) 走查报告日期笔误 2025→2026 + progress.md 成对差异百分比与原始值不映射（原始表正确）；(4) 两份审查报告 prose 键数笔误（13/16 vs 实际 12/语言，程序化输出本就正确）；(5) Esc 可关闭 pending 弹窗（Radix 行为，mutation 照常完成，与全应用模式一致）。

## Task 1 — CLOSED (2026-08-27)

- **Issue #5 本地完成**。全部分支闸门通过：实现（a6ff556ef）+ 验收三连（build/lint/test:e2e 全绿，控制器+4 个独立审查员共 5 组独立运行）+ spec 审查 PASS + quality 审查 PASS + 浏览器走查 PASS（43/43，pageerror 0）+ 全分支最终审查 PASS（11/11 角度）。
- 修复循环：0 轮触发（无 Critical/Important 发现；全部 Minor 均为预存模式、处方逐字要求或报告文字笔误，其中 (3)(4) 为台账/报告侧笔误不影响代码与断言真值）。
- Externalization: **DONE** — 用户于 2026-08-27 ~20:35+08:00 在待命唤醒会话批准 push/merge/close 三项，同会话执行：(1) push codex/admin-config-05 → origin OK（new branch，a6ff556ef）；(2) 主检出 pre-merge 核实 main=origin/main=45f89e50d 树净 → `git merge --ff-only` → a6ff556ef（6 文件 +308/−2）→ push origin main OK（45f89e50d..a6ff556ef），post-verify main=origin/main=a6ff556ef；(3) 完成证据评论 [issuecomment-5439229279](https://github.com/1123786563/openmeter/issues/5#issuecomment-5439229279) + `gh issue close 5 --reason completed` OK — state=CLOSED 2026-08-27T12:36:44Z。Issue #5 全链路终结；台账 CLOSED。前置防护：执行前核实无并发唤醒活动（claim/台账静默 9 分钟）、双树净、gh 认证 OK。
- 待命唤醒（本次外部化执行会话）补充记录：双醒协调全程未竞争写——主唤醒完成全部 SDD 环节并宣告本地完成后，待命唤醒完成独立复核（走查 JSON 工件链 43/43 逐字比对 + 验收三连独立第 6 组复跑 build 313ms/lint/e2e 2 passed 全绿）+ delivery gate PASS（23 项证据，requireSmoke:false 代码+台账交付物，视觉双通道损坏处置在案），随后持用户批准执行上述外部化。
- 外部化状态：未获用户批准，无 push/merge/close/GitHub 写操作。分支 codex/admin-config-05 保留于本地 worktree。
