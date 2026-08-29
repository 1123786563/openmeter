# Issue #21 — 应用：已装列表/卸载/Stripe 换 Key

- Issue: https://github.com/1123786563/openmeter/issues/21
- Branch: `codex/admin-config-21` (base `5a4666ec7`), worktree
  `/Users/wuyongjun/trea/openmeter-issue-21`.
- Prescriptive source: issue #21 comment 1（逐文件精确代码）.
  本文档补充本轮核验出的偏差与验收契约。

## 范围

替换 `/config/apps` 占位为真实页面：

1. `web/src/api/query-keys.ts` — 追加 `apps: () => ns('apps')`。
2. `web/src/api/hooks.ts` — Apps 段：`useApps` / `useUpdateApp` / `useUninstallApp`，
   成功后 `invalidateQueries(nsPrefix('apps'))`。
3. 新建 `web/src/features/config/apps/index.tsx` — 已装列表（名称/类型徽章/
   状态徽章/能力徽章/Stripe 信息/操作）+ 卸载确认 + 换 Key 入口；空态行。
4. 新建 `web/src/features/config/apps/stripe-key-dialog.tsx` — zod 单字段
   `secretApiKey`，PUT 回传 name/type 必填与现有 description/labels。
5. `web/src/routes/_authenticated/config/apps/index.tsx` — 占位替换为 `AppsPage`。
6. `web/src/i18n/locales/zh-CN.ts`、`en.ts` — `config.apps` 子树（含 #1 stub 的
   title/description 升级）。

## 非目标

- 应用目录区块与安装 dialog（#22）；`default_for_capability_types` 展示（安装
  响应专属字段，GET 列表不含）。
- OAuth 流程（spec 无）；Sandbox/外部开票应用的编辑 dialog。
- 后端改动；SDK 重新生成。

## 已核验偏差（相对 issue comment 1）

- **D-1（SDK 命名空间）**: comment 中 `api.apps.list/update/uninstall` 在安装的
  `@openmeter/client` 中不存在——OpenMeter 根客户端无 `apps` 属性；全部 apps
  操作位于 `api.internal.apps`（dist/sdk/sdk.d.ts + internal.d.ts + 包 README
  `client.internal.apps.update → PUT /openmeter/apps/{appId}` 实证）。请求/响应
  类型（`App`/`AppStripe`/`AppPagePaginatedResponse`/
  `UpdateAppStripeRequest{type:'stripe',name,description?,labels?,secretApiKey?}`）
  与 comment 契约逐字段一致。**实现一律用 `api.internal.apps.*`**。
- **D-2（插入锚点）**: comment 要求 Apps 段插在 "Organization default tax codes"
  段之后——该段属未实施的 #20，尚不存在。改为插在 "Notification channels (v1)"
  段之后、`Helpers` 之前（域分组意图不变）。
- locale：#1 已留 `config.apps.{title,description}` stub → 原位替换为 comment 的
  完整子树（不产生重复键）。

## 任务拆分

- T1 API 层：query-keys + hooks（D-1/D-2 生效）。门禁 `pnpm build && pnpm lint`。
- T2 页面层：stripe-key-dialog.tsx + apps/index.tsx + 路由替换。门禁同上。
- T3 i18n：zh/en 完整 `config.apps` 子树 + 键位对齐校验。门禁同上 + locale parity。

每任务独立 commit；实施后过规格符合性审查与代码质量审查（程序化，独立复跑），
Critical/Important 问题进入最多 5 轮修复与 scoped re-review；全分支最终审查。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- 三连以最终 HEAD 复跑；`pnpm test:e2e` 若因 :8888 shim（用户侧
  openmeter_shim.py，PID 21142）环境性失败，须以 pristine base 5a4666ec7
  对照判定非回归（先例：01:5x 轮）。
- 验收 walkthrough（临时 Playwright spec，全端点 mock，跑后删除）：
  1. 已装列表渲染（Sandbox ready 徽章 + 能力徽章 + Stripe 行 accountId/
     maskedApiKey/livemode）。
  2. 卸载：确认 dialog → DELETE `/api/v3/openmeter/apps/{id}`（wire 断言）→
     列表失效刷新 + toast。
  3. 换 Key：dialog 提交 → PUT `/api/v3/openmeter/apps/{id}` body 精确断言
     `{type:'stripe', name, description?, labels?, secretApiKey}`（SDK 序列化）→
     toast + 关闭。
- locale parity：zh/en 键数相等且新增键全部落位。

## 全局约束

- 遵循仓库 AGENTS.md/CLAUDE.md；前端遵循现有 hooks/query-keys/nsPrefix 模式。
- 不动后端、不动 e2e 既有用例、不引入新依赖。
- 未经用户批准不 push/merge/关闭 Issue/改 GitHub 状态。
- SDD 台账：`.superpowers/sdd/issue-21-apps-installed/progress.md`（worktree 内，
  gitignored）。
