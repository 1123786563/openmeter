# 管理端「配置域」四块界面实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 web/ 管理端新增「配置」分组，交付计划+功能+附加组件、通知中心、货币与税码、应用/门户/账单档案/应收周期四大块的完整管理界面。

**Architecture:** 沿用既有模式：`@openmeter/client`（v3 SDK，`web/src/api/client.ts` 单例注入 Bearer + X-Namespace）为主，v1 独有端点在 `web/src/api/legacy.ts` 手写补齐；页面按 `features/<domain>/` + `routes/_authenticated/` 文件路由组织；文案全部 i18n（zh-CN + en）。无后端改动（共识：接受 API 现状）。

**Tech Stack:** React 19 + TanStack Router/Query + react-hook-form + zod + shadcn/ui + i18next。

## Global Constraints

- 文案全部走 i18n，zh-CN 与 en 两份资源同步维护；术语用 `CONTEXT.md` 词表（计划/阶段/价目卡/功能/附加组件/通知渠道/通知规则/自定义货币/税码/账单档案/门户令牌/应收周期/应用）。
- 所有 API 请求经 `web/src/api/client.ts` / `legacy.ts`（自动注入 Authorization 与 X-Namespace）。
- 侧边栏新增「配置」分组（`web/src/components/layout/data/sidebar-data.ts`），现有五组不动。
- 写操作一律带确认弹窗（ConfirmDialog）或明确后果提示；删除失败/受限错误用 toast 原文透出。
- v3 响应字段 snake_case（SDK 已转 camel），v1 手写层保持与 spec camelCase 一致。
- 每任务完成即验证：`pnpm build`、`pnpm lint`、`pnpm test:e2e`（既有 2 条冒烟不得回归）；gofmt 不涉及。
- 端点能力以 `api/v3/openapi.yaml` / `api/openapi.yaml` 为准，禁止臆造字段；v3 标注 x-unstable/x-internal 不影响前端使用。

## 事实速查（已核实，实施时直接引用）

- **通知**：渠道仅 WEBHOOK（name/url/customHeaders/signingSecret/disabled），CRUD 齐；规则 4 类型（entitlements.balance.threshold: thresholds[1-10]+features、entitlements.reset: features、invoice.created、invoice.updated），公共 name/disabled/channels[]，有 `POST /rules/{id}/test`；事件只读+`POST /events/{id}/resend`。
- **货币**：`GET /openmeter/currencies`（filter[type]=custom、expand=cost_basis）、`POST /openmeter/currencies/custom`（name/precision/decimal_mark/thousand_separator/code[4-24]/symbol?）、cost-bases `GET|POST /custom/{id}/cost-bases`（fiat_code/rate/effective_from?）；**无更新/删除**。法币列表 v1 `GET /api/v1/info/currencies`。
- **税码**：CRUD（create: name/key/app_mappings[{app_type: sandbox|stripe|external_invoicing, tax_code}]；upsert 无 key）+ `GET|PUT /openmeter/defaults/tax-codes`（invoicing_tax_code、credit_grant_tax_code）。
- **Apps**：v3 `GET /app-catalog`、`POST /app-catalog/install`（oneOf stripe{api_key}/sandbox/external_invoicing）、`GET|PUT|DELETE /apps/{id}`；无 OAuth。
- **门户令牌**：v1 `POST /api/v1/portal/tokens`（必填 subject，可选 allowedMeterSlugs 等）响应含一次性明文 `om_portal_...`；`GET` 列表（无明文）；`POST /tokens/invalidate`。
- **账单档案**：v3 `GET|POST /openmeter/profiles`、`GET|PUT|DELETE /{id}`；create 必填 name/supplier/workflow/apps/default；app 关联创建后不可变；删除受限（非默认、无 customer override 引用）。
- **应收/线下**：`GET /customers/{id}/receivable-periods`（cursor 分页）；`PUT .../receivable-periods/{periodId}/external-invoice`（idempotency_key+invoice_number 必填）；`POST /customers/{id}/offline-payments`（idempotency_key/amount_fen/currency/external_reference/received_at 必填，receivable_period_id/note 可选）；**线下支付无列表端点**。
- **计划**：v3 create（name/key/currency/billing_cadence/phases[≥1]，phase: name/key/rate_cards/duration?（缺省=无限期且仅最后阶段））；rate card: name/key/price（oneOf free/flat/unit/graduated/volume），可选 feature/billing_cadence(null=一次性)/unit_config 等；`PUT /{id}`、archive、publish；v1 `POST /api/v1/plans/{planIdOrKey}/next` 克隆新 draft。
- **功能**：v3 CRUD（PUT update 仅 v3）+ `POST /{featureId}/cost/query`。**附加组件**：v3 CRUD + archive/publish；plan addons `GET|POST /plans/{planId}/addons`、`GET|PUT|DELETE /{planAddonId}`。

