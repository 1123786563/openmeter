# Task 1 brief — issue #26 T1 API 层

Worktree: /Users/wuyongjun/trea/openmeter-issue-26（分支 codex/admin-config-26，base f60cb90b0）

## 要求（逐字来自计划 docs/superpowers/plans/issue-26-billing-profiles-edit-delete.md T1 + 契约事实节）

1. `web/src/api/hooks.ts` 的 Billing profiles 段（useCreateBillingProfile 之后）追加：

```ts
export function useUpdateBillingProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.billing.updateProfile>[0]) =>
      api.billing.updateProfile(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('billing-profiles'),
      })
    },
  })
}

export function useDeleteBillingProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id }: { id: string }) => api.billing.deleteProfile({ id }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('billing-profiles'),
      })
    },
  })
}
```

注意：updateProfile body 类型由 SDK 提供（`UpsertBillingProfileRequestInput`，
不含 apps）——T1 不构造 body，只透传 input；不得改动段内既有 hook；
不得触碰其他文件（含 query-keys.ts）。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-26/web && pnpm build && pnpm lint
```

预期双双 exit 0；`git status` 不得出现 routeTree.gen.ts 修改。

## 提交

```
git add src/api/hooks.ts
git commit -m "feat(admin): 账单档案更新与删除 API hooks (issue #26 T1)"
```

## 报告

完整报告写入 `.superpowers/sdd/issue-26-billing-profiles-edit-delete/task-1-report.md`，
回复只给：状态（DONE/DONE_WITH_CONCERNS/NEEDS_CONTEXT/BLOCKED）、提交哈希、
一行测试摘要、疑虑。
