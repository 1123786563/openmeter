# SDD ledger — plan: docs/superpowers/plans/issue-31-anchor-gitignore-server.md

Issue: https://github.com/1123786563/openmeter/issues/31
Branch: codex/admin-config-31 (base main 158251056) | Worktree: /Users/wuyongjun/trea/openmeter-issue-31
Round: 2026-08-29 并行轮（唯一候选 #31，单轨）| Lock acquiredAt=2026-08-29T12:57:00Z

## 第 2 步领取记录

- 入选 #31：fork 唯一 open Issue（bug+ready-for-agent，无 needs-triage/needs-info，blocked_by=[]，无评论）；无其他候选故无串行冲突。
- 第 1 步核验：#1-#30 全部关闭、全部分支已合入；遗留 4 worktree（#11/#15/#16/#29）为已终结轨道保留产物，非进行中；进行中轨道 0 < 4。

## Pre-flight 冲突扫描（表）

| 对/任务 | 检查 | 结论 |
|---|---|---|
| T1↔T2 | T1 产出（.gitignore+7 新增文件，commit 落分支）；T2 消费（分支 tip 的 temp worktree 跑 build/vet） | 无文件交叠；T2 强依赖 T1 完成，串行执行 |
| T1 自身 | 编辑 .gitignore:36 + 复制 4 路径 + add + commit；验证 A1-A4 + worktree build/vet 转绿 | 自洽；红色基线已复现（build exit 1），修复即转绿 |
| T2 自身 | temp worktree from tip；B1 build/B2 vet ./...（根模块，AGENTS.md：根 ./... 本就排除 e2e）/B3 主检出 build | 自洽；B2 前置事实：主检出全量 `go vet ./...` exit 0（预检实测 2026-08-29），pristine 修复后为同文件集，无预存失败误伤 |
| 计划 vs 评审标准 | 无「断言为空的测试」/无复制逻辑块等评审雷区 mandates | 干净 |

## Rulings（预检 + 范围）

- Ruling-1（范围）: Issue 正文修复方向第 2 条只点名 auth 三文件，本计划恢复全部四个被吞路径（+commerce_loopback.go +两测试文件）— commerce_loopback.go 被 main.go:178 生产引用，缺它 pristine build 必败=Issue 验收不可能满足；测试文件同根因受害面 — 若错，成本=多恢复两文件（内容逐字节来自主检出磁盘，与「auth 三文件」同源同级，无新增风险）。
- Ruling-2（外部化预案）: 外化合并前需 rm 主检出四个未跟踪磁盘副本（与分支内容逐字节相同）以避免 untracked-overwrite 拒绝 — 若错，成本=外化轮合并被 git 拒绝后退回 rm 再合（无内容损失）。

## 环境事实

- 红色基线：worktree `go build ./...` exit 1，错误与 Issue 逐字一致（cmd/server/main.go:31:2 no required module ... openmeter/server/auth）。
- 主检出全量 `go vet ./...` exit 0（文件全在盘状态）。
- 四被吞路径从未被任何提交跟踪（`git log --all` 空）；根 `server` = 229MB CodeGraph 二进制 = 规则本意目标（commit 788a53dfc，# CodeGraph 块）。

## Standing DOWNGRADE（本轮探测记录）

- 本会话 `subagent` 派发被拒：「工具 'subagent' 不在无人值守自动化允许列表中」（2026-08-29T12:5xZ，真实 Task 1 派发尝试）。
- `subagent_fork` 补测同拒：同文案。
- 与既往 10+ 轮一致 → 控制器直执行 implementer/reviewer 角色，全部验证第一手命令+真实输出为证，评审以结构化分遍（spec 合规 / 代码质量）文档化于本工作区。

## Task 执行记录

