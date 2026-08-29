# Task 1 brief — issue #28 T1 API 层

Worktree: /Users/wuyongjun/trea/openmeter-issue-28（分支 codex/admin-config-28，base f60cb90b0）

## 要求（逐字来自计划 docs/superpowers/plans/issue-28-offline-payment.md T1 + 契约事实节）

1. `web/src/api/hooks.ts` 的 Receivable periods 段（useUpdateExternalInvoice 之后）追加：

```ts
export function useCreateOfflinePayment(customerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (
      body: Parameters<typeof api.commerce.createOfflinePayment>[0]['body']
    ) => api.commerce.createOfflinePayment({ customerId, body }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.receivablePeriods(customerId),
      })
    },
  })
}
```

注意：失效 `queryKeys.receivablePeriods(customerId)`（前缀失效，含所有 after 游标页）
——周期已付金额刷新是 Issue 验收项；不得改动段内既有 hook；不得触碰其他文件。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-28/web && pnpm build && pnpm lint
```

预期双双 exit 0；`git status` 不得出现 routeTree.gen.ts 修改。

## 提交

```
git add src/api/hooks.ts
git commit -m "feat(admin): 线下支付登记 API hook (issue #28 T1)"
```

## 报告

完整报告写入 `.superpowers/sdd/issue-28-offline-payment/task-1-report.md`，
回复只给：状态（DONE/DONE_WITH_CONCERNS/NEEDS_CONTEXT/BLOCKED）、提交哈希、
一行测试摘要、疑虑。