---

### Task 1: 「配置」分组骨架与路由占位

**Files:**
- Modify: `web/src/components/layout/data/sidebar-data.ts`
- Create: `web/src/features/config/`（目录）、`web/src/routes/_authenticated/config/{plans,features,addons,notification/{channels,rules,events},currencies,tax-codes,apps,portal-tokens,billing-profiles}/index.tsx`（占位）
- Modify: `web/src/i18n/locales/zh-CN.ts`、`en.ts`

**Interfaces:**
- Produces: 路由 `/config/*` 与 i18n 命名空间 `config.*`；后续所有任务往这些路由填真页面。

**Steps:**
- [ ] sidebar-data.ts 新增分组 `sidebar.groups.config`，条目：计划 `/config/plans`、功能 `/config/features`、附加组件 `/config/addons`、通知 `/config/notification/channels`、货币 `/config/currencies`、税码 `/config/tax-codes`、应用 `/config/apps`、门户令牌 `/config/portal-tokens`、账单档案 `/config/billing-profiles`（lucide 图标：ListChecks/Blocks/Puzzle/Bell/Coins/ReceiptText/Plug/KeyRound/FileText）。
- [ ] 每路由挂 placeholder-page（复用 `web/src/components/placeholder-page.tsx` 模式），i18n title/description 各配 zh/en。
- [ ] 命令面板（command-menu 消费 sidebar-data，无需另改）验证可导航。
- [ ] 验证：`pnpm build && pnpm lint && pnpm test:e2e`；浏览器：侧边栏出现「配置」分组，全部条目可点开占位页。
- [ ] Commit: `feat(admin): 配置分组骨架与九个配置路由占位`

### Task 2: 功能目录（列表/创建/编辑/删除/详情+成本查询）

**Files:**
- Modify: `web/src/api/hooks.ts`、`web/src/api/query-keys.ts`
- Create: `web/src/features/config/features/index.tsx`（ServerTable 列表+新建/编辑 dialog+删除 confirm）、`feature-detail.tsx`（信息+成本查询区）、`feature-form-dialog.tsx`
- Modify: `web/src/routes/_authenticated/config/features/index.tsx`、`$featureId.tsx`（新建）、i18n 两份

**Interfaces:**
- Produces hooks: `useFeatures(params?)`、`useFeature(id)`、`useCreateFeature()`、`useUpdateFeature()`、`useDeleteFeature()`、`useFeatureCostQuery(id, body)`（v3 SDK features 域）；query key `features(params)`/`feature(id)`。
- 表单字段：name（1-256）、key（^[a-z0-9]+(?:_[a-z0-9]+)*$，创建后不可改）、description?。
- 成本查询：客户选择器（复用 `CustomerPicker`）+ 时间窗 → 结果表。

**Steps:**
- [ ] hooks + query-keys（模式照抄 `useRechargeProducts`：mutation onSuccess `queryClient.invalidateQueries`）。
- [ ] 列表页（key/名称/描述/创建时间；搜索 name）；新建/编辑 dialog（react-hook-form+zod，编辑时 key 只读）；删除 confirm。
- [ ] 详情页 + 成本查询（`POST /features/{id}/cost/query`，参数与展示列以 spec `QueryFeatureCostRequest` 为准）。
- [ ] i18n 全量；验证三连 + 浏览器手测（用 mock 后端或真实后端建两个 feature）。
- [ ] Commit: `feat(admin): 功能目录管理（CRUD+成本查询）`

