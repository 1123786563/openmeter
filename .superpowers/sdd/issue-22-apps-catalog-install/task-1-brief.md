# Task 1 brief — issue #22 T1 API 层

Worktree: /Users/wuyongjun/trea/openmeter-issue-22（分支 codex/admin-config-22，base f60cb90b0）

## 要求（逐字来自计划 docs/superpowers/plans/issue-22-apps-catalog-install.md T1 + 契约事实节）

1. `web/src/api/query-keys.ts`：追加 `appCatalog: () => ns('app-catalog'),`（位置放在 apps 键旁）。
2. `web/src/api/hooks.ts` Apps (config) 段内追加两个 hook（模式照抄同段 useApps/useUninstallApp）：

```ts
export function useAppCatalog() {
  return useQuery({
    queryKey: queryKeys.appCatalog(),
    queryFn: ({ signal }) => api.internal.apps.listCatalog(undefined, { signal }),
  })
}

export function useInstallApp() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: Parameters<typeof api.internal.apps.install>[0]) =>
      api.internal.apps.install(body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('apps') })
    },
  })
}
```

注意：
- `listCatalog(request?: ListAppCatalogRequest, ...)` 第一参数可省；若 TS 报
  undefined 不可传则改为 `api.internal.apps.listCatalog({ signal })` 形态中
  与 SDK 签名匹配的写法（以 tsc 通过且语义一致为准）。
- install 的 onSuccess 失效 `nsPrefix('apps')`（已装列表刷新是 Issue 验收项）。
- 不得改动段内既有 hook；不得触碰其他文件。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-22/web && pnpm build && pnpm lint
```

预期双双 exit 0（build 含 tsr generate + tsc -b + vite build；routeTree 零 diff——
`git status` 不得出现 routeTree.gen.ts 修改）。

## 提交

```
git add src/api/query-keys.ts src/api/hooks.ts
git commit -m "feat(admin): 应用目录与安装 API hooks (issue #22 T1)"
```

## 报告

把完整报告写入
`.superpowers/sdd/issue-22-apps-catalog-install/task-1-report.md`（做了什么、
验证命令与真实输出摘要、提交哈希、疑虑），回复只给：状态（DONE /
DONE_WITH_CONCERNS / NEEDS_CONTEXT / BLOCKED）、提交哈希、一行测试摘要、疑虑。
