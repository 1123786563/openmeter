# Quality Review Report — Issue #4 Task 1（计划列表与详情·只读）

- 审查员：独立代码质量审查员（spec 符合性由另一位审查员负责）
- 审查对象：worktree `/Users/wuyongjun/trea/openmeter-issue-4`，分支 `codex/admin-config-04`，`96e1111c0..45f89e50d`
- **总裁定：PASS**（Critical 0 / Important 0 / Minor 3）

---

## 一、审查维度逐项结论

### 1. 类型安全 — 通过

- **无 `any`/`as any`/`@ts-ignore`/新增 `eslint-disable`**：对全量 diff 扫描 `eslint-disable|@ts-ignore|@ts-expect-error|@ts-nocheck|: any|as any|as unknown`，唯一命中为 `web/src/routeTree.gen.ts` 的 `} as any)`（生成器固有风格，hunk 前后行均为同款 `as any`，非手写代码）。
- **唯一 cast 有逐字先例**：`web/src/features/config/plans/index.tsx:112` 的 `(value as PlanListParams['status'])` 与 `web/src/features/commerce/orders.tsx:199-203` 的 `(value as OrderListParams['status'])` 完全同型，且前置 `value === 'all'` 守卫使 cast 收窄安全。
- **switch 穷尽性**：`plan-detail.tsx:46-77` 对 `card.price.type` 的 5 分支（free/flat/unit/graduated/volume）与 SDK 判别联合 `Price = PriceFree | PriceFlat | PriceUnit | PriceGraduated | PriceVolume`（`models/types.d.ts:5775`）一一对应，无 default 分支 → 编译器强制穷尽。`flat`/`unit` 的 `amount: string` 与 `formatAmount(amount: string | number, ...)` 签名匹配。
- **空值路径全部与 SDK 一致**：`card.currency ?? planCurrency`（`currency?: BillingCurrencyCode`）、`card.feature?.id ?? '—'`（`feature?: FeatureReference`）、`card.billingCadence ?? plan.billingCadence`（可选覆盖语义，SDK 注释明确）、`phase.duration ? … : …`（`duration?: string`）、`plan.description ? <InfoRow/> : null`。`Plan.status` 联合与 `PlanListParams['status']`、status-badge `tones.plan`、i18n `plan.status.*` 四处完全同构（'draft'|'active'|'archived'|'scheduled'）。
- **API 调用形状核对**：`usePlansPage` 的 `{ page: { number, size }, sort: { by: 'created_at', order: 'desc' }, filter: { status } }` 与 `ListPlansQuery`（`models/operations/plans.d.ts:3-21`：`sort.by: string`、'created_at' 为文档默认值、`filter.status?: StringFieldFilterExact`）逐字段匹配；`data?.meta.page.total`（`PageMeta.total: number` 必填）经可选链得 `number | undefined`，恰好满足 ServerTable 的 `total: number | undefined` prop。

### 2. React 质量 — 通过

- **hooks 规则**：无条件 hook；`usePlan` 的 `enabled: Boolean(planId)` 守卫空 id。
- **key 使用**：`SelectItem key={option}`（index.tsx:134）、`phase.key ?? phaseIndex`（plan-detail.tsx:139）、`card.key`（plan-detail.tsx:173）均符合 brief 预期；表内 header/cell key 由 ServerTable 统一处理。
- **queryKey 含 params**：`queryKeys.plansPage(params)` 把整个 params 对象入键，且 `ns()` 前缀嵌 namespace（`query-keys.ts:8-12`），切页/切筛选/切 namespace 均不串缓存。
- **受控组件**：筛选 Select `value={status ?? 'all'}` 受控；`onValueChange` 先 `setPage(1)` 再 `setStatus`（index.tsx:106-113），切筛选重置页码 ✅；pageSize 变化重置页码（index.tsx:96-101，与 features 页同款回调）。
- **不必要重渲染**：`columns` 每渲染重建，与 features/orders 既有模式一致，React Compiler/eslint 均无告警（lint 0 输出）。

### 3. 仓库约定 — 通过

