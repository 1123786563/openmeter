# Issue #14 — 通知规则：发票型与列表/启停 实施计划

- **Issue**: https://github.com/1123786563/openmeter/issues/14
- **处方源**: Issue #14 评论 1（逐文件完整代码，2026-08-26）；总计划 `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 8 前半
- **Branch / Worktree**: `codex/admin-config-14` @ base `49d1a760b`；`/Users/wuyongjun/trea/openmeter-issue-14`
- **Ledger**: `.superpowers/sdd/issue-14-notification-rules/progress.md`

## 范围

规则列表（类型徽章/名称/渠道名/禁用开关）+ 创建/编辑表单：支持 `invoice.created` / `invoice.updated` 两型（公共字段 name[1-256]/channels 多选 minItems 1/disabled），启停经全量 PUT 回传。

- Modify: `web/src/api/legacy.ts`（v1 notification rules 段：判别联合完整四型 + `listNotificationRules`/`createNotificationRule`/`updateNotificationRule`/`ruleToUpdateBody`）
- Modify: `web/src/api/query-keys.ts`（`notificationRules`）
- Modify: `web/src/api/hooks.ts`（`useNotificationRules`/`useCreateRule`/`useUpdateRule`/`useToggleRule`）
- Create: `web/src/components/multi-select.tsx`（通用多选下拉，Popover+Command+Checkbox；#15/#16 复用）
- Create: `web/src/features/config/notification/rule-form-dialog.tsx`（类型选择器 + 发票型表单，创建/编辑）
- Create: `web/src/features/config/notification/rules.tsx`（列表 + 类型徽章 + 渠道名解析 + 启停确认 + 分页）
- Modify: `web/src/routes/_authenticated/config/notification/rules/index.tsx`（占位 → 真页面）
- Modify: `web/src/i18n/locales/zh-CN.ts`、`en.ts`（`config.notification.rules.*`）

## 非目标

- 阈值型/重置型的表单编辑与 `POST /rules/{id}/test` 按钮（#15）
- 事件流页（#16）；渠道管理（#12/#13 已交付）
- 列表对四型规则仍须正确渲染徽章与启停（全量 body 回传天然支持任意型），仅表单开放 invoice 两型

## 关键契约（来自处方，字段名一律以 `api/openapi.yaml` 为准）

1. **GET** `/api/v1/notification/rules`：`includeDeleted`/`includeDisabled`（管理端传 true）/`feature[]`/`channel[]`/`page`/`pageSize`；响应 `{totalCount,page,pageSize,items}`；重复数组参数 `?feature=a&feature=b`。
2. **实体** `NotificationRule` oneOf 判别 `type` 四型；公共 `id/name/disabled/channels[]({id,type})/metadata?/createdAt/updatedAt/deletedAt?`。
3. **POST/PUT** body=`NotificationRuleCreateRequest`（oneOf 同判别）：公共 `name`[1..256]/`disabled?`/`channels`（string[] minItems 1）/`metadata?`；**PUT 全量替换**——启停必须 `ruleToUpdateBody(rule, disabled)` 回传类型特定字段。
4. **query key**：`notificationRules(params)` → `['api', ns, 'notification.rules', params]`；hooks invalidation 用 `nsPrefix('notification.rules')`。
5. **表单**：zod discriminatedUnion 仅 invoice 两型；type 编辑态禁用（创建后不可改）；channels 用 `MultiSelect`（数据 `useNotificationChannels({page:1,pageSize:100})`）。
6. **列表**：徽章按四型 type 渲染（含非编辑型）；编辑按钮仅 invoice 型（`isEditableType` 类型守卫）；启停 ConfirmDialog destructive=启用→禁用方向。

## 任务拆分

- **T1 数据层**：`legacy.ts` rules 段（处方步骤 1 完整代码）+ `query-keys.ts` + `hooks.ts`（处方步骤 2/3）。门禁：`pnpm build`。
- **T2 通用组件**：`web/src/components/multi-select.tsx`（处方步骤 4 完整代码）。门禁：`pnpm build`。
- **T3 页面与表单**：`rule-form-dialog.tsx` + `rules.tsx` + 路由替换 + 双语 i18n（处方步骤 5–8）。门禁：`pnpm build && pnpm lint && pnpm test:e2e`。

每任务：新 implementer subagent 实现 → 独立规格符合性审查 + 代码质量审查（Critical/Important 进入 ≤5 轮修复与 scoped re-review）。全部任务后：最强可用模型全分支审查。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- 三连门禁（e2e 冒烟须与 pristine base 同签名对照，:8888 用户 shim 环境性失败基线沿用既有裁定，非回归）
- 验收（Issue AC）：可创建发票型规则（渠道多选生效）并启停
- 程序化走查证据：zod 校验（不选渠道被拦截）、PUT 全量 body 回传路径、i18n 键两份对称（en/zh 键数一致）、routeTree 生成纯净

## 全局约束

- 文案全部 i18n（zh-CN + en 同步维护）；无后端改动；v1 手写层保持 camelCase 与 spec 一致
- 遵循仓库 TS 约定与 AGENTS.md；不引入 `any`/`@ts-ignore`/`eslint-disable`
- 每任务完成即提交（commit message 见处方）；台账 `.superpowers/sdd/issue-14-notification-rules/progress.md` 逐任务记录
