# Final Whole-Branch Adversarial Review Report — Issue #4（计划列表与详情·只读）

- 终审审查员：最强模型终审（独立子代理，无再派生）
- 审查对象：worktree `/Users/wuyongjun/trea/openmeter-issue-4`，branch `codex/admin-config-04`，全分支 `18309b955..HEAD`（HEAD = `45f89e50da349027daf280c8549763466163ea73`）
- 提交序列（`git log --format='%h %s' 18309b955..HEAD`）：`96e1111c0 docs(admin): issue #4 计划列表与详情实施计划` + `45f89e50d feat(admin): 计划列表与详情（只读）`，共 11 files +608/−9（10 任务文件 +547/−9 与 1 计划文档 +61）
- 审查方式：只读 + 验证命令独立重跑；规范原文经 `gh issue view 4 -R 1123786563/openmeter --json body,comments` 只读获取；未改任何源文件、未派生子代理、无 GitHub 写操作
- 审查日期：2026-08-27（终审）

## 总裁定：**SAFE TO PRESENT**（Critical 0 / Important 0 / Minor 5，全部为记录性/预存性，无必改项）

---

## 一、九角度逐项结论（每角度独立执行并给证据）

### 1. 规范保真 — PASS

**验收标准逐条满足证明：**

- 「列表可筛选状态并分页」：
  - 代码：`web/src/features/config/plans/index.tsx` — 状态 Select（`'all'` + `STATUS_OPTIONS = ['draft','active','scheduled','archived']`），`onValueChange` 同一事件内 `setPage(1)` + `setStatus(...)`；ServerTable 接线 `page/pageSize/total={data?.meta.page.total}/onPageChange`（pageSize 变化重置页码）。
  - 行为证据：walkthrough 场景 b（8 断言）——切「草稿」后**同一次请求**携带 `filter[status]=draft&page[number]=1`（页码重置的直接 wire 证明），行过滤为 3 行草稿、非草稿归零；场景 c（7 断言）——25 条 × pageSize 10 → 3 页，`page[number]=2/3` 请求逐页捕获。
- 「详情正确渲染所有阶段与价目卡结构」：
  - 代码：`plan-detail.tsx` — `plan.phases.map` 每阶段一 Card（`阶段 {{index}} · name` + duration/noDuration），阶段内价目卡 6 列表（价目卡/价格类型/功能/价格/计费周期/Key）；`RateCardPriceSummary` switch 覆盖 free/flat/unit/graduated/volume 全部 5 分支（无 default，编译器强制穷尽）；3 处回退（`card.currency ?? planCurrency`、`card.feature?.id ?? '—'`、`card.billingCadence ?? plan.billingCadence`）逐字到位。
  - 行为证据：walkthrough 场景 d（28 断言）——2 阶段 × 5 价格类型全覆盖，含 cadence/currency 回退验证；场景 g——未知 price.type 走 EnumBadge defaultValue 不崩溃。

**评论处方代码与实现的差异逐条裁定（终审独立复核）：**

| # | 差异 | 终审裁定 | 证据 |
|---|---|---|---|
| 1 | state 类型 `useState<PlanListParams['status']>()` + cast（评论原文 `useState<string \| undefined>`） | **正当·必要** | 评论自身的 `PlanListParams.status` 窄联合（hooks.ts:174 实测）与宽 `string` state 互斥；实现逐字节复用 `orders.tsx:62`（`useState<OrderListParams['status']>()`）与 :199-203（cast）仓库既有模式（终审已并排核对两文件）；前置 spec 审查员已用真实 tsconfig+tsc 程序复现 TS2322 |
| 2 | `plan-detail.tsx` 只导入 `RateCard`、不导入 `Plan` | **正当·必要** | `noUnusedLocals: true`；终审核对该文件确实无 `Plan` 类型注解引用（仅 `RateCard` 用于 props 与 price summary）；type-only 导入零运行时影响 |
| 3 | 格式化适配（import 排序/换行/i18n 缩进/en description 折行） | **可接受（非语义）** | 终审对全部 8 个手写改动文件跑 `prettier --check` → `All matched files use Prettier code style!`（exit 0） |
| 4 | `useParams({ from })` + `component: PlanDetail` 直连（未切换 #3 prop 模式） | **属实·无需处置** | `meter-detail.tsx:37` 同款先例在案；终审 lint（exit 0）+ build（exit 0）零告警 |