### Task 3: 计划列表/详情/状态操作/克隆新版本

**Files:**
- Modify: `web/src/api/hooks.ts`（已有 `usePlans`/`useSubscriptions` 所在文件）、`query-keys.ts`
- Create: `web/src/features/config/plans/index.tsx`（列表）、`plan-detail.tsx`（阶段/价目卡树状视图+操作）、`web/src/api/legacy.ts`（补 `clonePlanNextVersion(planIdOrKey)` → v1 `POST /api/v1/plans/{idOrKey}/next`）
- Modify: 路由 `config/plans/index.tsx`、新增 `$planId.tsx`；i18n

**Interfaces:**
- Produces: `usePlans`（已有，确认分页参数）、`usePlan(id)`、`usePublishPlan()`、`useArchivePlan()`、`useClonePlanNext()`（v1）；详情页信息区渲染 phase→rateCards 两级（价格按类型徽章：free/flat/unit/graduated/volume）。
- 状态操作仅对 draft 可发布、对 published 可归档（按钮可用性按 `status` 字段；克隆按钮对非 draft 显示）。

**Steps:**
- [ ] legacy.ts 加 clonePlanNextVersion（v1，camelCase 响应）。
- [ ] 列表（name/key/版本/状态徽章/币种/周期；状态筛选）。
- [ ] 详情（基本信息、阶段列表：每阶段 duration/价目卡表：类型/名称/价格摘要、操作按钮发布/归档/克隆新版本——均 ConfirmDialog）。
- [ ] i18n；验证三连 + 浏览器（用既有 WeKnora 计划或 API 建的 e2e 计划走查：克隆→新 draft 出现在列表）。
- [ ] Commit: `feat(admin): 计划列表/详情/发布归档/克隆新版本`

### Task 4: 计划创建/编辑向导（基础：free/flat/unit）

**Files:**
- Create: `web/src/features/config/plans/plan-form-wizard.tsx`（三步：基本信息→阶段列表→价目卡编辑）
- Modify: `web/src/api/hooks.ts`（`useCreatePlan`、`useUpdatePlan`）、列表页加「新建计划」按钮、详情页对 draft 加「编辑」；i18n

**Interfaces:**
- 向导状态形状（zod）：`{ name, key, currency(默认从客户常用或 CNY), billingCadence: 'P1M'|'P1Y', description?, phases: [{ key, name, duration?: ISO8601, rateCards: [{ key, name, type: 'flat_fee'|'usage_based', feature?, billingCadence: 'P1M'|'P1Y'|null(一次性), price: PriceForm }] }] }`；`PriceForm = { kind:'free' } | { kind:'flat', amount } | { kind:'unit', amount }`（本任务三种，阶梯在 Task 5 扩展为 `kind:'tiered', tiers:[{first_unit,last_unit?,unit_amount,flat_amount?}], mode:'graduated'|'volume'`）。
- 提交映射：flat_fee→price {type:'flat'|'free'}，usage_based→feature 必选+price {type:'unit'}（字段名以 v3 `CreatePlanRequest` 为准）。
- 阶段校验：仅最后一个 phase 可省略 duration；rate_cards ≥1。

**Steps:**
- [ ] 向导步骤 1 基本信息（表单+zod）；步骤 2 阶段增删改（动态 useFieldArray）；步骤 3 每阶段价目卡行编辑（类型切换联动表单、feature 选择器复用 Task 2 数据）。
- [ ] 编辑模式：仅 draft 计划可编辑（回填同结构，PUT 提交）。
- [ ] i18n；验证三连 + 浏览器全流程：向导建计划（flat+unit 各一张价卡）→ 列表出现 → 发布。
- [ ] Commit: `feat(admin): 计划创建/编辑向导（free/flat/unit）`

### Task 5: 阶梯价格（graduated/volume）编辑器

**Files:**
- Modify: `plan-form-wizard.tsx`（PriceForm 扩展 tiered 变体）、i18n

