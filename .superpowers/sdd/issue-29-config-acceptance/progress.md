# SDD ledger — plan: docs/superpowers/plans/issue-29-config-acceptance.md

- Issue: https://github.com/1123786563/openmeter/issues/29 `[admin-config 29/29] 配置域全链路验收`
- Branch: codex/admin-config-29 @ base 601fe0b6e (= main tip); worktree /Users/wuyongjun/trea/openmeter-issue-29（web 环境已就绪：SDK dist + pnpm install + @openmeter/client 布局复刻，setup-worktree-web.sh OK）
- 领取台账（含环境 Ruling）: ../issue-2026-08-29-acceptance-round-29-claim.md
- 模式：本会话无人值守策略已拒后台进程（run_in_background）；subagent 通道可用性以 T1 首 dispatch 探测为准，若拒则沿用 Standing DOWNGRADE（控制器直执行+第一手证据）。

## Pre-flight 冲突扫描（计划逐对/逐任务自洽核对）

| 对/任务 | 检查 | 结果 |
| --- | --- | --- |
| T1 ↔ T2 | T1 产 smoke 第 3 用例；T2 产 README 用例清单第 3 条描述同一用例 | 一致：README 文案为处方逐字，与用例名 `config plans smoke: ...` 语义同指；无文件重叠（smoke.spec.ts vs README.md） |
| T1 ↔ T3 | T3 走查复用 T1 已验证的登录/preview 设施；走查独立于 smoke.spec.ts（工件脚本，不入仓库） | 一致：T3 不改 smoke.spec.ts；无双向依赖 |
| T3 ↔ T4 | T4 全量回归含 test:e2e（T1 产物）与 build（T2 后复核）；T3 不改受控文件 | 一致：T4 是终局门禁，覆盖前序产物 |
| T1 自洽 | 处方代码引用 `MOCK_CUSTOMER_NAME` 后新增 `MOCK_PLAN_NAME`；当前文件恰有 2 用例（sign-in/customers）→ 3 | 一致：断言与文件现状吻合（已读 smoke.spec.ts 头部确认 login helper + page.route 模式在位） |
| T2 自洽 | README 现有「E2E 冒烟测试」小节恰列 1、2 两条；处方插入点「之后、『与 OpenMeter API / Casdoor 的关系』之前」 | 一致（已读 README 全文确认小节顺序） |
| T3 自洽 | 四块写操作端点：features(v3 POST)、channels(v1 POST)、tax-codes(v3 POST)、apps install(internal SDK)——与 hooks.ts/legacy.ts 实测一致 | 一致；走查脚本 mock 这些端点需 wire 格式与 openapi/SDK 对齐（T3 内核对） |
| 计划 ↔ 仓库约束 | 无后端改动；playwright.config/mock-idp 不改 | 一致：全部任务仅触 web/e2e/smoke.spec.ts、web/README.md、SDD 工件 |

裁定：扫描干净，无需裁决项。T1 开工。

## Tasks

- T1 e2e 冒烟第 3 用例（smoke.spec.ts + test:e2e 3 绿）
- T2 README 两处补充
- T3 浏览器全链路走查四块（stateful mock + 4 截图 + 报告）+ 真实环境尽力探查
- T4 全量回归（web 三连 + go build + Go 零 diff）
- 最终全分支审查（最强模型）

## 通道探测记录（2026-08-29T12:0xZ）

- subagent 工具：被无人值守自动化允许列表拒（同 todo_write / run_in_background）。
- Ruling: Standing DOWNGRADE 生效——控制器逐任务直执行，每任务留第一手证据（命令+输出落盘），spec/quality 两遍审查由控制器分遍执行并记录，最终全分支审查同样控制器执行。与 #11/#15/#16 轮同模式。
- Ruling: 任务提交采用每任务一提交（处方步骤 6 为单提交收尾）——与前序 28 轨惯例一致，终态树相同；若错仅提交粒度差异（观感级）。

## T1 执行记录