其余全部按评论逐字实施：hooks 三段（queryKey/queryFn/enabled）、query-keys 两行、status-badge `plan` 域 4 状态、列表 7 列、两个路由文件（$planId.tsx 6 行逐字）、i18n 两份完整块（28+10 键逐键逐值，含 `{{index}}/{{duration}}/{{count}}`）。commit message 与评论规定逐字一致。

### 2. SDK 契约再验证（独立从 dist/ 重新推导）— PASS

终审不采信任何二手结论，直接从 `api/spec/packages/aip-client-javascript/dist/` 重新推导，全部与实现吻合：

- **操作形状**（`dist/models/operations/plans.d.ts`）：`ListPlansQuery = { page?: {size?; number?}; sort?: SortQueryInput; filter?: ListPlansParamsFilter }`；`ListPlansResponse = PlanPagePaginatedResponse`；`GetPlanRequest = { planId: string }`；`GetPlanResponse = Plan`（直接对象，无 data 包裹）→ hooks 的 `data?.data ?? []` / `data?.meta.page.total` / `usePlan` 直接拿 Plan 用法正确。
- **筛选类型**（`dist/models/types.d.ts:5707`）：`StringFieldFilterExact = string | { eq?; oeq?; neq? }` → `{ status: params.status }` 纯字符串写法类型合法；`ListPlansParamsFilter`（:1825-1830）含 `key/name/status/currency` 四键。
- **排序**（:5816）：`SortQueryInput = { by: string; order?: 'asc' | 'desc' }` → `{ by: 'created_at', order: 'desc' }` 合法且 `created_at` 为文档默认属性。
- **领域模型**：`Plan`（:5403+：id/name/description?/createdAt/updatedAt/key/version/currency/billingCadence/proRatingEnabled/effectiveFrom?/effectiveTo?/status 四值联合/phases/validationErrors?）、`PlanPhase`（:5028：name/description?/**key: string 必填**/duration?: string/rateCards）、`RateCard`（:4821：name/key/feature?: FeatureReference/currency?/billingCadence?: string/price/paymentTerm…）、`Price`（:5775 判别联合五变体；PriceFlat/PriceUnit.amount: string；Graduated/Volume.tiers: PriceTier[]）、`PriceTier`（:2711：upToAmount?/flatPrice?: PriceFlat/unitPrice?: PriceUnit）、`PlanPagePaginatedResponse`（:5559：data: Plan[] + meta）——与 hooks/两组件的每一处取值路径逐一吻合。
- **wire 序列化**（`dist/funcs/plans.js` listPlans）：`toWire({ page, sort: encodeSort(...), filter })` → `toURLSearchParams` → 产出 `page[number]=&page[size]=&sort=created_at desc&filter[status]=` 形态——与 walkthrough 实测编码（`page%5Bnumber%5D=2`、`sort=created_at+desc`、`filter%5Bstatus%5D=draft`）逐串一致；`sdk/plans.d.ts` 类签名 `list(request?, options?)` / `get(request, options?)` 与 hooks 调用一致。
- **穷尽性**：price switch 无 default，五变体逐一对应 SDK 判别联合 → 编译器强制穷尽（build exit 0 即证明）。

### 3. 查询语义 — PASS

