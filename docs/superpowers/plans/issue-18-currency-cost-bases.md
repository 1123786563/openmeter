# Issue #18 — 自定义货币成本基准管理

- Issue: https://github.com/1123786563/openmeter/issues/18
- 依赖: 自定义货币 CRUD 已在 main（useCurrencies expand cost_basis 行内已带数据）
- 分支: codex/admin-config-18（worktree /Users/wuyongjun/trea/openmeter-issue-18，base f60cb90b0）
- 总纲: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 4 剩余步

## 范围

自定义货币行的成本基准查看 + 追加：
- 行内「成本基准」计数改为可点按钮 → Dialog 展示既有 cost basis 列表（法币/费率/有效期/创建时间）+ 追加表单（法币下拉、费率、可选生效日）
- SDK（生成物不改）: `api.internal.currencies.createCostBasis({currencyId, body})`，body=`{fiatCode, rate: string, effectiveFrom?: Date, effectiveTo?: Date}` → CostBasis
- 列表数据复用 useCurrencies expand=cost_basis（不新增列表 hook）；创建成功 invalidate currencies 前缀

## 非目标

- 不做 cost basis 编辑/删除（API 无端点；append-only 语义，新记录覆盖旧记录）
- 表单不含 effectiveTo（保持开放直至被替代；API 支持，后续需要再加）
- 不改法币 Tab、自定义货币创建表单

## 任务拆分

- T1 数据层: hooks 加 `useCreateCostBasis()`（invalidation currencies）
- T2 对话框+挂载+i18n: `cost-basis-dialog.tsx`（Form+zod，模式对齐 custom-currency-dialog；提交成功留窗刷新列表+重置表单）；index.tsx 计数单元格改按钮；i18n config.currencies.costBasis.* 双语

## 测试与验收命令

- `pnpm build` / `pnpm lint` / 新文件 prettier clean + 修改文件零新偏差 / routeTree 零 diff
- locale 奇偶 en=zh
- `pnpm test:e2e`（签名对照 pristine base）
- 验收（Issue 原文）：每个自定义货币可查看其成本基准并追加新条目

## 全局约束

- 同总纲（i18n 双语、SDK 调用、错误 handleServerError 透出、写操作反馈）