- **命名导出**：`PlansPage`/`PlanDetail`/`usePlansPage`/`usePlan`/`PlanListParams` 全部命名导出，与 `FeaturesPage`/`FeatureDetailPage` 同型。
- **i18n 双语 + 键完整性**（程序化验证，见下节自查输出）：`config.plans` 28 键、顶层 `plan` 10 键，zh/en 键树与插值变量逐键一致；全文件 516/516 键无重复兄弟键；新组件 37 个 t() 引用（16+21）在两份 locale 全部可解析；反向死键检查 38 个新键全部有引用（literal 或 `STATUS_OPTIONS`/`EnumBadge domain='plan' kind='priceType'` 动态路径）。
- **prettier**：8 个手写改动文件中 7 个 clean；`hooks.ts` 的违规（485-488 行 recharge hunk 多行拆参）经严格同条件复检（base 内容置于 `web/src/` 下临时文件名、同 config 同插件跑 `--check`）确认**在 base 96e1111c0 即已存在**，实现者「预存不动」的最小 diff 处置属实且正确。注：`pnpm lint`（eslint）不检查 prettier 格式，该债务无 CI 门禁拦截，留待后续 chore。
- **最小 diff**：`git diff 96e1111c0..45f89e50d --stat` 全量核对为恰好 10 文件 +547/-9（api/hooks +30、query-keys +2、status-badge +6、plans/index +153、plan-detail +223、两路由 +2/-7 与 +6、两 locale +51/+52、routeTree.gen +22），无任何无关文件、无 package.json/依赖改动、`pnpm-lock.yaml` 无改动。
- **模式一致性**（对照 #2/#3 `features/config/features/`）：列表页 state 管理/ServerTable 接线/onPageChange 回调/columns 风格与 `features/index.tsx` 同构；状态筛选 Select 与 `orders.tsx` 同构；详情页 loading skeleton + 空数据兜底与 `feature-detail.tsx` 同构；路由接线采用 `component: PlanDetail` + `useParams({ from })`（`meter-detail.tsx:37` 同款先例），且比 features 路由的 `Route.useParams()` + `eslint-disable react-refresh/only-export-components` 包装更干净（无需 disable）。

### 4. 健壮性 — 通过

- **列表页**：`isLoading` → ServerTable 5 行 skeleton；空页 → `emptyMessage={t('config.plans.empty')}`；`isFetching` → 表格容器 opacity-70；`data?.data ?? []` 防未就绪。
- **详情页**：`isLoading` → 标题+卡片两级 Skeleton；`!plan`（含 404）→ `t('config.plans.detail.notFound')` 文案兜底。
- **分页边界**：`total === undefined` 时 ServerTable `pageCount: 1` 且不渲染分页条（`server-table.tsx:60-61,135`），行为明确。
- **时区/日期**：`formatDateTime`（`lib/format.ts`）对 null/undefined/Invalid Date 均回退 `'—'`，date-fns `PPpp` 本地时区渲染；`formatAmount` 对 NaN 字符串回退原样输出，`Intl` 异常有 try/catch。
- **cadence/currency 裸串渲染**：`{card.billingCadence ?? plan.billingCadence}` 与 `subscription-detail.tsx:108` 逐字一致，`{candidate.currency} · {candidate.billingCadence}`（create-subscription-dialog.tsx:122）为先例，属既有模式而非新引入的不一致。

### 5. 性能/安全 — 通过

- 无新增依赖（package.json/lockfile 零改动）；无 `eval`/`dangerouslySetInnerHTML`/`innerHTML`（全量 diff 扫描为空）；无秘密/凭据；所有动态数据经 React 文本节点或 `t()` 插值（`{{index}}`/`{{duration}}`/`{{count}}` 均为数字/短枚举），无 XSS 注入面。

### 6. 可维护性 — 通过