- Task 1: complete (commits 33921bc41..b40d256c7, review clean — 双遍 PASS 见 task-1-review.md；门禁：check-ignore 4 路径不再命中/根二进制仍被 /server 命中、ls-files 6 文件、cmp 6/6 逐字节一致、worktree build 红→绿 exit 0、scoped vet exit 0)
  - Ruling-3（勘误）: 计划验收节「.gitignore + 7 个纯新增文件」为计数笔误，实为 6 新增 + 1 修改 — 验收语义不变 — 若错，成本=零（纯文档计数）。
  - minor（deferred）: 无代码级 minor；计划文本勘误已随 Ruling-3 闭合。
- Task 2: complete (无新 commit；pristine 验收全绿见 task-2-report.md：B1 build exit 0 / B2 全量 vet exit 0 / B3 主检出 build exit 0 且磁盘未动；强化=pristine 三包真实测试 3/3 ok；临时 worktree 已清理)

## 附加门禁与 Ruling-4

- 后台进程探测：`run_in_background` 被拒（「无人值守运行不允许启动后台进程」）——与 subagent/subagent_fork 同一允许列表，Standing DOWNGRADE 第三通道证据。
- `make lint-go`（前台完整跑）：exit 2，5 发现。分类：4 项预存于 main（servicevalidation_test.go:69 gci、order/service_test.go:810 与 refund/service_test.go:2434 ineffassign、cmd/server/commerce.go:1078 staticcheck——均为本分支未触碰文件）；1 项随恢复文件进入 git 视野（auth_test.go:110 sloglint → slog.DiscardHandler 建议）。
- Ruling-4: auth_test.go:110 sloglint 发现 parked，不在本轨修复 — 计划硬约束「恢复文件逐字节、零润色」+ pristine main 的 lint-go 本就 4 发现红（非净门禁）+ 该修复属未来 lint 清理轮（须连同 4 项预存一起） — 若错，成本=一个已红门禁里多一条单行建议发现，后续一行可修。

## 最终全分支审查

- Final review: PASS — RELEASE_READY_PENDING_USER_APPROVAL（五角度见 final-review.md；分支 tip b40d256c7，2 commits：33921bc41 计划 + b40d256c7 修复）
- 外部化等待用户批准；预案=计划文档「外部化预案」节 5 步（含 Ruling-2 的未跟踪副本 rm 前置）。
- 本地完成判定（第 6 步四要素）：实现 ✅ / 覆盖测试 ✅（三包真实测试+pristine build/vet）/ 任务审查 ✅（T1 双遍+T2 报告）/ 全分支审查 ✅。

## 轮次终态

- 分支 codex/admin-config-31 保留（未 push 未合并）；worktree /Users/wuyongjun/trea/openmeter-issue-31 保留（外化轮消费）。
- 运行锁 2026-08-29T12:57:00Z 于本轮结束时删除。

## 外化记录（2026-08-29T14:36:32Z 轮，用户对话批准「批准」）

- 批准通道：用户在对话中直接回复批准（等待事项原文「批准外发 #31」通道），凭 wait-round-31-claim-2.md 记录执行。
- 前置核验：worktree 干净、tip b40d256c7、分支未 push、main=origin/main=158251056 零漂移；6 未跟踪副本 cmp 逐字节一致复验 6/6 后 rm。
- 步骤 1 push：codex/admin-config-31 → origin（远端 tip b40d256c7）。
- 步骤 2 rm：主检出 6 个未跟踪副本删除（零信息损失，内容已全量入 git）。
- 步骤 3 merge：`git merge --no-ff` → f9b682cf6（零冲突，8 文件 +1483/-1；ls-files 6 恢复文件全命中；check-ignore：根 `server` 仍被 `/server` 命中、auth.go 不再被忽略）。
- 步骤 4 门禁（文件自此来自 git）：go build ./... exit 0；go vet ./... exit 0；go test ./openmeter/server/ ./openmeter/server/auth/ ./api/v3/server/ = 3/3 ok（4.1s/4.6s/4.7s）；make lint-go exit 2 = 恰 5 发现、与既档集合完全一致（4 项 main 预存 + Ruling-4 parked auth_test.go:110 sloglint），合并零新增。
- 步骤 5 关单：#31 附证据评论并 close --reason completed；main push fork。
- 轨道终态：RELEASED。closed 集 = {1-31} 全清，open = 0。
