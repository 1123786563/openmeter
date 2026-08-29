### Task 2 — pristine 验收验证（Issue 验收标准逐条）

1. 从分支 tip 新建**临时** worktree（`git worktree add <tmp> codex/admin-config-31`， detached 或新临时目录均可，用后 `git worktree remove` 清理）——文件只能来自 git，排除「磁盘副本干扰」。
2. 在临时 worktree 内运行并记录：
   - `go build ./...` exit 0（Issue 验收第 2 条）；
   - `go vet ./...` exit 0（根模块全量，覆盖全部恢复的测试文件类型检查；e2e 是独立模块不在根 `./...` 内，符合 AGENTS.md）。
3. 主检出（用户工作机）`go build ./...` 仍 exit 0（Issue 验收第 3 条；锚定规则不删磁盘文件，主检出磁盘态不变，直接复跑证实）。
4. 把验证输出写入报告文件，追加台账。

## 测试与验收命令（汇总）

| # | 命令 | 期望 |
|---|------|------|
| A1 | `git ls-files openmeter/server/auth` | 非空，恰 3 文件 |
| A2 | `git ls-files cmd/server/commerce_loopback.go openmeter/server/cors_test.go api/v3/server/namespaces_test.go` | 3 行全列出 |
| A3 | `git check-ignore -v openmeter/server/auth/auth.go cmd/server/commerce_loopback.go openmeter/server/cors_test.go api/v3/server/namespaces_test.go` | 无输出（exit 1） |
| A4 | `git check-ignore -v server` | 命中 `.gitignore:/server` |
| B1 | 临时 pristine worktree：`go build ./...` | exit 0 |
| B2 | 临时 pristine worktree：`go vet ./...` | exit 0 |
| B3 | 主检出：`go build ./...` | exit 0（磁盘态不变的回归证实） |

不运行前端/e2e 套件：本修复零前端/零行为变更（`git show --stat` 将只含 .gitignore + 7 个纯新增文件）。

## 全局约束

- 逐字节恢复：被恢复文件内容与主检出磁盘版本完全一致，禁止顺手修改（格式化留给后续 lint 轮）。
- 单 commit 或少量 commit 均可，但必须全部落在 `codex/admin-config-31`，不落 main。
- 遵守 AGENTS.md：不用 `panic`、不引入 `context.Background()` 等——本修复理论上不触碰任何逻辑代码。
- 不 push、不 merge、不动 GitHub 状态（发布判断属第 7 步）。
- 主检出的四个磁盘文件保持原样（外部化时合并会与未跟踪副本冲突，预案：merge 前 `rm` 主检出的未跟踪副本——内容已逐字节入 git，无损失；该预案记入台账供外化轮消费）。

## 外部化预案（第 7 步消费，不在本轮执行）

1. push `codex/admin-config-31`；
2. 外化合并前先 `rm` 主检出四个未跟踪路径的磁盘副本（与分支内容逐字节相同，避免 untracked-overwrite 拒绝合并）；
3. `git merge --no-ff codex/admin-config-31` 进 main → push fork；
4. main 上复跑 `go build ./...`（此时文件来自 git）+ `git ls-files` 复核；
5. Issue #31 附证据评论并关闭。
