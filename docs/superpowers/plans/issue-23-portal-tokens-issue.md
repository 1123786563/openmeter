# Issue #23 — 门户令牌：发放（一次性明文）

- Issue: https://github.com/1123786563/openmeter/issues/23
- Branch: `codex/admin-config-23` (base `5a4666ec7`), worktree
  `/Users/wuyongjun/trea/openmeter-issue-23`.
- Prescriptive source: issue #23 comment 1（逐文件精确代码）。
  本文档补充验收契约。

## 范围

1. `web/src/api/legacy.ts` — 文件末尾追加 Portal tokens (v1) 段：
   `PortalToken` 接口 + `createPortalToken(body)`（POST `/v1/portal/tokens`）。
2. `web/src/api/query-keys.ts` — `portalTokens: (params: object = {}) =>
   ns('portal-tokens', params)`（#24 复用）。
3. `web/src/api/hooks.ts` — `useCreatePortalToken()`（成功后
   `invalidateQueries(nsPrefix('portal-tokens'))`）。
4. 新建 `web/src/features/config/portal-tokens/token-once-dialog.tsx` —
   一次性明文弹窗（复制按钮含非安全上下文降级、`onPointerDownOutside`
   防误关、警示条）。
5. 新建 `web/src/features/config/portal-tokens/issue-token-dialog.tsx` —
   CustomerPicker + MeterMultiSelect（Popover+Command+Checkbox，空=全部 meter）；
   成功 → toast + 关表单 + 开一次性明文弹窗。
6. 新建 `web/src/features/config/portal-tokens/index.tsx` — 页面（PageHeader
   actions 发放按钮）。
7. `web/src/routes/_authenticated/config/portal-tokens/index.tsx` — 占位替换。
8. `web/src/i18n/locales/zh-CN.ts`、`en.ts` — `config.portalTokens` 子树
   （#1 stub 升级）。

## 非目标

- 令牌列表与失效操作（#24）；expiresAt 设置（spec readOnly）；
  后端改动；v1 SDK 化。

## 已核验规格事实（api/openapi.yaml，本轮一手复核）

- `POST /api/v1/portal/tokens` requestBody = `PortalToken` schema：可写仅
  `subject`（必填）与 `allowedMeterSlugs`（可选 string[]）；`token` readOnly
  且 "The token is only returned at creation"，示例前缀 `om_portal_`。
- `GET /api/v1/portal/tokens`（listPortalTokens）存在，#24 使用；列表不含 token。
- 无偏差：comment 契约与 spec 逐字段一致。

## 任务拆分

- T1 API 层：legacy + query-keys + hooks。门禁 `pnpm build && pnpm lint`。
- T2 页面层：三个组件 + 路由替换。门禁同上。
- T3 i18n：zh/en `config.portalTokens` 子树 + parity。门禁同上。

每任务独立 commit；规格符合性 + 代码质量双审查（程序化，独立复跑）；
Critical/Important 最多 5 轮修复与 scoped re-review；全分支最终审查。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- 三连以最终 HEAD 复跑；:8888 shim 环境性失败须 pristine base 对照判非回归。
- 验收 walkthrough（临时 Playwright spec，全端点 mock，跑后删除）：
  1. `/config/portal-tokens` 渲染，发放按钮开 dialog；客户必选校验（不选客户
     提交 → 错误提示、零 POST）。
  2. 选客户 + 勾选/取消 meter → 提交 → POST `/api/v1/portal/tokens` body 精确
     断言 `{subject: customer.key, allowedMeterSlugs?}`（空选=缺省全量）。
  3. 一次性明文弹窗出现 `om_portal_` 前缀、复制按钮、警示文案；关闭后重开页面
     无处再见明文。
- locale parity：zh/en 键数相等且新增键全部落位。

## 全局约束

- 遵循仓库 AGENTS.md/CLAUDE.md；v1 走 legacy.ts 手写层（既有模式）。
- 不动后端、不动 e2e 既有用例、不引入新依赖。
- 未经用户批准不 push/merge/关闭 Issue/改 GitHub 状态。
- SDD 台账：`.superpowers/sdd/issue-23-portal-tokens/progress.md`（worktree 内，
  gitignored）。
