# Issue #4 实施计划 — 计划：列表与详情（只读）

- Issue: https://github.com/1123786563/openmeter/issues/4 `[admin-config 04/29] 计划：列表与详情（只读）`
- 总纲: `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 3 前半（只读部分）
- Worktree: `/Users/wuyongjun/trea/openmeter-issue-4`（branch `codex/admin-config-04`，base `18309b955` = main = origin/main，含 #1–#3 成果）
- 规范来源：Issue #4 正文 + 处方化评论计划（完整代码）；SDK 契约已由控制器逐项对照 `api/spec/packages/aip-client-javascript/dist/` 核实（见下）

## 范围（Scope）

1. `web/src/api/query-keys.ts`：`queryKeys` 新增 `plansPage: (params: object = {}) => ns('plans-page', params)` 与 `plan: (id: string) => ns('plan', id)`（现有 `plans()` listAll 键保持不变）。
2. `web/src/api/hooks.ts`：`usePlans` 之后新增 `PlanListParams` 接口与 `usePlansPage(params)`（`api.plans.list({ page:{number,size}, sort:{by:'created_at',order:'desc'}, filter: status ? {status} : undefined })`）与 `usePlan(planId)`（`api.plans.get({planId})`，`enabled: Boolean(planId)`）。
3. `web/src/components/status-badge.tsx`：`tones` 新增 `plan` 域 `{draft:'secondary', active:'success', archived:'outline', scheduled:'info'}`（供 #5/#9 复用）。
4. 新建 `web/src/features/config/plans/index.tsx`：`PlansPage`（列：名称链接→详情 / key / v+版本 / StatusBadge / 币种 / 计费周期 / 创建时间；状态 Select 筛选 draft|active|scheduled|archived；ServerTable 分页；空态文案）。
5. 新建 `web/src/features/config/plans/plan-detail.tsx`：`PlanDetail`（返回链接、名称+状态徽章+版本、InfoRow 基本信息卡、阶段→价目卡两级视图；`RateCardPriceSummary` 按 free/flat/unit/graduated/volume 渲染，tier 折叠为档数；`EnumBadge domain='plan' kind='priceType'`）。
6. 路由：`web/src/routes/_authenticated/config/plans/index.tsx` 占位替换为 `PlansPage`；新建 `$planId.tsx` 挂 `PlanDetail`；`tsr generate` 重生成 `routeTree.gen.ts`（预期新增 `$planId` 节点）。
7. i18n：`config.plans` 扩展为评论给定完整块（title/description/empty/filter/fields/detail），顶层新增 `plan` 域（status/priceType/price.free），zh-CN 与 en 两份同步。

## 非目标（Non-goals）

- 不做发布/归档/克隆新版本/任何写操作（属 #5：`usePublishPlan`/`useArchivePlan`/`useClonePlanNext`、ConfirmDialog、操作按钮）。
- 不做计划创建/编辑向导（#6–#9）。
- 不改 sidebar（#1 已提供「配置→计划」入口）。
- 不改 v1 `legacy.ts`（克隆 API 属 #5）。
- 不改后端/API spec/SDK。
- 不处理既有 prettier 违规（hooks.ts recharge hunk 等仓库预存问题）。

## SDK 契约核实记录（控制器，2026-08-27）

- `Plans.list(request)` → `ListPlansResponse = PlanPagePaginatedResponse { data: Plan[]; meta: { page: { total } } }`；`Plans.get({planId})` → `Plan`（直接对象，无 data 包裹）。
- `ListPlansQuery = { page?: {size?,number?}, sort?: SortQueryInput, filter?: ListPlansParamsFilter }`；`ListPlansParamsFilter.status: StringFieldFilterExact = string | {eq?,oeq?,neq?}` —— 评论的纯字符串写法类型合法。
- `Plan.status: 'draft'|'active'|'archived'|'scheduled'`；`PlanPhase = { name, description?, key, duration?: string, rateCards: RateCard[] }`；`RateCard = { name, key, feature?: FeatureReference({id}), currency?, billingCadence?, price: Price, paymentTerm, … }`；`Price = PriceFree|PriceFlat|PriceUnit|PriceGraduated|PriceVolume`（oneOf，discriminator `type`；tier 金额均为 Numeric 字符串）——与评论「渲染数据事实」逐项一致。
- 前端现状：`EnumBadge(domain,kind,value)` 已存在（解析 `plan.priceType.<value>`）；`StatusBadge.domain: keyof typeof tones` 加 `plan` 条目即自动扩展；`formatAmount(amount: string|number, currency?: string)`、`formatDateTime` 均存在；`ServerTable` props 与评论用法一致；`config.plans` i18n 现仅 title/description（#1 占位），本次按评论完整块替换扩展；顶层 `plan` 键不存在，无冲突。

## 已知偏差风险（implementer 注意）

1. **路由取参模式**：评论在 `PlanDetail` 内用 `useParams({ from: '/_authenticated/config/plans/$planId' })`；仓库规范（#3 `$featureId.tsx`）是路由文件内 `Route.useParams()` + named `RouteComponent`（含 eslint-disable pragma）传 prop。若评论写法触发 eslint（react-hooks/rules-of-hooks 或 react-refresh）或类型报错，改用仓库规范模式并记录偏差。
2. 路由 `component: PlanDetail` 直接引用命名导入（非 inline 箭头）不会重蹈 #3 的 rules-of-hooks 覆辙；若 lint 仍报，同上处理。
3. e2e 端口卫生：跑 `test:e2e` 前确认 :4173/:9999 无残留进程（#2/#3 台账先例）。

## 任务拆分（SDD）

- Task 1（唯一任务）：上述范围 1–7 一次实施 + 提交（commit `feat(admin): 计划列表与详情（只读）`）。实施后依次：spec 审查 → 质量审查（并行）→ 浏览器 walkthrough → 全分支终审。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- build：tsr generate + tsc + vite 全通过（routeTree 重生成仅增 `$planId` 节点）。
- lint：0 error。
- e2e：既有 2 条 Playwright 冒烟不回归（本任务未改登录/仪表盘路径）。
- 浏览器手测（walkthrough）：`/config/plans` 列表渲染既有计划（名称链接进详情）、状态筛选与分页生效、详情页各阶段→价目卡两级正确、价格类型徽章正确、空态与 404 详情兜底。
- i18n：zh/en 两份键结构程序化比对（含 interpolation 变量一致）。

## 全局约束

- 严格遵循 AGENTS.md（Go 约定本任务不涉及；web 侧遵循 #1–#3 已确立模式：命名导出、`t()` 双语、prettier 新文件格式化、最小 diff、不动无关文件）。
- implementer subagent 禁止：派生子代理、任何 GitHub 写操作、push/merge/close。
- 每步偏差必须记录（#2 SDK 冲突、#3 eslint 模式偏差均有先例：以仓库/SDK 真实契约为准，评论代码冲突时记录并采用可编译的等价形式）。
- 外部副作用（push/merge/close）默认禁止，完成后单独请示用户。
