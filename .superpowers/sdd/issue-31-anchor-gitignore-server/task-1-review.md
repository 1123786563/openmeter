# Task 1 Review — 2026-08-29（Downgrade 模式：控制器执行，双遍分审）

Diff: review-33921bc41..b40d256c7.diff（1 commit，.gitignore 修改 + 6 新增文件，+1389/-1）

## 第一遍：Spec 合规（对照 task-1-brief.md 逐条）

| Brief 要求 | 证据 | 判定 |
|---|---|---|
| .gitignore 行 36 `server`→`/server` + 注释说明锚定原因 | diff：`-server` `+/server`，3 行注释（root binary only / bare pattern 后果 / issue #31 引用）；check-ignore 实测 `.gitignore:39:/server server` | ✅ |
| 复制四路径（6 文件）自主检出 | cmp 逐文件 0 差（6/6 identical） | ✅ |
| git add + commit 引用 issue #31 | b40d256c7 消息含 "issue #31"，--stat 恰为 .gitignore+6 文件 | ✅ |
| check-ignore 四路径不再命中 | 4×「not ignored ✓」（exit 1 语义） | ✅ |
| 根 `server` 二进制仍被忽略 | worktree 新规则实测 `.gitignore:39:/server server`；嵌套 `openmeter/server` 不再被忽略 ✓ | ✅ |
| ls-files auth 恰 3 文件 | auth.go/auth_test.go/jwks.go 恰 3 行 | ✅ |
| ls-files 其余 3 文件 | 3/3 列出 | ✅ |
| worktree `go build ./...` exit 0 | 红基线 exit 1（与 Issue 逐字同错）→ 修复后 exit 0 | ✅ |
| scoped `go vet` exit 0 | ./openmeter/server/... ./api/v3/server/... ./cmd/server/... exit 0 | ✅ |

**勘误（plan 文本 vs 实际）**：计划「测试与验收命令」节写「.gitignore + 7 个纯新增文件」，实为 **.gitignore + 6 个新增文件**（auth 3 + loopback 1 + 两测试 2）。纯计数笔误，不影响任何验收语义——Ruling 记台账。

**Spec 结论：✅ 全部满足，无缺失，无超范围改动。**

## 第二遍：代码质量（对恢复代码的实质审读——该 6 文件历史上从未入 git，即从未被任何评审看过）

- `cmd/server/commerce_loopback.go`（43 行，生产 package main）：测试台专用 collaborators；门禁 `cfg.Test.Enabled` 不满足即返回 nil（fail-closed）；注释明确 "must not be used for production"。与 `main.go:178` 调用点契约吻合（`commerceRuntimeDependencies` 结构）。无缺陷。
- `openmeter/server/auth/auth.go`（411 行，生产，安全敏感）：RS256-only parser、ExpirationRequired、静态响应体防内部错误泄漏（reject() 双错误设计）、org allowlist + 两级角色、`Validate()` 遵循 AGENTS.md 的 errors.Join+NewNillableGenericValidationError 惯例、无 panic、无 context.Background/TODO、命名 helper 非 closure。公开路径豁免表有逐条理由注释（portal meters 前缀豁免有 PortalTokenAuth 说明）。无缺陷。
- `openmeter/server/auth/jwks.go`（224 行）：singleflight 防惊群、失败也推进节流时间戳（防打爆不可达端点）、`context.WithoutCancel(ctx)`（传播保留式脱离，符合 AGENTS.md context 纪律）、LimitReader 1MiB 封顶、x5c 回退解析、不可用键跳过而非整集失败。无缺陷。
- `openmeter/server/auth/auth_test.go`（550 行）：表驱动 middleware 行为测试（状态码+context 身份断言）、JWKS 轮换测试、节流失败刷新单次请求断言、x5c 解析、NewOptional 三态。断言真实行为非 mock 行为。18 处 require。无空断言。
- `openmeter/server/cors_test.go`（116 行）：正/负/通配/portal 旁路五场景，含「不允许 origin 无 allow-origin 头」负路径与「不带 credentials」安全断言。无空断言。
- `api/v3/server/namespaces_test.go`（41 行）：路由行为 + 配置校验负路径两测。无空断言。

**质量结论：Approved。** 恢复代码为已在主检出长期验证的成品（字节级还原是计划的硬约束），本次零逻辑改动；审读未发现 Critical/Important 缺陷。

## 发现汇总

- Critical: 0；Important: 0；Minor: 1（计划文本「7 个纯新增文件」计数笔误→实 6，已记台账 Ruling-3）。
- ⚠️ Cannot verify from diff: 无（全部验收在 diff 与命令输出内闭环）。

**Verdict: PASS（spec ✅ + quality approved）**
