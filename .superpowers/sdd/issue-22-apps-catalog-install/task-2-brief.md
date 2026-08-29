# Task 2 brief — issue #22 T2 目录区 + 安装 dialog + 挂载 + i18n

Worktree: /Users/wuyongjun/trea/openmeter-issue-22（分支 codex/admin-config-22）
T1 已提供（勿改）：`useAppCatalog()`（data 类型 `AppCatalogItemPagePaginatedResponse`，
目录项在 `data.data`）、`useInstallApp()`（mutation，body 即 install 请求）、
`queryKeys.appCatalog()`。

## 必读参考（worktree 内先读再写）

- `docs/superpowers/plans/issue-22-apps-catalog-install.md`（本轨计划：契约事实/非目标）
- `web/src/features/config/apps/index.tsx`（挂载点 + Table/Badge 模式）
- `web/src/features/config/apps/stripe-key-dialog.tsx`（dialog 模式：Form/zod/toast/
  handleServerError/一次性提交）
- `web/src/i18n/locales/zh-CN.ts` / `en.ts` 的 `config.apps` 子树（复用 type.*/capability.*）

## 交付物

1. 新建 `web/src/features/config/apps/install-app-dialog.tsx`：
   - Props `{ open, onOpenChange, item: AppCatalogItem | null }`（类型从 '@openmeter/client' 导入）。
   - 打开时 reset：name 预填 item.name（可改）。
   - zod schema：name min(1)；item.type==='stripe' 时 apiKey min(1)（**Issue 验收项**，
     错误键 config.apps.install.validation.apiKeyRequired）、createBillingProfile
     boolean（Switch 默认 true）。
   - 提交体按 item.type 分支（SDK 契约三分支 createBillingProfile 均必填）：
     stripe → `{ type:'stripe', name, createBillingProfile, apiKey }`；
     sandbox → `{ type:'sandbox', name, createBillingProfile:false }`；
     external_invoicing → `{ type:'external_invoicing', name, createBillingProfile:false }`。
   - `useInstallApp().mutate(body, { onSuccess: toast + 关闭, onError: handleServerError })`。
2. 新建 `web/src/features/config/apps/app-catalog-section.tsx`：
   - `useAppCatalog()`；标题 config.apps.catalog.title；loading Skeleton ×3；空态文案。
   - 每项目一行（Table 或 card 列表，风格对齐已装列表）：type 徽章
     （`t(\`config.apps.type.${item.type}\`)`）、name + description、capabilities 徽章
     （`t(\`config.apps.capability.${c.type}\`)`）、installMethods 徽章（新键
     `config.apps.catalog.installMethod.{with_oauth2,with_api_key,no_credentials_required}`）。
   - 安装按钮：`item.installMethods.includes('with_api_key') ||
     item.installMethods.includes('no_credentials_required')` 可用；否则（仅 with_oauth2
     或空）禁用并在旁边显示 config.apps.catalog.oauthUnsupported 文案（不实现 OAuth，
     计划非目标）。点击 setInstallTarget(item)。
3. 修改 `web/src/features/config/apps/index.tsx`：已装列表区块之后挂载
   `<AppCatalogSection />`（页面注释按需补一句目录区语义）。已装列表/卸载/换 Key
   逻辑零触碰。
4. i18n：`web/src/i18n/locales/zh-CN.ts` 与 `en.ts` 的 `config.apps` 子树内**只追加**：
   - `catalog: { title, empty, installMethod: { with_oauth2, with_api_key,
     no_credentials_required }, installAction, oauthUnsupported }`
   - `install: { title, description, name, apiKey, apiKeyHint, createBillingProfile,
     createBillingProfileHint, validation: { nameRequired, apiKeyRequired }, submit,
     toast: { installed } }`
   zh/en 键完全同构；en 用英文文案、zh 用中文文案；不得改动任何既有键。

## 约束

- 只动上述 4 个（2 新建 2 修改）文件；不碰 hooks/query-keys/后端。
- 文案全部走 i18n；写操作后果在 dialog description 讲清（安装会创建应用关联）。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-22/web && pnpm build && pnpm lint
```

双 exit 0；`git status` 无 routeTree.gen.ts 改动。另跑 locale 自检（两文件新子树
键集合相等）：可用
`node -e "const fs=require('fs');for(const f of ['src/i18n/locales/zh-CN.ts','src/i18n/locales/en.ts']){const s=fs.readFileSync(f,'utf8');const m=s.match(/catalog:\s*\{/g);console.log(f, m&&m.length)}"`
粗查 + 人工比对键名一致。

## 提交

```
git add src/features/config/apps/ src/i18n/locales/zh-CN.ts src/i18n/locales/en.ts
git commit -m "feat(admin): 应用目录浏览与安装表单 (issue #22)"
```

## 报告

完整报告写入 `.superpowers/sdd/issue-22-apps-catalog-install/task-2-report.md`
（文件清单、关键实现决策、验证命令与输出摘要、提交哈希、疑虑），回复只给四行：
状态/提交哈希/一行测试摘要/疑虑。