**Interfaces:**
- `PriceForm` 增加 `{ kind:'tiered', mode:'graduated'|'volume', tiers: [{ firstUnit, lastUnit?: number|null, unitAmount, flatAmount? }] }`；校验：首档 firstUnit=0、各档区间连续无重叠（lastUnit+1=下档 firstUnit）、末档 lastUnit 空=无限。

**Steps:**
- [ ] 阶梯行编辑器（增删行、区间连续性 zod refine、错误定位到行）。
- [ ] 映射到 spec 的 graduated/volume price 结构（字段名以 `Price` oneOf 为准）。
- [ ] i18n；验证三连 + 浏览器建含阶梯价的计划并发布。
- [ ] Commit: `feat(admin): 计划向导支持阶梯价（graduated/volume）`

### Task 6: 附加组件（独立列表 + 计划内管理）

**Files:**
- Modify: `web/src/api/hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/addons/index.tsx`（列表+CRUD dialog+归档/发布）、`web/src/features/config/plans/plan-addons-tab.tsx`（计划详情新 tab）
- Modify: 计划详情页加 tab；i18n

**Interfaces:**
- hooks: `useAddons(params)`、`useAddon(id)`、`useCreateAddon`/`useUpdateAddon`/`useDeleteAddon`、`useArchiveAddon`/`usePublishAddon`；plan addons: `usePlanAddons(planId)`、`useCreatePlanAddon(planId)`、`useUpdatePlanAddon`、`useDeletePlanAddon`。
- 附加组件表单：instancePolicies/price 结构以 v3 `CreateAddonRequest` 为准（实施时读 spec 后定，价格复用 Task 4/5 的 PriceForm 组件）。

**Steps:**
- [ ] 独立列表页（CRUD+归档/发布）。
- [ ] 计划详情 tab：列出该计划的 addons + 增删改（复用 PriceForm）。
- [ ] i18n；验证三连 + 浏览器走查两处。
- [ ] Commit: `feat(admin): 附加组件管理与计划内附加组件 tab`

### Task 7: 通知渠道（Webhook CRUD）

**Files:**
- Modify: `web/src/api/legacy.ts`（v1 notification channels）、`hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/notification/channels.tsx`（列表+新建/编辑 dialog+删除/禁用）
- Modify: 路由 `config/notification/channels/index.tsx`；i18n

**Interfaces:**
- Channel 字段：name、url（https 校验）、customHeaders（键值对动态行）、signingSecret?（whsec_ 前缀展示）、disabled。
- hooks: `useNotificationChannels()`、`useCreateChannel`/`useUpdateChannel`/`useDeleteChannel`（v1 camelCase）。

**Steps:**
- [ ] legacy 层 + hooks。
- [ ] 列表（名称/URL/禁用徽章/操作：编辑、禁用切换、删除）+ dialog 表单。
- [ ] i18n；验证三连 + 浏览器建一个 webhook 渠道。
- [ ] Commit: `feat(admin): 通知渠道（Webhook）管理`

### Task 8: 通知规则与事件流

**Files:**
- Modify: `legacy.ts`、`hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/notification/rules.tsx`（列表+按类型创建/编辑表单+启停+test）、`events.tsx`（只读流+筛选+resend）
- Modify: 路由两个 index；i18n

**Interfaces:**
- 规则类型选择器（4 种）→ 动态字段：balance.threshold（thresholds 数组编辑 1-10 + features 多选（复用 Task 2 features 数据））、reset（features 多选）、invoice.created/updated（无额外字段）；公共：name、channels 多选（Task 7 数据）、disabled。
- 事件页：过滤 from/to/rule/channel；行展开 deliveryStatus；resend 按钮（可选渠道）。

**Steps:**
- [ ] 规则 hooks（v1）+ 列表 + 类型化表单（oneOf 分支渲染）+ 启停 + `POST /rules/{id}/test` 按钮（结果 toast）。
- [ ] 事件流页（分页/过滤/resend confirm）。
- [ ] i18n；验证三连 + 浏览器：建规则（阈值型）→ test → 事件页可见。
- [ ] Commit: `feat(admin): 通知规则（4 类型）与事件流`

### Task 9: 货币（法币列表+自定义创建+cost-bases）

