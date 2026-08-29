# Issue #10 — Addons 独立管理页

- Issue: https://github.com/1123786563/openmeter/issues/10（[admin-config 10/29]；#11 计费依赖本页）
- 分支: codex/admin-config-10（worktree /Users/wuyongjun/trea/openmeter-issue-10，base f60cb90b0）
- 总纲: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 5

## 范围

替换 `/config/addons` 占位路由为完整管理页：
- 列表（name/key/version/instanceType/currency/status/价卡数/createdAt）+ 状态操作
- 创建/编辑 Dialog（key+currency 创建后不可改，PUT 为 UpsertAddonRequestInput）
- 价卡编辑复用 plans 的 rateCardSchema/priceFormSchema/toRateCardInput/fromRateCardToForm（free/flat/unit，无 tiered）
- publish（draft→active）/ archive / delete（ConfirmDialog）

SDK（生成物不改）: `api.addons.{list,create,update,get,delete,archive,publish}`；CreateAddonRequestInput={name,description?,labels?,key,instanceType,currency,rateCards: RateCardInput[]}；UpsertAddonRequestInput 同名去 key/currency。

## 非目标

- 订阅级 addon 附加/移除（#11 计费侧）
- tiered 价卡、unitConfig、paymentTerm（后续按需）
- 价卡级 currency 覆盖（继承 addon currency）

## 任务拆分

- T1 数据层: queryKeys.addons + useAddons/useCreateAddon/useUpdateAddon/useDeleteAddon/useArchiveAddon/usePublishAddon（invalidation addons 前缀）
- T2 契约导出: plan-form-schema.ts 将 toRateCardInput/fromRateCardToForm 改为导出（doc 注明 #10 复用）
- T3 页面: features/config/addons/{index.tsx, addon-form-dialog.tsx}；路由换挂；i18n config.addons.* 双语

## 测试与验收命令

- `pnpm build` / `pnpm lint` / prettier（新文件 clean，修改零新偏差）/ routeTree 零 diff / locale 奇偶 / e2e 签名对照
- 验收（Issue 原文）：可创建 addon（名称/key/货币/实例类型/价卡）、编辑不可变字段锁定、草稿可发布/归档、可删除

## 全局约束

- 同总纲（写操作 ConfirmDialog、handleServerError、i18n 双语、SDK 调用、价卡契约唯一定义处不复制）
