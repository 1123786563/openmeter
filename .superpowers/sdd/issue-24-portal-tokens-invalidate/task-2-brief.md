# Task 2 brief — issue #24 T2 令牌列表 + 失效确认 + i18n

Worktree: /Users/wuyongjun/trea/openmeter-issue-24（分支 codex/admin-config-24）
T1 已提供（勿改）：`usePortalTokens(limit=100)`（data 为 `PortalToken[]` **裸数组**）、
`useInvalidatePortalToken()`（mutation，body `{id?|subject?}`）。

## 必读参考（worktree 内先读再写）

- `docs/superpowers/plans/issue-24-portal-tokens-invalidate.md`
- `web/src/features/config/portal-tokens/index.tsx`（挂载点，现为仅发放按钮）
- `web/src/features/config/apps/index.tsx`（列表模式：Table/Badge/Skeleton/
  ConfirmDialog/toast/handleServerError/行操作按钮）
- `web/src/features/config/portal-tokens/issue-token-dialog.tsx`（发放流，勿改）
- `web/src/lib/format.ts` 的 `formatDateTime`

## 交付物

1. 修改 `web/src/features/config/portal-tokens/index.tsx`：
   - `const { data: tokens, isLoading } = usePortalTokens()`。
   - PageHeader 下方列表区（mt-6，标题 config.portalTokens.list.title）：Table 列
     subject（font-medium，可 font-mono）/ 创建时间（formatDateTime，缺省 '—'）/
     允许的 meters（Badge 列表；空数组或不传 → config.portalTokens.list.unrestricted
     文案徽章）/ 状态（`token.expired` 为 true → Badge outline 灰色
     config.portalTokens.status.expired；否则 '—'）/ 操作（失效按钮，destructive ghost
     sm；`token.expired` 为 true 时禁用）。
   - loading Skeleton；空态行（colSpan=5，config.portalTokens.list.empty）。
   - 失效走 ConfirmDialog（destructive）：title=config.portalTokens.invalidateConfirm.title，
     desc 插值 subject；确认 → `invalidateMutation.mutate({ id: target.id }, …)`；
     onSuccess toast（config.portalTokens.toast.invalidated）+ 关闭；onError
     handleServerError（原文透出）。
   - 发放按钮/IssueTokenDialog 既有逻辑零触碰。
   - **列表任何位置不得渲染 token.token 明文**（Issue 明示列表不含明文；类型上列表
     响应也无该字段——防御性：不引用该字段）。
2. i18n：`config.portalTokens` 子树内**只追加**：
   - `list: { title, empty, fields: { subject, createdAt, meters, status, actions },
     unrestricted }`
   - `invalidateConfirm: { title, description }`（description 含 {{subject}} 插值）
   - `status: { expired }`
   - `toast: { invalidated }`
   zh/en 同构、en 英文文案；不改既有键。

## 约束

- 只动上述 2 个文件；不碰 legacy/hooks/query-keys/发放流。
- 失效是破坏性操作，必须 ConfirmDialog；错误原文 toast。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-24/web && pnpm build && pnpm lint
```

双 exit 0；无 routeTree 改动；locale 新子树 zh/en 键一致。

## 提交

```
git add src/features/config/portal-tokens/index.tsx src/i18n/locales/zh-CN.ts src/i18n/locales/en.ts
git commit -m "feat(admin): 门户令牌列表与失效 (issue #24)"
```

## 报告

完整报告写入 `.superpowers/sdd/issue-24-portal-tokens-invalidate/task-2-report.md`，
回复只给四行：状态/提交哈希/一行测试摘要/疑虑。
