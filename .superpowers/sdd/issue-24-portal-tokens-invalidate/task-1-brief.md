# Task 1 brief — issue #24 T1 API 层

Worktree: /Users/wuyongjun/trea/openmeter-issue-24（分支 codex/admin-config-24，base f60cb90b0）

## 要求（逐字来自计划 docs/superpowers/plans/issue-24-portal-tokens-invalidate.md T1 + 契约事实节）

1. `web/src/api/legacy.ts` 的 Portal tokens 段（文件末尾 createPortalToken 之后）追加：

```ts
/** GET /api/v1/portal/tokens — 列表（spec 为裸数组，无分页包装）。limit 1-100。 */
export async function listPortalTokens(limit = 100): Promise<PortalToken[]> {
  return apiFetch<PortalToken[]>(`/v1/portal/tokens?limit=${limit}`)
}

/**
 * POST /api/v1/portal/tokens/invalidate — 按 id 或 subject 失效（二选一），
 * 204 无内容。管理端按行 id 失效。
 */
export async function invalidatePortalTokens(body: {
  id?: string
  subject?: string
}): Promise<void> {
  return apiFetch<void>('/v1/portal/tokens/invalidate', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}
```

2. `web/src/api/hooks.ts` 的 Portal tokens 段（useCreatePortalToken 之后）追加，
并把 `listPortalTokens, invalidatePortalTokens` 加入文件顶部既有的 legacy 导入：

```ts
export function usePortalTokens(limit = 100) {
  return useQuery({
    queryKey: queryKeys.portalTokens({ limit }),
    queryFn: () => listPortalTokens(limit),
  })
}

export function useInvalidatePortalToken() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { id?: string; subject?: string }) =>
      invalidatePortalTokens(body),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('portal-tokens'),
      })
    },
  })
}
```

注意：`queryKeys.portalTokens(params)` 已存在（query-keys.ts:56），不得改动
query-keys.ts；不得改动段内既有 hook；不得触碰其他文件。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-24/web && pnpm build && pnpm lint
```

预期双双 exit 0；`git status` 不得出现 routeTree.gen.ts 修改。

## 提交

```
git add src/api/legacy.ts src/api/hooks.ts
git commit -m "feat(admin): 门户令牌列表与失效 API 层 (issue #24 T1)"
```

## 报告

完整报告写入 `.superpowers/sdd/issue-24-portal-tokens-invalidate/task-1-report.md`
（做了什么、验证命令与真实输出摘要、提交哈希、疑虑），回复只给：状态
（DONE/DONE_WITH_CONCERNS/NEEDS_CONTEXT/BLOCKED）、提交哈希、一行测试摘要、疑虑。
