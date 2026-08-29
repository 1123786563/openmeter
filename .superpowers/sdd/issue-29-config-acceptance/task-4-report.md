# Task 4 Report — 全量回归

- 状态：DONE_WITH_PREEXISTING_FINDING（前端三连全绿；Go 回归在主检出全绿+分支零 Go diff；发现 main 预存打包缺陷，详见下）

## 命令与结果（worktree = /Users/wuyongjun/trea/openmeter-issue-29）

| 命令 | 结果 |
| --- | --- |
| `cd web && pnpm build` | ✓ built（exit 0，产物 219KB 主包） |
| `cd web && pnpm lint` | 0 errors, 2 warnings（均为 main 既有的 form.watch 信息性 warning：rule-form-dialog.tsx:172 / plan-addon-form-dialog.tsx:97，非本分支引入） |
| `cd web && pnpm test:e2e` | **3 passed (6.2s)**（sign-in ✓ customers ✓ config plans ✓）。备注：T4 首跑 sign-in 单次失败（mock IdP/preview webServer 复用时序抖动），隔离重跑 ✓、全量重跑 3/3 ✓——与既有台账「sign-in 偶发抖动（跨轨共性）」同模式，非本分支回归 |
| `git status --porcelain`（worktree） | 干净（走查工件已归档 SDD 工作区并从 e2e/ 移除） |
| `git diff --stat main...HEAD -- '*.go'` | **空**（零后端改动） |

## Go 回归与环境裁定（Ruling）

- 处方步骤 4 命令：仓库根 `go build ./...`。
- **主检出（/Users/wuyongjun/trea/openmeter，HEAD=601fe0b6e=分支 base）**：`go build ./...` → **exit 0 无输出** ✓。
- **issue-29 worktree（同一提交的干净检出）**：失败——`cmd/server/main.go:31:2: no required module provides package .../openmeter/server/auth`。
- 根因（**main 预存缺陷，与本分支无关**）：`.gitignore:36` 的无锚定规则 `server` 匹配任意深度的 server 目录，把 fork 提交 925f6be4d（feat(admin) 管理端全量交付）新增并引用的 `openmeter/server/auth/`（auth.go/jwks.go/auth_test.go）整体吞出版本控制——`git ls-files` 为空、`git status --ignored` 显示 `!!`。该包只存在于用户主检出磁盘（ignored 本地文件），故主检出自建可过，任何新克隆/worktree 必然编译失败。
- 裁定：本轨验收结论=「系列零后端误改」（零 Go diff + 主检出构建绿）；pristine-clone 构建失败记为 **main 预存打包缺陷**，修复属仓库结构/后端变更，超出 #29「无后端改动」红线，不在本轨修。建议后续 issue：修 .gitignore 规则（`server` → 锚定或删除）并补提交 `openmeter/server/auth/` 三文件（草稿文案已备，随外化批准一并提请）。

## 验收标准映射

| issue 验收标准 | 证据 |
| --- | --- |
| e2e 冒烟覆盖至少一个新配置页 | smoke 第 3 用例（/config/plans）+ 3 passed |
| 四大块各完成一次写操作并留证 | T3 报告四块表格 + 4 截图 + evidence JSON |
| 全量回归绿 | 前端三连全绿 + go build exit 0（主检出）+ 零 Go diff |
