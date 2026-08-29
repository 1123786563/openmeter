# Plan — issue #11 附加组件：计划内管理 tab

- Issue: https://github.com/1123786563/openmeter/issues/11
  ([admin-config 11/29] 附加组件：计划内管理 tab)
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 6 后半
  （前半 #10 addons 独立管理页、#4 计划详情只读页均已合入 main = ec85f6871）。
- 处方源（权威）：issue #11 comment 1 全文，已存
  `/tmp/issue-round3/issue-11-comments.md`（逐文件完整代码）。
- Branch codex/admin-config-11 @ base ec85f6871；worktree
  /Users/wuyongjun/trea/openmeter-issue-11。
- Ledger: .superpowers/sdd/issue-11-plan-addons-tab/progress.md（主检出目录）。

## 范围（Scope）

- `web/src/api/query-keys.ts`：追加
  `planAddons: (planId: string, params: object = {}) => ns('plan-addons', planId, params)`
  （处方预设「#10 已落位，若缺失则补」——实测 main 缺失，**必须补**）。
- `web/src/api/hooks.ts`：追加「Plan addons」分节四个 hooks（处方步骤 1 逐字）：
  `usePlanAddons` / `useCreatePlanAddon` / `useUpdatePlanAddon` / `useDeletePlanAddon`，
  onSuccess 均失效 `nsPrefix('plan-addons')`（nsPrefix 为 hooks.ts:1363 模块私有 helper）。
- 新建 `web/src/features/config/plans/plan-addon-form-dialog.tsx`（处方步骤 2 逐字，
  见下方接口适配 Ruling-1/2）。
- 新建 `web/src/features/config/plans/plan-addons-tab.tsx`（处方步骤 3 逐字，
  ServerTable props 契约已核实与 main 现物一致）。
- `web/src/features/config/plans/plan-detail.tsx`：主内容区包 Tabs（overview 保留
  #4 既有信息/阶段视图原样移入；新增 addons tab 挂 `PlanAddonsTab`，处方步骤 4）。
- i18n：`config.planDetail.tabs.*` + `config.planAddons.*`（处方步骤 5 全键集），
  zh-CN.ts 与 en.ts 同构追加。

## 非目标（Non-goals）

- PlanAddon 无任何价格字段（spec 已核实）：本 tab 不使用 PriceForm；改价经行内
  链接跳 `/config/addons`（#10 页）。
- 不改 addons 独立管理页（#10 产物零改动）。
- 不做计划级阶段编辑；`from_plan_phase` 选项只读自 `plan.phases`。
- 无后端/TypeSpec/SDK 改动。

## 已核实的契约事实（main ec85f6871 实测 + SDK dist，2026-08-29）

- SDK `api.planAddons.list({planId, page: {number,size}}, {signal})` →
  `{ data: PlanAddon[], meta: { page: { number, size, total } } }`；create/update/delete
  签名与处方一致；`UpsertPlanAddonRequest` 无 `addon` 字段（关联创建后不可换）。
- `PlanAddon`：`{ id, name, description?, addon: {id}, fromPlanPhase, maxQuantity?,
  validationErrors?, createdAt, updatedAt }`（camelCase）。
- SDK `Addon`：`status: 'draft'|'active'|'archived'`、`instanceType: 'single'|'multiple'`
  （types.d.ts L5055-5115）。
- `ServerTable` props（web/src/components/data-table/server-table.tsx L25-40）：
  `columns/data/page/pageSize/total/onPageChange/isLoading/isFetching/emptyMessage`
  ——与处方用法完全一致。
- `queryKeys` 为对象字面量（query-keys.ts），`ns(...key)` variadic。

## 接口适配 Ruling（实施与审查均以此为准）

- **Ruling-1（useAllAddons 不存在）**：处方 Consumes 写 `useAllAddons()`，但 #10
  实际落地为 `useAddons()`（hooks.ts:1019，返回 v3 分页信封）。适配：组件内
  `const { data: addonsData } = useAddons()`，扁平化 `const allAddons = addonsData?.data ?? []`
  （与 addons/index.tsx:39-40 同模式）。**不新增 useAllAddons**（YAGNI）。
  代价：若未来 addons 超过 100 条下拉不全——与 #10 页同限，可接受。
- **Ruling-2（planAddons query key 补位）**：处方「若缺失则补」条件成立，按处方
  原式补入 query-keys.ts。与 #16 的 `notificationEvents` 追加同属 append-only，
  无跨轨冲突。
- 处方其余代码逐字有效；表单 maxQuantity 仅 multiple 实例可填、single 一律省略
  （spec 约束）；创建选 addon 时默认回填 name 仅 isCreate 分支。

## 任务拆分

- Task 1：数据层 + i18n——query-keys 补 `planAddons`；hooks.ts 追加四 hooks
  （逐字）；zh-CN/en 追加 `config.planDetail.tabs` + `config.planAddons` 全键集。
  验证：`cd web && pnpm build && pnpm lint`（i18n 键先落便于 T2 引用）。
- Task 2：组件层——新建 plan-addon-form-dialog.tsx / plan-addons-tab.tsx（处方
  逐字 + Ruling-1 适配）；plan-detail.tsx 包 Tabs 挂载（#4 视图原样移入 overview）。
  验证：`cd web && pnpm build && pnpm lint && pnpm test:e2e`。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- `pnpm build`（tsc -b + vite build）0 error；routeTree 零 diff（无路由变更，
  仅组件挂载）。注意：本机 `tsr generate` 会重排 routeTree.gen.ts 导入序——
  门禁以「恢复提交版后 tsc -b 通过 + git diff 对 routeTree 为零」为准。
- `pnpm lint` 0 error 0 warning（新增行）。
- e2e：既有 2 条冒烟（sign-in 应 PASS；customers smoke 在本环境历史性环境失败，
  以与 pristine base 同签名为非回归判据——历轮共识）。
- locale 奇偶：zh 键数 == en 键数（求值比较 config.planAddons + config.planDetail
  子树两侧同构）。
- prettier：新文件 clean；修改文件 formatted(base) vs formatted(mine) 差异恰为
  插入块（基线脏行不触碰）。

验收（Issue AC）：「在计划详情内可直接增删改该计划的附加组件」——代码面 =
plan-detail addons tab 可增删改 + 删除确认框 + toast；类型面 = 提交体与
CreatePlanAddonRequest / UpsertPlanAddonRequest 逐字段吻合（tsc 证明）。

## 全局约束（Master plan Global Constraints 摘要 + 本轨补充）

- 文案全部 i18n，zh-CN 与 en 两份同步；术语按 CONTEXT.md（附加组件/阶段）。
- 写操作一律 ConfirmDialog（删除 destructive）；错误经 handleServerError 原文透出。
- API 仅经 `@openmeter/client` SDK（本轨无 legacy/v1 端点）。
- 不引入新依赖；不 panic；组件命名/导出与处方一致（PlanAddonsTab/PlanAddonFormDialog）。
- 禁止臆造字段：所有请求/响应字段以 SDK 类型为准。
