# Task 1 Brief — 四处测试文件 lint 修复（发现 1–4）

Worktree: /Users/wuyongjun/trea/openmeter-issue-32（分支 codex/admin-config-32，BASE a5fc38788）
Issue: https://github.com/1123786563/openmeter/issues/32（来源：#31 Ruling-4 parked）
计划: docs/superpowers/plans/issue-32-lint-cleanup.md（本 brief 为其任务 1 全文，值一律以此为准）

## 修复 1 — gci（openmeter/subscription/service/servicevalidation_test.go）

import 块现为一个混合组。按 .golangci.yaml gci 配置（standard / default / prefix(github.com/openmeterio/openmeter)）重排为三组（standard 组此处只有 testing，单独一组；alpacahq+testify 为 default 组；四个 openmeterio 包为 prefix 组，组间空行）。修后以 `golangci-lint fmt --diff openmeter/subscription/service/servicevalidation_test.go` 输出为空复核。

## 修复 2 — ineffassign（openmeter/commerce/order/service_test.go:810 附近）

现状（orders 赋值后未读取）：

```go
created := commerce.OrderStatusCreated
orders, total, err = svc.ListOrders(t.Context(), commerce.ListOrdersInput{Namespace: "ns", Status: &created})
if err != nil {
    t.Fatal(err)
}
if total != 2 {
    t.Fatalf("created-status total = %d, want 2", total)
}
```

改为补 len 断言（与同函数兄弟用例 unfiltered/cust-1 双值断言风格一致）：

```go
if total != 2 || len(orders) != 2 {
    t.Fatalf("created-status list = %d items, total %d; want 2/2", len(orders), total)
}
```

禁止改为 `_`（Issue 明令不得静默丢弃本应验证的结果）。若补断言后测试失败：停下，诊断，报告——不得回退或弱化。

## 修复 3 — ineffassign（openmeter/commerce/refund/service_test.go:2434 附近）

同型（refunds 未读取）：

```go
pending := RefundStatusPendingFence
refunds, total, err = h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns", Status: &pending})
if err != nil {
    t.Fatal(err)
}
if total != 2 {
    t.Fatalf("pending-fence total = %d, want 2", total)
}
```

改为：

```go
if total != 2 || len(refunds) != 2 {
    t.Fatalf("pending-fence list = %d items, total %d; want 2/2", len(refunds), total)
}
```

同样禁止 `_`；失败即停诊断。

## 修复 4 — sloglint（openmeter/server/auth/auth_test.go:110）

`return slog.New(slog.NewTextHandler(io.Discard, nil))` → `return slog.New(slog.DiscardHandler)`。
`io` 在该文件仅此一处使用（已 grep 证实）→ 同时移除 `io` import。函数签名与丢弃型 logger 用途不变。

## 每处修完后定向验证

```bash
cd /Users/wuyongjun/trea/openmeter-issue-32
go test ./openmeter/subscription/service/... 2>&1 | tail -3
go test ./openmeter/commerce/order/... 2>&1 | tail -3
go test ./openmeter/commerce/refund/... 2>&1 | tail -3
go test ./openmeter/server/auth/... 2>&1 | tail -3
golangci-lint run ./openmeter/subscription/service/... ./openmeter/commerce/order/... ./openmeter/commerce/refund/... ./openmeter/server/auth/... 2>&1 | tail -5
```

## 提交

单 commit，message 例：`fix(lint): clear gci/ineffassign/sloglint findings in test files (#32)`。
不触碰本 brief 之外的任何文件（含同文件其他行；io import 移除是修复 4 的必要伴随）。

## 报告契约

写全量报告到 /Users/wuyongjun/trea/openmeter/.superpowers/sdd/issue-32-lint-cleanup/task-1-report.md（含每条命令与真实输出）。回复仅：状态（DONE/DONE_WITH_CONCERNS/NEEDS_CONTEXT/BLOCKED）、commit hash、一行测试摘要、concerns（如有）。
