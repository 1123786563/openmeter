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

