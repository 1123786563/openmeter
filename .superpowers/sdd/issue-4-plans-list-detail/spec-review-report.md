# Spec Review Report — Issue #4 Task 1（计划列表与详情·只读）

- 审查对象：worktree `/Users/wuyongjun/trea/openmeter-issue-4`，branch `codex/admin-config-04`，范围 `96e1111c0..45f89e50d`（单 commit `feat(admin): 计划列表与详情（只读）`，10 files，+547/−9）
- 规范来源：Issue #4 正文+处方化评论（`gh issue view 4 --json body,comments` 只读获取）→ 实施计划 → implementer 报告
- 审查方式：只读 diff/SDK 类型核对 + 独立重跑验证命令 + 偏差独立复现（/tmp 临时文件，已清理）
- 审查日期：2026-08-27（commit 45f89e50d，13:18:04 +08）

## 总裁定：**PASS**（Critical 0 / Important 0 / Minor 2）

实现与 Issue #4 处方化评论逐项忠实一致；两条代码级偏差均经审查员独立复现证明必要、最小、等价；三项验证证据真实且时间线自洽。

---

## 审查清单逐项结论

### 1. 文件清单与范围 — PASS

- `git diff --stat 96e1111c0..45f89e50d`：恰好 10 个文件 = 评论「文件（精确路径）」的 8 个（query-keys.ts / hooks.ts / status-badge.tsx / plans/index.tsx 新建 / plan-detail.tsx 新建 / routes index.tsx 替换 / routes $planId.tsx 新建 / zh-CN.ts + en.ts）+ `routeTree.gen.ts` 再生成，**无范围外文件**。
- 范围内单 commit，message 与评论规定逐字一致（`feat(admin): 计划列表与详情（只读）`）。
- `git status --short` 仅 `?? .superpowers/sdd/issue-4-plans-list-detail/`（SDD 目录按约定不入提交）；`pnpm-lock.yaml` 无改动，与报告声明一致。

### 2. hooks（`web/src/api/hooks.ts` +30）— PASS

- `PlanListParams` / `usePlansPage` / `usePlan` 与评论契约**逐字一致**：
  - `queryKey: queryKeys.plansPage(params)` / `queryKeys.plan(planId)`；
  - `queryFn: api.plans.list({ page: { number, size }, sort: { by: 'created_at', order: 'desc' }, filter: params.status ? { status: params.status } : undefined }, { signal })`；
  - `api.plans.get({ planId }, { signal })`；`enabled: Boolean(planId)`。
- 位置正确：紧随既有 `usePlans` 之后（diff 单 hunk，`@@ -170,6 +170,36 @@`）。
- **既有 `usePlans`（listAll 版）未动**：hooks.ts diff 纯增量（除 `---` 文件头外无删除行），`usePlans` 函数体与 base 逐字相同。
- SDK 契约核对（`api/spec/packages/aip-client-javascript/dist/models/operations/plans.d.ts` + `types.d.ts`）：
  - `ListPlansQuery = { page?: {size?,number?}, sort?: SortQueryInput, filter?: ListPlansParamsFilter }`；`ListPlansResponse = PlanPagePaginatedResponse`（`data: Plan[]` + `meta.page.total`）；`GetPlanResponse = Plan`（无 data 包裹）——与 hooks 用法吻合。
  - `ListPlansParamsFilter.status: StringFieldFilterExact = string | {eq?,oeq?,neq?}`（types.d.ts:5707）——纯字符串写法类型合法。

### 3. query-keys（+2）— PASS

- `plansPage: (params: object = {}) => ns('plans-page', params)` 与 `plan: (id: string) => ns('plan', id)` 两行与评论逐字一致；既有 `plans: () => ns('plans')` 保持原位原样。

### 4. status-badge（+6）— PASS

- `tones` 新增 `plan` 域：draft→secondary、active→success、archived→outline、scheduled→info，与评论逐字一致（status-badge.tsx:52-57）。
- diff 仅此一处插入；`StatusBadge`/`EnumBadge` 既有逻辑零改动。`EnumBadge` 解析 `${domain}.${kind}.${value}` → `plan.priceType.<value>`，与评论契约吻合（status-badge.tsx:121-141）。

### 5. 列表页 `web/src/features/config/plans/index.tsx`（+153）— PASS