- **queryKey 构造/稳定性**：`ns()`（query-keys.ts:8-13）前缀 `['api', currentNamespace ?? '', ...key]`；`plansPage(params)` 的 params 对象每次渲染新建，但 React Query 默认 hashFn 做结构化序列化（稳定键序、undefined 属性剔除）→ 同值同 hash，无无限 refetch 陷阱；与 orders/subscriptions/customers 等全仓库 `(params: object = {}) => ns(x, params)` 模式完全一致。
- **namespace 面**：`namespace-switcher.tsx:46` 切换时 `invalidateQueries()`（全量）→ 新键随 ns 前缀换缓存且强制重取，无跨 namespace 串味。
- **enabled 门**：`usePlan` 的 `enabled: Boolean(planId)`（hooks.ts:196）守卫空 id；该路由 planId 恒有值，门为防御性。
- **筛选-请求一致性**：`setPage(1)`+`setStatus` 同事件批量 → 单次重渲染 → 新 queryKey `{page:1,status:'draft'}` → 新缓存条目必然发新请求；walkthrough 场景 b 的同请求 `filter[status]=draft&page[number]=1` 为直接 wire 证明。
- **缓存面（只读任务无 mutation）**：全局 `staleTime: 10s`（main.tsx:37）、PROD `refetchOnWindowFocus`；同一 planId 二次进入 10s 内命中缓存（键 `['api',ns,'plan',id]` 稳定）；筛选变更换键必发新请求；切回 'all' 10s 内不重发属产品合法行为（walkthrough 已如实记录）。404 重试语义（PROD `failureCount > 3` → 5 次尝试）与 walkthrough 场景 e 的 11 次 GET（1 成功 slow + 5+5 两个 404 id）**精确吻合**。`plans()`（listAll 向导键）与 `plans-page`/`plan` 三键独立，零交叉污染。

### 4. i18n 深检 — PASS（终审自建 AST 脚本 `/tmp/issue4-final-i18n-check.mjs`）

用 web 自带 TypeScript 6.0.3 做 AST 级检查（非正则近似），全部通过：

- zh/en 全文件键树一致：**516 叶键/侧**，无 only-zh / only-en 键；
- 516 键的 `{{var}}` 插值变量逐键一致（zh 与 en）；
- `config.plans` 28 键、顶层 `plan` 10 键，双侧齐备；
- **重复键**：zh-CN.ts 与 en.ts AST 级无重复兄弟键（运行时覆盖无法察觉的场景已用源码级解析排除）；
- **t() 双向解析**：两新组件 33 个字面量 t()（index 12 + detail 21）全部在 zh/en 双侧可解析；1 个模板动态站点（`plan.status.${option}`）确认；
- **动态键全就位**：`plan.status.{draft,active,scheduled,archived}` ×4 + `plan.priceType.{free,flat,unit,graduated,volume}` ×5 = **9/9 双侧存在**（StatusBadge 构造 `${domain}.status.${value}`、EnumBadge 构造 `${domain}.${kind}.${value}`，终审已读 status-badge.tsx:103/133 源码确认键拼接路径）；
- **死键**：38 个新键（28+10）全部有引用（字面量 t() 或经 STATUS_OPTIONS/EnumBadge 的动态路径），零死键；
- tierSummary/phaseIndex/duration 三键插值占位双侧在位。

### 5. 回归面 — PASS

- `git diff 18309b955..HEAD --stat` 全量 = **恰 11 文件**：10 个任务文件（api/hooks +30、query-keys +2、status-badge +6、plans/index +153 新建、plan-detail +223 新建、routes index +2/−7、routes $planId +6 新建、zh +51/−1、en +52/−1、routeTree.gen +22）+ 计划文档 +61。**无范围外文件**。
- **routeTree 零扰动**：终审提取 base 与 HEAD 的全部路由 ID 集合比对——diff 仅为新增 `'/config/plans/$planId'`（8 处 union/接口/注册槽位）与 `'/_authenticated/config/plans/$planId'`（4 处），**零既有路由 ID 被删除或改动**；且终审重跑 `pnpm build`（含 `tsr generate`）后 `git status` 仍净（生成文件与生成器逐字节同步，无陈旧产物）。
- **sidebar 零改动**：`git diff 18309b955..HEAD -- web/src/components/layout/` = 0 行。
- **依赖零改动**：`git diff 18309b955..HEAD -- web/package.json web/pnpm-lock.yaml package.json pnpm-lock.yaml` = 0 行。
- `PlaceholderPage` 仍被 currencies/notification×3/addons/billing-profiles/portal-tokens/tax-codes 等 8+ 占位路由使用，未成死代码；其 JSDoc 中 `config.plans.title` 仅为注释示例。
- 既有 e2e 冒烟（sign-in + customers）2 passed，登录/仪表板路径无回归。