- T1: complete (commits 73493cd3..d93e912c3, review clean 双遍 PASS)。要点：处方用例逐字落位；处方授权第二读端点 mock（namespaces → NamespaceList wire）；customers「环境性失败」20+ 轮真因=同一 Header namespaces 崩溃，共享 helper 一并修复，套件 3/3 过。Minor：处方注释 listAll vs 实际 list（处方逐字保留）。
- T2: complete (d93e912c3..<T2hash>, review clean)。README 用例清单第 3 条 + 「配置域（/config）」小节均处方逐字；插入点正确（用例清单后、关系小节前）；11 条路由与实际路由树逐一核对全匹配。Ruling：第 3 条括注「页面实现前断言占位文案」为处方逐字文本，描述用例的双择断言结构仍然准确，保留——若错仅文档措辞轻微历史化（观感级）。
- T3: complete（DONE_WITH_ENVIRONMENT_SHORTFALL）。四块全链路写操作全过（真实 UI 请求体捕获+wire 精确响应+invalidate→refetch→新行+toast+4 张 1440x900 截图+像素证据）；产品缺陷 0（三次失败均为走查脚本自身问题：install 响应包裹形状/glob `*` 不跨 `/`/严格模式断言）；Tier-2 真实探查不可达（browser 工具允许列表拒 + :5173 会话中途停服 curl 000 + shim 无状态 + compose 栈未起），如实记录。工件归档 walkthrough/。
- T4: complete（DONE_WITH_PREEXISTING_FINDING）。前端三连全绿（build ✓ / lint 0 err 2 既有 warn / e2e 3 passed，首跑 sign-in 单次抖动与历史同模式，重跑全过）；零 Go diff；主检出 go build exit 0。**发现 main 预存缺陷**：.gitignore:36 无锚定规则 `server` 吞掉 openmeter/server/auth/（fork 提交 925f6be4d 引用未入库）——pristine clone 必然编译失败，主检出靠 ignored 本地文件构建。修复超 #29 红线，建议新 issue（文案已备）。
- Ruling: go build 在主检出验证（ignored 文件在位=用户真实环境）+ 分支零 Go diff 证明系列无后端误改；不在验收轨内修 gitignore 缺陷——若错则 pristine-clone 构建红多停留一个 issue 周期（已文档化，代价可控）。

## 终审与轨道终态（2026-08-29T12:3xZ）

- 最终全分支审查：FINAL PASS — RELEASE_READY_PENDING_USER_APPROVAL（五角度，0 Critical/Important，3 Minor 记录在案；报告 final-review-report.md）。
- 轨道终态：本地完成（T1/T2/T3/T4 全过 + 终审过）。分支 codex/admin-config-29 @ 93e7998f1（3 提交，148 行，web/e2e/smoke.spec.ts + web/README.md + 计划文档）。
- 剩余风险（汇报用）：(1) 真实后端走查未执行（环境不可达三证：浏览器工具允许列表拒、:5173 中途停服、shim 无状态；stateful-mock 全链路证据替代，等用户裁决证据等级）；(2) main 预存缺陷 .gitignore:36 吞 openmeter/server/auth/（pristine clone go build 必败，建议新 issue，修复超出本轨红线）；(3) 4 截图待外化时上传 issue 评论；(4) sign-in 冒烟历史性偶发抖动（非本轨引入）；(5) customers 冒烟行为由环境性恒败转确定性通过（净改进，CI 可见变化）。
- 全部 Ruling 索引：走查模式（领取台账）/ 每任务一提交 / README 括注保留 / namespaces mock 扩展 customers / go build 主检出验证+缺陷不本轨修。

## Externalization: DONE (2026-08-29 20:3x–20:5x +08:00)

- 用户在对话中回复「**批准外发 #29**」（凭上轮 wake-log 等待事项与批准消费协议执行；批准即接受 stateful-mock 走查证据等级——等待事项 2 一并消费）。
- 执行链（运行锁 acquiredAt=2026-08-29T20:32:32+08:00）：
  1. 外化前新鲜核验全过（worktree 干净 tip=93e7998f1、main=origin/main=601fe0b6e、#29 OPEN 无新评论、分支未 push）。
  2. push codex/admin-config-29 → origin OK（new branch，93e7998f1，ls-remote 复核命中）。
  3. 主检出 merge --no-ff → **313b1f50b**（零冲突；+3 文件 148 行：计划文档/README/冒烟用例）。
  4. 合并后门禁三连全绿：pnpm build exit 0（573ms）/ pnpm lint 0 errors（2 条既有 form.watch 信息性 warning，与基线一致）/ pnpm test:e2e **3 passed**（sign-in ✓ customers ✓（本轨转确定性绿）config plans ✓（本轨新增用例））。
  5. 外化 docs 提交（本工作区工件含 4 截图强制入库 + wake-log），main 推送 fork。
  6. #29 附证据评论（4 截图经仓库 raw URL 嵌入；uploads.github.com 两端点均 422 Bad Size 不可用，已记录）+ gh issue close 29 --reason completed。
  7. .gitignore:36 预存缺陷新 issue 建档（等待事项 3，随批准一并消费）。
- 轨道终态：**EXTERNALIZED & CLOSED**。