- **7 列**齐备：name（`Link to='/config/plans/$planId'` params={{planId: row.original.id}}）/ key（mono）/ v+version（tabular-nums）/ status（`StatusBadge domain='plan'`）/ currency / billingCadence / createdAt（formatDateTime）——列定义与评论逐行一致（index.tsx:35-94）。
- **状态筛选 Select**：`'all'` + `STATUS_OPTIONS = ['draft','active','scheduled','archived']`（与评论同序）；`onValueChange` 先 `setPage(1)` 再 `setStatus(...)`——切换重置页码 ✓（index.tsx:21,122-131,136-145）。
- **ServerTable 分页接线**：`onPageChange` 中 pageSize 变化 → `setPageSize + setPage(1)`，否则 `setPage(next.pageIndex + 1)`，与评论逐字一致；`total={data?.meta.page.total}`、`data={data?.data ?? []}` 吻合 `PlanPagePaginatedResponse` 契约（index.tsx:104-120）。`ServerTable` 实际 props（server-table.tsx:31-39：`onPageChange: (pagination: {pageIndex, pageSize}) => void`）与用法一致。
- **空态**：`emptyMessage={t('config.plans.empty')}` ✓（index.tsx:148）。
- 与评论的唯一差异为偏差 1（见第 9 项裁定）。

### 6. 详情页 `web/src/features/config/plans/plan-detail.tsx`（+223）— PASS

- **loading skeleton**：两段 Skeleton 兜底（plan-detail.tsx:82-92）✓；**notFound 兜底**：`if (!plan)` → `config.plans.detail.notFound`（:93-104）✓。
- **InfoRow 基本信息**：key（mono）/ currency / billingCadence / createdAt / updatedAt / `description?`（条件渲染）——6 行齐备（:124-147）。
- **阶段→价目卡两级**：`plan.phases.map` 每阶段一 Card；标题 `phaseIndex · phase.name` + `duration ? duration : noDuration`（:152-170）。
- **价目卡表 6 列**：rateCardName / priceType（`EnumBadge domain='plan' kind='priceType'`）/ feature / price / cadence / key（:174-183）。
- **RateCardPriceSummary 5 分支**：free（文案）/ flat（formatAmount）/ unit（amount / 单位）/ graduated|volume 合并折叠为 `tierSummary {{count}}` 档数（:47-74）。
- 三个回退语义逐字到位：`card.currency ?? planCurrency`（:46）、`card.feature?.id ?? '—'`（:200）、`card.billingCadence ?? plan.billingCadence`（:209）。
- `formatAmount(amount: string|number, currency?: string)`、`formatDateTime` 均为 `web/src/lib/format.ts` 既有导出（:8/:35）。

### 7. 路由 — PASS

- `routes/_authenticated/config/plans/index.tsx`：整文件替换为评论给定 6 行（占位 `PlaceholderPage` 移除，`component: PlansPage`）。
- `routes/_authenticated/config/plans/$planId.tsx`：新建，与评论逐字一致（6 行，`component: PlanDetail`）。
- `routeTree.gen.ts`（+22）：逐 hunk 核对，全部差异均为 `$planId` 节点标准接线（import、`update()` 块、FileRoutesByFullPath/ByTo/ById、FileRouteTypes 三处 union、模块声明、Children 接口与实现各一处），**无任何 $planId 无关改动**。

### 8. i18n — PASS

- zh-CN / en 两份的 `config.plans` 块（title/description/empty/filter×2/fields×9/detail×14 = 28 叶键）与顶层 `plan` 域（status×4/priceType×5/price.free = 10 叶键）与评论**逐键逐值一致**（含 `{{index}}`/`{{duration}}`/`{{count}}` 插值）。
- 审查员独立复跑 `/tmp/issue4-i18n-check.ts`（node type-stripping，只读加载两份 locale）：`PASS ×4 + INFO 516/516`，与 implementer 报告输出逐字一致。
- 唯一文本差异：en `plans.description` 被 prettier 折为两行（内容不变，属偏差 3 记录范围）；旧占位 description 按评论规定替换。

### 9. implementer 偏差逐条裁定 — 全部成立

**偏差 1（`PlansPage` 状态筛选 state 类型）— 必要 ✚ 最小 ✚ 等价 ✚，裁定：正当**

- **必要性独立复现**：审查员在 /tmp 以 worktree 真实 `@/api/hooks` 类型构建 tsc 程序（继承 `tsconfig.app.json` 全部严格选项），运行评论原文 `useState<string | undefined>()` + `usePlansPage({ page, pageSize, status })`：
  ```
  repro.tsx(9,44): error TS2322: Type 'string | undefined' is not assignable to type
  '"draft" | "active" | "archived" | "scheduled" | undefined'.
  ```
  与 implementer 报告的 `index.tsx(23,74) TS2322` 同码同型（仅联合序序列化不同）。根因：评论**自身**的 `PlanListParams.status` 窄联合与评论**自身**的宽 `string` state 写法互斥——评论处方内部矛盾，不改必不能编译。