### 6. 跨提交一致性 — PASS

- `96e1111c0`：仅 `docs/superpowers/plans/issue-4-plans-list-detail.md`（+61），message「计划列表与详情实施计划」与内容相符（终审通读：范围 1–7、非目标、SDK 核实记录、偏差风险 3 条——与后续实施逐项对应）。
- `45f89e50d`：10 文件 +547/−9，message 与 Issue 评论规定逐字一致。
- **无 fix-round 提交**（两提交即全部历史），第二提交未引入计划外新面（文件清单与计划文档范围 1–7 + routeTree 再生成完全一致）。
- 工作树净：仅 `?? .superpowers/sdd/issue-4-plans-list-detail/`（按约定不入提交）。

### 7. 安全 — PASS

- 全分支 diff 模式扫描：`eval(`、`new Function`、`dangerouslySetInnerHTML`、`innerHTML`、动态 `import(`、`require(`、`process.env`、常见 secret 形态（sk-/AKIA/ghp_/password/secret/token 赋值）——**全部零命中**（唯一 URL 为计划文档内的 GitHub issue 链接，非外联端点）。
- XSS 面：所有服务端数据（name/key/description/duration/currency/cadence/version）经 React 文本节点渲染；i18n 插值变量均为数字/短枚举（{{index}}/{{duration}}/{{count}}）；无 HTML 注入路径。
- 依赖面：package.json 与 pnpm-lock.yaml 双侧零 diff（见角度 5），无供应链变化。

### 8. 证据审计 — PASS

- **/tmp 时间线自洽**：baseline-build 13:13（改动前基线）→ i18n 脚本 13:16 → build/lint/e2e 三日志 13:17（build 尾部 `✓ built in 285ms`；lint 11 字节即 `$ eslint .` 零输出；e2e `2 passed (6.2s)`）→ commit `45f89e50d` 13:18:04 → walkthrough 13:26–13:45（9 截图 mtime 13:40–13:41、results JSON 13:41）。单调递进，无倒挂。
- **walkthrough wire 与断言互证**：终审直接解析 `/tmp/issue4-walkthrough-results.json`——7 tests / **86 checks 全 pass / 0 fail**（分场景 15+8+7+28+5+17+6 = 86，与报告表格逐行一致）；每场景显式 `pageerror count is zero` 断言在册且全 pass；`requests` 数组的 URL 编码形态（`page%5Bnumber%5D`、`filter%5Bstatus%5D=draft&page%5Bnumber%5D=1` 同请求）与报告 wire 表及控制器先验事实三方一致；26 次 GET、9 截图、7 份 trace 目录均在盘。
- **台账 Ruling 与报告结论一致**：spec PASS 0C/0I/2M = spec-review-report ✓；quality PASS 0C/0I/3M = quality-review-report ✓；walkthrough PASS 86/86 + 3 Minor = 报告 ✓；「recharge hunk 预存」由控制器与质量审查员两方确证后，**终审第三次独立确证**——`git blame 18309b955` 显示 484-488 行属 925f6be4d（base 前提交）；prettier 输出与 HEAD 实文 diff 的全部 5 行分歧集中于 line 485 recharge hunk，**新增 plan hooks 行（170-199）零分歧**。
- 截图指标合理（std 13.77-26.43、唯一色 612-1216、成对差异最小 3.1%，无重复/空白嫌疑）；dist 重建说明与 git 核验（源码树零改动）一致。

### 9. 仓库约定 — PASS

