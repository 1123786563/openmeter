# Task 1 brief — issue #11 计划内附加组件 tab（数据层 + i18n）

Worktree: /Users/wuyongjun/trea/openmeter-issue-11（branch codex/admin-config-11，
base ec85f6871；实施计划已随 plan commit 6aaa5b916 入分支）。

## 需求源（先读这两个文件）

1. 实施计划：`/Users/wuyongjun/trea/openmeter-issue-11/docs/superpowers/plans/issue-11-plan-addons-tab.md`
   （范围/非目标/Ruling/验收命令）。
2. 处方（权威，逐字代码）：`/tmp/issue-round3/issue-11-comments.md`。
   本任务只做其中 **步骤 1（hooks）** 与 **步骤 5（i18n）**，外加计划 Ruling-2 的
   query-keys 补位。步骤 2/3/4（组件与挂载）属 Task 2，**勿做**。

## 交付（按序）

1. `web/src/api/query-keys.ts`：在 `receivablePeriods` 条目之后追加
   `planAddons: (planId: string, params: object = {}) => ns('plan-addons', planId, params),`
2. `web/src/api/hooks.ts`：按处方步骤 1 逐字追加「Plan addons」分节
   （`PlanAddonListParams` + `usePlanAddons`/`useCreatePlanAddon`/
   `useUpdatePlanAddon`/`useDeletePlanAddon`；onSuccess 均失效
   `nsPrefix('plan-addons')`）。插入位置：既有「Addons (config)」分节之后或文件
   末尾模块私有 helper `nsPrefix` 之前，保持分节注释风格一致。
3. `web/src/i18n/locales/zh-CN.ts` 与 `web/src/i18n/locales/en.ts`：在 `config`
   对象内追加 `planDetail`（现不存在，全新子树：`tabs: { overview, addons }`）与
   `planAddons`（处方步骤 5 zh 全文键集；en 同构对应直译）。**不得改动/删除任何
   既有键**；zh 与 en 键集完全同构。

## 验证（全部在 worktree 的 web/ 目录）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-11/web && pnpm build && pnpm lint
```

- `pnpm build` 0 error；`pnpm lint` 0 error 0 warning。
- routeTree.gen.ts 零 diff（本任务无路由变更；若本机工具重排了它，恢复提交版再验证 tsc）。

## 提交

- 只 add 上述三类文件；commit message：
  `feat(admin): 计划内附加组件数据层与 i18n（issue #11 task 1）`

## 硬约束

- 处方代码逐字采用（含注释与命名）；不新增依赖；不新增 useAllAddons。
- 只在 /Users/wuyongjun/trea/openmeter-issue-11 内工作；绝不动主检出
  /Users/wuyongjun/trea/openmeter 或其他 worktree。
- 禁止派生任何 subagent。
