# Issue #31 — 锚定 .gitignore 无锚定 `server` 规则，恢复被吞的四个源码/测试路径

- **Issue**: https://github.com/1123786563/openmeter/issues/31
- **分支**: `codex/admin-config-31`（worktree: `/Users/wuyongjun/trea/openmeter-issue-31`，base = main `158251056`）
- **类型**: bug / 仓库结构修复（无产品行为变更）

## 背景与根因（已第一手核实）

`.gitignore:36` 存在无锚定规则 `server`（位于 `# CodeGraph` 注释块内，commit `788a53dfc` 引入，本意是忽略仓库根目录的 CodeGraph `server` 二进制，磁盘实测 229,750,194 字节）。无锚定模式匹配**任意深度**名为 `server` 的文件/目录，把以下四个真实源码/测试路径吞出版本控制（`git check-ignore -v` 逐一证实均命中 `.gitignore:36:server`；`git log --all` 证实四者**从未被任何提交跟踪**，纯主检出磁盘文件）：

1. `openmeter/server/auth/`（`auth.go` 11,757B / `jwks.go` 5,866B / `auth_test.go` 16,202B）——被 `cmd/server/main.go:31` 生产导入；
2. `cmd/server/commerce_loopback.go`（`package main`，无构建标签）——定义 `loopbackTestRuntimeDependencies`，被 `cmd/server/main.go:178` 调用；
3. `openmeter/server/cors_test.go`——已跟踪 `cors.go` 的测试；
4. `api/v3/server/namespaces_test.go`——`api/v3/server` 包测试。

后果：pristine clone / 新 worktree / CI fresh checkout 上 `go build ./...` 必败。本 worktree 基线复现（exit 1，错误与 Issue 引用逐字一致）：

```
cmd/server/main.go:31:2: no required module provides package github.com/openmeterio/openmeter/openmeter/server/auth
```

**范围裁决（Ruling，入台账）**：Issue 正文「修复方向」第 2 条只点名 `openmeter/server/auth/` 三个文件，但其验收标准「新建临时 worktree 执行 `go build ./...` exit 0」**必须**同时恢复 `cmd/server/commerce_loopback.go`（否则 `cmd/server` 包缺 `loopbackTestRuntimeDependencies` 符号，build 仍败）。两个测试文件（`cors_test.go`、`namespaces_test.go`）虽不阻塞 build，但属同一根因的受害面，且恢复后 pristine clone 的测试覆盖才完整（`go vet ./...` 会类型检查它们）。故本计划恢复**全部四个路径**——这是对 Issue 验收标准的最小完整满足，不是范围扩张。

## 范围

1. `.gitignore:36`：`server` → `/server`（锚定到仓库根，继续忽略 CodeGraph 根二进制），并在该行补一行注释说明锚定原因。
2. `git add` 四个被吞路径（从主检出复制内容到 worktree，因 worktree 中它们不存在）：
   - `openmeter/server/auth/auth.go`、`openmeter/server/auth/jwks.go`、`openmeter/server/auth/auth_test.go`
   - `cmd/server/commerce_loopback.go`
   - `openmeter/server/cors_test.go`
   - `api/v3/server/namespaces_test.go`
3. 提交到分支 `codex/admin-config-31`。

## 非目标

- 不改动任何被恢复文件的内容（逐字节复制主检出磁盘版本，不做格式化/重构/润色）。
- 不删除主检出磁盘上的任何文件（外部化合并时的未跟踪副本处理属发布步骤预案，见下）。
- 不处理根目录 CodeGraph `server` 二进制本身（继续被 `/server` 规则忽略）。
- 不做任何 Go 代码逻辑变更、不做 API/SDK/前端变更。
- 不在本轮执行 push / merge / 关闭 Issue（第 7 步发布判断另行处理）。

## 任务拆分

### Task 1 — 锚定规则并恢复四个路径

1. 编辑 `.gitignore`：行 36 `server` → `/server`，行内或行上补注释 `# CodeGraph server binary at repo root only (anchored: nested server dirs must stay trackable)`。
2. 从主检出复制四个路径的文件到 worktree 相同相对路径（源：`/Users/wuyongjun/trea/openmeter/{openmeter/server/auth,cmd/server,openmeter/server,api/v3/server}/...`）。
3. `git add .gitignore` + 四个路径；提交（信息含 issue #31 引用）。
4. 任务级验证（全部必须通过）：
   - `git check-ignore -v <四个路径中每个文件>` 逐个**不再命中**（exit 1）；
   - `git check-ignore -v server` 仍命中 `/server` 规则（根二进制继续被忽略）；
   - `git ls-files openmeter/server/auth` 非空且恰含 3 文件；`git ls-files cmd/server/commerce_loopback.go openmeter/server/cors_test.go api/v3/server/namespaces_test.go` 全部列出；
   - 复制后文件与主检出逐字节一致：`cmp` 逐个 0 差；
   - worktree 内 `go build ./...` exit 0（红色基线转绿）；
   - worktree 内 `go vet ./openmeter/server/... ./api/v3/server/... ./cmd/server/...` exit 0。

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