**Files:**
- Modify: `hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/currencies/index.tsx`（双 tab：法币/自定义）、`custom-currency-dialog.tsx`、`cost-bases-panel.tsx`
- Modify: 路由；i18n

**Interfaces:**
- hooks: `useCurrencies(params)`（filter[type]、expand=cost_basis）、`useCreateCustomCurrency()`、`useCostBases(currencyId)`、`useCreateCostBasis(currencyId)`；法币 v1 `useFiatCurrencies()`。
- 自定义货币表单：name、code（4-24，zod 校验不与法币冲突——前端用法币列表校验）、precision、decimalMark、thousandSeparator、symbol?。
- cost-bases 面板：列表（fiat/rate/生效期）+ 追加表单（fiat_code 下拉（法币）、rate、effective_from?）。
- 页面明示「自定义货币创建后不可编辑/删除（API 限制）」。

**Steps:**
- [ ] hooks + 法币 tab + 自定义 tab（列表+创建 dialog）。
- [ ] cost-bases 面板（挂在自定义货币行展开或详情 dialog）。
- [ ] i18n；验证三连 + 浏览器：建 CREDIT 类自定义货币+追加一条 USD cost-basis。
- [ ] Commit: `feat(admin): 货币管理（法币/自定义/cost-bases）`

### Task 10: 税码（CRUD+组织默认）

**Files:**
- Modify: `hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/tax-codes/index.tsx`（列表+CRUD+默认设置卡片）
- Modify: 路由；i18n

**Interfaces:**
- hooks: `useTaxCodes(includeDeleted?)`、`useCreateTaxCode`/`useUpsertTaxCode`/`useDeleteTaxCode`、`useOrgDefaultTaxCodes()`/`useUpdateOrgDefaultTaxCodes()`。
- 表单：name、key（创建必填，编辑隐藏）、description?、appMappings 动态行（app_type 枚举三值 + tax_code 文本）。
- 默认设置卡片：开票税码 / 额度发放税码 两个下拉（数据=税码列表）+ 保存。

**Steps:**
- [ ] hooks + 列表 + 表单 dialog + 删除。
- [ ] 默认税码卡片（GET/PUT defaults）。
- [ ] i18n；验证三连 + 浏览器走查。
- [ ] Commit: `feat(admin): 税码管理与组织默认税码设置`

### Task 11: 应用（列表/目录/安装/卸载/换 Key）

**Files:**
- Modify: `hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/apps/index.tsx`（已装列表+目录区+安装 dialog+卸载）、`stripe-key-dialog.tsx`
- Modify: 路由；i18n

**Interfaces:**
- hooks: `useApps()`、`useAppCatalog()`、`useInstallApp()`（oneOf：stripe{name, createBillingProfile?, apiKey} / sandbox{name} / external_invoicing{name}）、`useUpdateApp()`、`useUninstallApp()`。
- 已装列表：名称/类型徽章/capabilities/默认能力徽章/操作（Stripe：更换 API Key dialog；通用：卸载 confirm）。
- 目录：浏览 app-catalog，按类型给安装表单分支。

**Steps:**
- [ ] hooks + 已装列表 + 卸载 + Stripe 换 Key。
- [ ] 目录 + 安装 dialog（stripe 分支显示 API Key 密码框 + createBillingProfile 开关）。
- [ ] i18n；验证三连 + 浏览器（sandbox 安装/卸载可真实验证；Stripe 用假 Key 走到报错 toast 即可）。
- [ ] Commit: `feat(admin): 应用集成管理（安装/卸载/Stripe 换 Key）`

### Task 12: 门户令牌（发放一次性明文/列表/失效）

**Files:**
- Modify: `legacy.ts`（v1 portal tokens）、`hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/portal-tokens/index.tsx`（客户维度）
- Modify: 路由；i18n

**Interfaces:**
- hooks: `usePortalTokens(limit)`、`useCreatePortalToken()`、`useInvalidatePortalToken()`。
- 发放 dialog：CustomerPicker + allowedMeterSlugs 多选（meters 数据）；成功后**一次性明文弹窗**（`om_portal_...`，复制按钮 + 「关闭后无法再次查看」警示）。
- 列表：subject/创建时间/允许的 meters/失效操作。

