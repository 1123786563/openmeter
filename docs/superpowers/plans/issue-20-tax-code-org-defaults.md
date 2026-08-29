# Issue #20 — 税码：组织默认设置

- Issue: https://github.com/1123786563/openmeter/issues/20（[admin-config 20/29]，ready-for-agent）
- 依赖: #19 已合并（税码 CRUD 已在 main）
- 分支: codex/admin-config-20（worktree /Users/wuyongjun/trea/openmeter-issue-20，base f60cb90b0）
- 总纲: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 10 第二步（默认税码卡片）

## 范围

组织默认税码卡片（挂税码列表页顶部）：开票税码与额度发放税码两个下拉（数据=税码列表，排除已删除），GET/PUT `/openmeter/defaults/tax-codes`，保存成功 toast。

SDK（已核实，生成物不改）：
- `api.defaults.getOrganizationTaxCodes({} , {signal})` → `OrganizationDefaultTaxCodes { invoicingTaxCode: TaxCodeReference{id}, creditGrantTaxCode: TaxCodeReference{id}, createdAt, updatedAt }`
- `api.defaults.updateOrganizationTaxCodes(body)`，body = `{ invoicingTaxCode?: {id}, creditGrantTaxCode?: {id} }`

## 非目标

- 不动税码 CRUD（#19 已交付）
- 不做删除默认/清空默认（后端两字段语义为引用，UI 始终双值提交）
- 不改 SDK/后端/i18n 既有键

## 任务拆分

- T1 数据层: query-keys 加 `orgDefaultTaxCodes`；hooks 加 `useOrgDefaultTaxCodes()` / `useUpdateOrgDefaultTaxCodes()`（invalidation 前缀 org-default-tax-codes）
- T2 卡片+挂载+i18n: 新建 `web/src/features/config/tax-codes/org-defaults-card.tsx`（props: taxCodes，内部过滤 deletedAt；两 Select 初值来自服务端 defaults，options 仅约束新选择；保存双值 PUT）；index.tsx 挂载；i18n config.taxCodes.defaults.* 双语

## 测试与验收命令

- `pnpm build` / `pnpm lint` / 新文件 prettier 零偏差 / routeTree 零 diff
- locale 奇偶校验 en=zh（新增 defaults 子树两侧同步）
- `pnpm test:e2e`（:4173/8888 环境，签名与 pristine base 对照）
- 验收（Issue 原文）：两个默认位可设置并保存成功

## 全局约束

- 总纲 Global Constraints（i18n 双语、API 经 client/SDK、写操作反馈、snake_case 由 SDK 转换）
- 组件模式对齐既有 config 页（Card/Select/Label/Button/toast/handleServerError）