- **命名导出**：`PlansPage`/`PlanDetail`/`usePlansPage`/`usePlan`/`PlanListParams` 全部命名导出（与 `FeaturesPage`/`OrdersPage` 同型）。
- **类型导入 style**：`import type { ColumnDef }`、`import type { Plan }`、`import type { RateCard }` 均为 type-only；无 `any`/`@ts-ignore`/新增 eslint-disable（routeTree.gen.ts 的 `as any` 为生成器固有）。
- **JSDoc 领域意图**：`usePlansPage` 注释说明保留 `usePlans` 的原因（订阅向导 listAll 仍在用）；`RateCardPriceSummary` 注释「tiers collapse to a count」压缩领域语义——符合 AGENTS.md「名称压缩非显然语义的单次 helper 可保留」精神（web 侧沿 #1–#3 先例）。
- **最小 diff**：10 文件纯任务面；预存 prettier 违规（recharge hunk）按最小 diff 不动是正确处置（终审确证其 100% 预存于 base）。
- **模式先例**：状态筛选与 orders.tsx 逐字节同型；`useParams({from})` 与 meter-detail 同型；ServerTable 接线与 features/orders 同构；i18n 双语同步与 #1–#3 一致。

---

## 二、必跑自验证（终审独立执行）

| 命令 | 结果 |
|---|---|
| `cd web && pnpm lint` | **exit 0**（`$ eslint .` 零输出，0 error 0 warning） |
| `node /tmp/issue4-final-i18n-check.mjs`（终审自建 AST 深检脚本） | **ALL CHECKS PASSED**（输出见角度 4，12 项 PASS/INFO 全绿） |
| `git diff 18309b955..HEAD --stat` | 11 files，+608/−9（= 计划文档 +61 与实现 +547/−9） |
| `git log --format='%h %s' 18309b955..HEAD` | 96e1111c0 + 45f89e50d 两提交 |
| （加跑）`pnpm build` | exit 0（`✓ built in 360ms`），build 后 `git status` 净（routeTree 字节同步） |
| （加跑）`prettier --check` 8 个手写改动文件 | All matched（hooks.ts 的唯一分歧 = 预存 recharge hunk，见角度 8） |

## 三、发现分级

### Critical — 0
无。

### Important — 0
无。

### Minor — 5（全部记录性/预存性，无必改项）

1. **（预存·三方确证+终审第四次）`hooks.ts:485` recharge hunk prettier 违规**：多行拆参形态，git blame 定位为 base 前提交 925f6be4d 引入；本分支未触碰该区域，新增行零格式分歧。`pnpm lint` 不含 prettier 门禁，无 CI 拦截。建议后续独立 chore 清理（与 #2/#3 台账记录同源同结论）。
2. **（类型层死分支·处方代码）`plan-detail.tsx:139` `phase.key ?? phaseIndex` 右支按 SDK 类型不可达**（`PlanPhase.key: string` 必填）：Issue 处方的防御式回退，运行时若服务端违约仍正确回退，保留合理，仅记录。
3. **（文档记法）task-1-report 偏差 2 错误码记 TS6196，实际同类条件产出 TS6133**：同属 noUnusedLocals 未使用导入错误类；错误条件本身已被 spec 审查员独立复现，纯文档层面记法差异。
4. **（共享组件预存·走查观察）zh 界面分页文案硬编码英文（'Page x of y'/'Rows per page'，data-table/pagination.tsx）与日期 enUS locale（lib/format.ts）**：均为共享组件既有行为、非本任务回归；与 issue-3 走查发现同族，留待统一 i18n/本地化 chore。
5. **（注释级·无需动作）`placeholder-page.tsx:21` JSDoc 示例仍引用 `config.plans.title`**：plans 已成真实页面，示例略显过时；注释-only、零行为影响，按最小 diff 原则不动。

## 四、总裁定

**SAFE TO PRESENT** —— 九角度全部 PASS。实现与 Issue #4 处方化评论逐项忠实一致（两条代码级偏差均为处方自身缺陷所迫、最小且逐字沿用仓库先例）；SDK 契约经终审独立从 dist/ 重新推导完全吻合；查询/i18n/路由语义经 AST 与 wire 双向验证无缺陷；回归面纯增量（routeTree 仅增 $planId、sidebar/依赖/后端零改动）；四份既有报告与台账 Ruling 交叉一致且关键声明均被终审独立复证。5 条 Minor 全部为预存或记录性质，不构成展示阻塞，无需整改。