- `InfoRow`（复用 6 次）与 `RateCardPriceSummary`（单次使用）均有语义命名；后者 JSDoc「Human-readable price summary per rate card; tiers collapse to a count.」准确表达了领域压缩意图，符合 AGENTS.md「名称压缩了非显然领域语义的单次 helper 可保留」。
- `usePlansPage` JSDoc 明确说明保留 `usePlans` 的原因（订阅向导 listAll 仍在用——实际 3 处订阅页引用，已核实），前瞻性清晰。
- 无死代码：`PlaceholderPage` 引用随占位页移除而清除，`PlaceholderPage` 组件本身仍被其他占位路由使用（未成死代码）；导出项全部被消费（`PlanListParams` 被 index.tsx 类型引用）。

### 7. 回归面 — 通过

- **routeTree.gen.ts**：diff 纯增量，全部 hunk 仅新增 `AuthenticatedConfigPlansPlanIdRoute` 相关行（import/update 定义/5 处类型接口/module declaration/children 注册），零删除、零既有路由改动。
- **再生成同步性**：本次审查重跑 `pnpm build`（内含 `tsr generate`）后 `git status` 干净——提交的生成文件与生成器输出逐字节一致，无陈旧生成代码。
- **sidebar/既有页面零改动**：diff --stat 全量清单无 sidebar、无既有 feature/commerce 页面文件；实现者 e2e（sign-in + customers 冒烟 2 passed）与本次独立复跑 lint/build 全绿。

---

## 二、自查命令输出摘要（独立复跑）

| 命令 | 结果 |
|---|---|
| `cd web && pnpm lint` | exit 0，无输出（0 error 0 warning） |
| `cd web && pnpm build` | exit 0，`✓ built in 320ms`；build 后 `git status` 干净（routeTree.gen 再生成零 diff） |
| `node /tmp/issue4-quality-i18n-check.mjs` + 死键补充脚本 | `config.plans` 28 键树一致、插值一致；`plan` 10 键树一致、插值一致；全文件 zh/en 516/516 键、两文件源级无重复兄弟键；两组件 16+21 个 t() 键在 zh/en 全部解析；38 个新键无死键（含动态键路径核实）。脚本留存 `/tmp/issue4-quality-i18n-check.mjs` |
| `git diff 96e1111c0..45f89e50d --stat` | 10 files, +547/-9，与实现者报告清单逐项一致 |
| `prettier --check`（9 个改动文件） | 7 文件 all matched；`hooks.ts` 违规经 base 同条件复检确认为预存（96e1111c0 即违规），新增代码本身 clean |
| 禁用模式 diff 扫描 | 手写代码零命中（仅 routeTree.gen.ts 生成代码内 `as any`） |

## 三、发现清单

### Critical（必须修）— 0

无。

### Important（应修）— 0

无。

### Minor（可接受，点名记录）— 3

1. **`plan-detail.tsx:139` `phase.key ?? phaseIndex` 的 `??` 右支按类型不可达**：SDK `PlanPhase.key: string` 为必填。该防御式回退为 issue 处方代码且 brief 明确预期此模式，运行时若服务端缺 key 仍能正确回退——保留合理，仅记录类型层面的死分支事实。
2. **`hooks.ts:485-488` 预存 prettier 违规未修**：recharge hunk 多行拆参超出行宽策略。经同条件复检确认 base 已违规，本任务按最小 diff 不动是正确处置；但 `pnpm lint` 不含格式门禁，该债务无 CI 拦截，建议后续独立 chore 一并清理。
3. **路由接线选了两种既有模式之一**：`$planId.tsx` 直接 `component: PlanDetail` + 组件内 `useParams({ from })`（meter-detail 先例），而非 features 路由的 `Route.useParams()` + props + `eslint-disable` 包装（feature-detail 先例）。所选模式免除了 disable 注释、类型同样收窄（build 零错），两模式在仓库并存属实——不构成缺陷，提示后续任务保持同域内自洽即可。

## 四、总裁定

**PASS** — 0 Critical / 0 Important / 3 Minor。代码类型安全、模式与 #1–#3 先例一致、i18n 键树程序化验证全绿、回归面纯增量且生成文件同步，三项验证命令独立复跑全部通过。Minor 项均为记录性质，无需阻塞合入。
