# Task 1 Report — 计划列表与详情（只读）

- Worktree: `/Users/wuyongjun/trea/openmeter-issue-4`，branch `codex/admin-config-04`
- Commit: `45f89e50da349027daf280c8549763466163ea73` `feat(admin): 计划列表与详情（只读）`
- 10 files changed, **547 insertions(+), 9 deletions(-)**

## 改动文件清单（+/- 行数）

| 文件 | +/- | 说明 |
|---|---|---|
| `web/src/api/query-keys.ts` | +2 | `plansPage`/`plan` 两键（scope 1） |
| `web/src/components/status-badge.tsx` | +6 | `tones` 新增 `plan` 域 4 状态（scope 2） |
| `web/src/api/hooks.ts` | +30 | `PlanListParams`/`usePlansPage`/`usePlan`，置于 `usePlans` 之后（scope 3） |
| `web/src/features/config/plans/index.tsx` | +153 | 新建 `PlansPage`（scope 4） |
| `web/src/features/config/plans/plan-detail.tsx` | +223 | 新建 `PlanDetail`（scope 5） |
| `web/src/routes/_authenticated/config/plans/index.tsx` | +2/-7 | 占位页整文件替换（scope 6） |
| `web/src/routes/_authenticated/config/plans/$planId.tsx` | +6 | 新建详情路由（scope 6） |
| `web/src/i18n/locales/zh-CN.ts` | +51/-1 | `config.plans` 完整块替换 + 顶层 `plan` 域（scope 7） |
| `web/src/i18n/locales/en.ts` | +52/-1 | 同步（scope 7） |
| `web/src/routeTree.gen.ts` | +22 | `pnpm build` 内 `tsr generate` 自动再生，仅新增 `$planId` 节点 |

提交范围自查（`git status --short` + `git diff --stat`）：改动 = brief 范围 1–7 文件 + `routeTree.gen.ts`，无其他文件。

## 验证（三项全绿）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-4/web
pnpm build 2>&1 | tee /tmp/issue4-build.log   # exit 0（tsr generate + tsc -b + vite build，✓ built in 285ms）
pnpm lint 2>&1 | tee /tmp/issue4-lint.log     # exit 0（eslint .，0 error 0 warning 输出）
pnpm test:e2e 2>&1 | tee /tmp/issue4-e2e.log  # exit 0（2 passed (6.2s)：sign-in smoke + customers smoke）
```

- `pnpm build`：exit 0（首次跑到类型错误后修复，见偏差清单；最终 log 尾部 `✓ built in 285ms`）。
- `pnpm lint`：exit 0，无任何输出（0 error）。
- `pnpm test:e2e`：exit 0，`2 passed`（既有两条 Playwright 冒烟不回归）。跑前已执行端口卫生 `lsof -ti :4173 :9999 | xargs kill -9`（无残留进程）。

## i18n 程序化比对（node 原生 type-stripping 一次性脚本）

脚本：`/tmp/issue4-i18n-check.ts`（node 直接加载两份 locale .ts，flatten 键树 + 提取 `{{var}}`）。结果：

```
PASS  config.plans: key tree identical (28 keys)
PASS  config.plans: interpolation variables identical
PASS  plan: key tree identical (10 keys)
PASS  plan: interpolation variables identical
INFO  whole-file flatten ok: zh 516 keys, en 516 keys (no duplicate keys thrown)
RESULT: all checks passed
```

- `config.plans` 28 键、顶层 `plan` 10 键，zh/en 键树完全一致；`{{index}}`/`{{duration}}`/`{{count}}` 插值变量逐键一致；全文件 516/516 键，flatten 过程无重复键异常（TS 对对象字面量重复属性本身也会报错）。

## 偏差清单（相对 Issue 评论处方代码）

1. **`PlansPage` 状态筛选的 state 类型**（评论代码 → 仓库规范模式）
   - 评论原文：`const [status, setStatus] = useState<string | undefined>()`，`onValueChange` 中 `setStatus(value === 'all' ? undefined : value)`。
   - 实际采用：`useState<PlanListParams['status']>()` + `setStatus(value === 'all' ? undefined : (value as PlanListParams['status']))`，并相应 `import { usePlansPage, type PlanListParams } from '@/api/hooks'`。
   - 原因与证据：`PlanListParams.status`（评论自己给出的契约）是窄联合 `'draft'|'active'|'archived'|'scheduled'`，宽 `string` 无法赋入。首次 build 报错 `src/features/config/plans/index.tsx(23,74): error TS2322: Type 'string | undefined' is not assignable to type '"active" | "draft" | "archived" | "scheduled" | undefined'`。修法逐字采用仓库既有同型模式（`web/src/features/commerce/orders.tsx:62` 的 `useState<OrderListParams['status']>()` 与 `:199-203` 的 cast），与 plan「已知偏差风险」节的处置方向一致。
2. **`plan-detail.tsx` 移除未使用的 `Plan` 类型导入**
   - 评论原文：`import type { Plan, RateCard } from '@openmeter/client'`。
   - 实际采用：`import type { RateCard } from '@openmeter/client'`（`Plan` 在组件内无类型注解引用）。
   - 原因与证据：`tsconfig` `noUnusedLocals: true`，首次 build 报错 `src/features/config/plans/plan-detail.tsx(2,15): error TS6196: 'Plan' is declared but never used.`。
3. **格式化适配（非语义偏差，仅记录）**：两个新文件按仓库 `.prettierrc` 跑 `prettier --write`（import 排序与换行微调）；i18n 块缩进从评论草稿的 8 空格适配为文件实际嵌套层级；en.ts `plans.description` 一行超 80 列由 prettier 折行（位于本次新增块内）。`hooks.ts` 预存 recharge hunk 格式违规（485-488 行，已核实为该文件唯一违规）按最小 diff 原则未动。
4. **未触发的预期风险（记录性）**：`useParams({ from: '/_authenticated/config/plans/$planId' })` 与路由 `component: PlanDetail` 直接命名导入**均未**触发 eslint/TS 报错（与 `meter-detail`/`subscription-detail` 等既有页面同型），无需切换 #3 的 `Route.useParams()` prop 模式。

其余全部按评论代码逐字实施（query-keys 两行、status-badge `plan` 域、hooks 三段、列表/详情两个组件主体、两个路由文件、两份 i18n 完整块）。

## 环境准备记录（非代码改动）

- worktree 初始无 `web/node_modules`，且 SDK `file:../api/spec/packages/aip-client-javascript` 按 `files: ["dist", ...]` 注入时 `dist` 尚未构建，导致 `@openmeter/client` 解析为空。处置：先在 SDK 包内 `pnpm install && pnpm build`（tsc 产 dist），再于 `web/` 全量 `rm -rf node_modules && pnpm install --force` 重新注入（增量 install 不会重打包 directory 依赖）。与 issue-1 plan 中「如 build 报模块缺失先构建其 package」的说明一致。`pnpm-lock.yaml` 无改动（`git status` 干净验证）。

## git 记录

- `git log -1`: `45f89e50da349027daf280c8549763466163ea73 feat(admin): 计划列表与详情（只读）`（单次 commit，10 files，+547/-9）
- 最终 `git status --short`：仅 `?? .superpowers/sdd/issue-4-plans-list-detail/`（本报告所在目录，按 brief 约定不入提交），工作树其余干净。