**Steps:**
- [ ] legacy 层 + hooks + 列表 + 发放 + 一次性明文弹窗（复制用 navigator.clipboard，降级提示手动复制）。
- [ ] i18n；验证三连 + 浏览器：为 demo 客户发一个 token，明文可见且列表无明文。
- [ ] Commit: `feat(admin): 门户令牌发放与失效`

### Task 13: 账单档案（CRUD）

**Files:**
- Modify: `hooks.ts`、`query-keys.ts`
- Create: `web/src/features/config/billing-profiles/index.tsx`（列表+创建/编辑 dialog+删除）
- Modify: 路由；i18n

**Interfaces:**
- hooks: `useBillingProfiles()`、`useBillingProfile(id)`、`useCreateBillingProfile`/`useUpdateBillingProfile`/`useDeleteBillingProfile`。
- 表单：name、supplier（BillingParty 嵌套：名称/地址行/税号等，字段以 v3 spec 为准）、workflow（readOnly 展示）、apps（tax/invoicing/payment 关联选择——**创建后不可变**，编辑时禁用并提示）、default 开关（唯一默认逻辑由后端裁决，错误透出）。

**Steps:**
- [ ] hooks + 列表 + 创建 dialog。
- [ ] 编辑（apps 禁用态）+ 删除（受限错误 toast 原文）。
- [ ] i18n；验证三连 + 浏览器走查。
- [ ] Commit: `feat(admin): 账单档案管理`

### Task 14: 客户详情「应收周期」tab + 登记表单

**Files:**
- Modify: `web/src/features/customers/customer-detail.tsx`（新 tab）
- Create: `web/src/features/customers/receivable-periods-tab.tsx`、`offline-payment-dialog.tsx`、`external-invoice-dialog.tsx`
- Modify: `hooks.ts`、`query-keys.ts`、i18n

**Interfaces:**
- hooks: `useReceivablePeriods(customerId, cursor?)`、`useCreateOfflinePayment(customerId)`、`useUpdateExternalInvoice(customerId, periodId)`。
- 周期列表：期间/credits_consumed/应付/已付/币种/状态徽章/closedAt；行操作：登记外部发票（invoice_number 必填 + url/issuer/issued_at?）；页首「登记线下支付」dialog（幂等键自动生成 UUID 可改、金额（元↔fen 换算，沿用充值产品口径）、币种、external_reference、received_at 日期、receivable_period_id 下拉可选、note?）。
- 页面明示「线下支付无列表端点，已缴情况见周期已付金额」。

**Steps:**
- [ ] hooks + 周期 tab（cursor 分页「加载更多」，复用事件页模式）。
- [ ] 两个登记 dialog（确认后果文案）。
- [ ] i18n；验证三连 + 浏览器：对 demo 客户登记一笔线下支付 → 周期已付金额变化。
- [ ] Commit: `feat(admin): 客户应收周期与线下支付登记`

### Task 15: 全链路验收（浏览器 E2E 扩展 + 回归）

**Files:**
- Modify: `web/e2e/smoke.spec.ts`（新增配置域冒烟：登录→配置→任一新页渲染）
- 验证文档：`web/README.md` 补配置域说明

**Steps:**
- [ ] 扩展 Playwright 冒烟（mock /api 响应进入计划或功能页断言渲染）。
- [ ] 浏览器真实全链路走查四大块（含各一写操作），截图留档。
- [ ] 全量回归：`pnpm build/lint/test:e2e` + `go build ./...`（确认无后端误改）。
- [ ] Commit: `test(admin): 配置域冒烟与全链路验收`

## Self-Review

- 覆盖：共识表 10 行全部映射到 Task 1-14；验收=Task 15。货币无删改/线下无列表/门户令牌一次性明文三处 API 边界均已在对应任务内显式落地。
- 占位符：无 TBD；「以 spec 为准」处均给出了准确查证位置（v3 openapi.yaml schema 名）。
- 类型一致性：PriceForm 在 Task 4 定义、Task 5 扩展、Task 6 复用；hooks 命名风格与既有 `useRechargeProducts` 对齐。
