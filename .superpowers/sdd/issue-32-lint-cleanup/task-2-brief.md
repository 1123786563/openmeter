# Task 2 Brief — 生产代码 tagged switch 转换（发现 5）

Worktree: /Users/wuyongjun/trea/openmeter-issue-32（分支 codex/admin-config-32，Task 1 后 HEAD c70437a88）
Issue: https://github.com/1123786563/openmeter/issues/32 发现 5 | 计划: docs/superpowers/plans/issue-32-lint-cleanup.md

## 修复 — staticcheck QF1003（cmd/server/commerce.go:1077-1082）

现状：

```go
// Wallet bucket priority: recharge=30, plan=10 (see commerce.SourcePriority).
if in.Source == commerce.BucketSourceRecharge {
	input.Priority = 30
} else if in.Source == commerce.BucketSourcePlan {
	input.Priority = 10
}
```

改为 tagged switch（无 default——与原 if 链无 else 分支语义等价：Source 为其他值时 Priority 保持结构体字面量中的 1）：

```go
// Wallet bucket priority: recharge=30, plan=10 (see commerce.SourcePriority).
switch in.Source {
case commerce.BucketSourceRecharge:
	input.Priority = 30
case commerce.BucketSourcePlan:
	input.Priority = 10
}
```

硬约束：
- 前导注释原样保留。
- 逐分支语义等价：Recharge→30、Plan→10、其他→不改写（保持 1）。若 in.Source 类型非离散枚举（可同时命中多值），停止并报告——不得在未证实等价前提交。
- 仅此一处改动，不触碰文件其他行。

## 验证

```bash
cd /Users/wuyongjun/trea/openmeter-issue-32
go build ./cmd/server/... && go vet ./cmd/server/...
golangci-lint run ./cmd/server/...
```

三者全过（lint 修复前同命令恰 1 发现 QF1003）。

## 提交

单 commit，message 例：`refactor(lint): tagged switch on in.Source in commerce grant priority (#32)`

## 报告契约

全量报告写 /Users/wuyongjun/trea/openmeter/.superpowers/sdd/issue-32-lint-cleanup/task-2-report.md（含命令+真实输出+逐分支等价性说明）。回复仅：状态、commit、一行测试摘要、concerns。