- **最小性**：改动仅 3 处（import 加 type、useState 泛型、onValueChange cast），保留评论的 `PlanListParams` 接口原样；备选（放宽接口 status 为 string）偏离处方更大。
- **等价性/仓库规范性**：逐字复用仓库既有同型模式 `web/src/features/commerce/orders.tsx:62`（`useState<OrderListParams['status']>()`）与 :197-204（`value as OrderListParams['status']` cast），该模式经 `git log` 确认为分支前既有（925f6be4d）。运行时语义与评论完全一致；cast 安全（Select 只产出 'all' 或 4 个状态值）。

**偏差 2（移除未使用的 `Plan` 类型导入）— 必要 ✚ 最小 ✚ 等价 ✚，裁定：正当**

- `tsconfig.app.json` 确认 `noUnusedLocals: true`；审查员 /tmp 复现：未使用的 `import type { Plan }` 报 `error TS6133: 'Plan' is declared but its value is never read.`（与报告所记 TS6196 同为 noUnusedLocals 未使用导入错误类，仅错误码记法差异，见 Minor-1）。type-only 导入删除零运行时影响。

**偏差 3（格式化适配）— 裁定： acceptable（非语义）**

- 审查员对全部 9 个改动源文件跑 `prettier --check`：8 个新文件/块全部 clean，`hooks.ts` 唯一被标记者为 recharge hunk（485-488 行）。
- **预存性经 git 考古确证**：`git blame 96e1111c0 -L 452,459` 显示该多行形式由更早的 925f6be4d（管理端全量交付）引入，本次 diff 未触碰该区域；计划「非目标」节亦明示不处理该预存违规。`format:check` 不在本任务验证门（build/lint/test:e2e）内，CI workflows 亦无 prettier 门禁。

**偏差 4（`useParams({ from })` 风险未触发）— 裁定：记录属实**

- 审查员全量 tsc（经 routeTree.gen.ts 拉入全部路由与页面）exit 0 + `pnpm lint` exit 0，证实该写法在当前工具链下无 eslint/TS 报错，无需切换 #3 模式。

### 10. 验证证据真实性 — PASS

- **审查员独立重跑**：`cd web && pnpm lint` → `$ eslint .`，无任何输出，**exit 0** ✓。
- **审查员独立类型检查**：/tmp 程序含 `routeTree.gen.ts`（拉入全部路由/页面/两新组件）→ tsc **exit 0**，独立佐证 build 的 tsc 环节。
- **三份日志核验**：
  - `/tmp/issue4-build.log`（13:17）：`tsr generate && tsc -b && vite build`，尾部 `✓ built in 285ms`；产物含 `dist/assets/_planId-r6c9s91V.js`（4.82 kB）与 `plans-BGQT2jxw.js`（2.63 kB，真实列表页；对照基线 0.21 kB 占位）。
  - `/tmp/issue4-lint.log`（13:17）：11 字节 `$ eslint .` 后无输出 = 0 error。
  - `/tmp/issue4-e2e.log`（13:17）：`2 passed (6.2s)`（sign-in smoke + customers smoke）；WebServer 以 `--strictPort 4173` 成功起 preview，间接佐证端口卫生已做。
  - 时间线自洽：baseline-build 13:13（无 `_planId` 块、plans 占位 0.21 kB = 改动前基线）→ i18n 脚本 13:16 → 三份验证日志 13:17 → commit 45f89e50d 13:18:04。

---

## 发现分级

### Critical（规范违背/功能缺失）— 0

无。

### Important（明显缺陷）— 0

无。

### Minor — 2

1. **task-1-report 偏差 2 的错误码记法**：报告记为 `TS6196`，本仓库当前 tsc 对同一条件（noUnusedLocals 下未使用的类型导入）实际产出 `TS6133`。同错误类、纯文档层面的记法差异，无代码影响（错误条件本身已由审查员复现证实）。
2. **首次失败 build 的原始输出未留存**：/tmp 仅存基线（13:13）与最终（13:17）两份 build 日志，首次含 TS2322/TS6133 的失败输出未 tee 落盘。鉴于两条错误均已由审查员独立复现、且基线→最终日志时间线完整，偏差叙事成立；仅作证据卫生提示（后续任务建议首次失败也落盘）。

## 结论

10 项审查清单全部 PASS。处方化评论的所有可执行条款（文件清单、hooks/query-keys/status-badge 契约、列表 7 列 + 筛选 + 分页接线、详情两级视图 + 5 分支价格摘要 + 3 处回退语义、路由与 routeTree 再生成边界、i18n 双语逐键）均忠实落地；两条代码级偏差均为评论处方自身缺陷（自相矛盾的 state 类型、未使用导入）所迫，处置方式最小且逐字沿用仓库既有模式。验证三项（build/lint/e2e）证据真实、可复现、时间线自洽。**总裁定 PASS，无需整改即可进入质量审查与浏览器 walkthrough。**
