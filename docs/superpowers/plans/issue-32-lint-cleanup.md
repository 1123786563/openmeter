# Issue #32 实施计划 — lint 清理：清零 make lint-go 5 项存量发现

- Issue: https://github.com/1123786563/openmeter/issues/32
- 分支: `codex/admin-config-32`（base = main @ 791c21745）
- Worktree: `/Users/wuyongjun/trea/openmeter-issue-32`
- 来源轨道: #31 Ruling-4 parked（`.superpowers/sdd/issue-31-anchor-gitignore-server/progress.md`）

## 范围（5 项发现，逐条精确修复）

1. **gci — `openmeter/subscription/service/servicevalidation_test.go`**
   import 块未按 `.golangci.yaml` gci 分组（standard / default / prefix(github.com/openmeterio/openmeter)）。
   修复：将 `github.com/alpacahq/alpacadecimal` 与 `github.com/stretchr/testify/require` 归入 default 组，
   openmeterio 四个包归入 prefix 组（组间空行分隔）。以 `golangci-lint fmt --diff` 精确 diff 为准。

2. **ineffassign — `openmeter/commerce/order/service_test.go:810`**
   `orders, total, err = svc.ListOrders(..., Status: &created)` 中 `orders` 赋值后未被读取。
   测试意图判定：同函数兄弟用例（unfiltered/cust-1/组合过滤/分页）均断言 `len+total` 双值，
   本用例（created-status 过滤）应同样断言查询结果。修复：断言改为
   `if total != 2 || len(orders) != 2 { t.Fatalf("created-status list = %d items, total %d; want 2/2", ...) }`。
   不得改为 `_` 静默丢弃（Issue 明令禁止）。

3. **ineffassign — `openmeter/commerce/refund/service_test.go:2434`**
   同型：`refunds, total, err = h.svc.ListRefunds(..., Status: &pending)` 中 `refunds` 未读取。
   修复：断言改为 `if total != 2 || len(refunds) != 2 { t.Fatalf("pending-fence list = %d items, total %d; want 2/2", ...) }`。

4. **sloglint — `openmeter/server/auth/auth_test.go:110`**
   `return slog.New(slog.NewTextHandler(io.Discard, nil))` → `return slog.New(slog.DiscardHandler)`。
   `io` 在该文件仅此一处使用 → 同时移除 `io` import。函数签名与丢弃型 logger 用途不变。

5. **staticcheck QF1003 — `cmd/server/commerce.go:1078`（生产代码）**
   `if in.Source == commerce.BucketSourceRecharge { ... } else if in.Source == commerce.BucketSourcePlan { ... }`
   （无 else 分支）→ tagged switch：
   ```go
   switch in.Source {
   case commerce.BucketSourceRecharge:
       input.Priority = 30
   case commerce.BucketSourcePlan:
       input.Priority = 10
   }
   ```
   保留前导注释 `// Wallet bucket priority: recharge=30, plan=10 (see commerce.SourcePriority).`
   逐分支语义等价：两 case 均不命中时 Priority 保持结构体字面量中的 1（与现 if 链无 else 分支一致）。

## 非目标

- 不重新启用 `.golangci.yaml` FIXME 注释掉的 linters
- 不做任何 lint 配置/规则调整
- 不顺手重构 5 项发现之外的代码（含同文件其他行）

## 任务拆分

- **Task 1**（四处测试文件修复，发现 1–4）：gci 分组 / 两处 ineffassign 补断言 / sloglint+io 移除。
  每处修完后运行所在包定向测试确认绿。
- **Task 2**（生产代码，发现 5）：tagged switch 转换。修后 `go build ./cmd/server/...`、
  `go vet ./cmd/server/...`，并提交逐分支等价性说明（注释保留、无 else 语义保持）。
- **Task 3**（全量门禁验收）：依赖 T1+T2。命令见下节，全部第一手真实输出为证。

## 测试与验收命令

- `make lint-go` → exit 0（Makefile 定义：根模块 golangci-lint + api/v3/client + e2e vet + e2e golangci-lint）
- `go build ./...`（根模块）、`go vet ./...`（根模块）
- 受影响包测试：
  `go test ./openmeter/subscription/service/... ./openmeter/commerce/order/... ./openmeter/commerce/refund/... ./openmeter/server/auth/... ./cmd/server/...`
- 修复前红基线：`make lint-go` exit 2 恰 5 发现（#32 Issue 正文既档，#31 台账同源）。
- 零行为变化证据（commerce.go）：diff 仅控制流转换；`cmd/server` 无测试面则以 build+vet+逐分支人工比对为证。

## 全局约束

- 遵守仓库 AGENTS.md（Go conventions / Testing conventions）。
- 最小 diff：只动 5 处发现点及其直接伴随（io import、断言行）；不触碰无关文件。
- 测试断言补强不得改变被测语义（len==total==2 为无分页查询的既定事实，源自现有 total 断言）。
- 发现 2/3 若补断言后测试失败，视为暴露真实缺陷：停下诊断，不得回退为 `_` 或弱化断言来「修绿」。
