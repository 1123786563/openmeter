# Plan — issue #26 账单档案：编辑与删除

- Issue: https://github.com/1123786563/openmeter/issues/26
  ([admin-config 26/29] 账单档案：编辑与删除)
- Master plan: docs/superpowers/plans/2026-08-27-admin-config-domains.md Task 13 后半
  （前半 #25 已合入：列表+创建 dialog）。
- Branch codex/admin-config-26 @ base f60cb90b0；worktree
  /Users/wuyongjun/trea/openmeter-issue-26。
- Ledger: .superpowers/sdd/issue-26-billing-profiles-edit-delete/progress.md（worktree-local）。

## 范围（Scope）

- `web/src/api/hooks.ts`（Billing profiles 段内追加）：
  - `useUpdateBillingProfile()`（`api.billing.updateProfile({id, body})`，
    onSuccess 失效 `nsPrefix('billing-profiles')`）；
  - `useDeleteBillingProfile()`（`api.billing.deleteProfile({id})`，onSuccess 同上）。
- `web/src/features/config/billing-profiles/billing-profile-form-dialog.tsx`：
  增加编辑模式（新可选 prop `profile?: Profile | null`）：
  - 打开时回填 name/description/supplier（法定名称/税号/key/地址六项/电话）/
    default；
  - apps 三个选择器（appTax/appInvoicing/appPayment）**禁用**并显示
    「创建后不可变」提示（Issue 验收项），值回填自 profile.apps 仅供展示；
  - 提交走 `useUpdateBillingProfile`：body `{name, description?, supplier,
    workflow: profile.workflow, default}`——**不含 apps**（update 契约无此字段）
    且 **workflow 必须回填 profile.workflow**（PUT 全量替换语义；禁止
    workflow:{} 把服务端 workflow 设置重置为默认——create 的 `workflow: {}`
    注释只对创建成立）。标题/按钮文案用 edit 键。
- `web/src/features/config/billing-profiles/index.tsx`：列表新增「操作」列
  （编辑 → 打开 dialog(profile=row)；删除 → ConfirmDialog destructive，确认后
  deleteProfile，受限错误经 handleServerError **原文 toast 透出**——Issue 验收项）。
- i18n：`config.billingProfiles.edit`/`appsImmutable`（禁用提示）/
  `delete`/`deleteConfirm.{title,description}`/`toast.{updated,deleted}`/
  `list.actions`，zh-CN 与 en 同构。

## 非目标（Non-goals）

- 创建流程（#25 已交付，零行为改动；同 dialog 复用，无 profile prop 时为创建）。
- apps 关联变更（API 不可变；UI 只读展示）。
- workflow 设置编辑（UI 一贯只读；编辑回填原值）。
- 默认档案切换专用端点（default 开关随 PUT 提交，唯一默认由后端裁决、错误透出）。
- 后端/TypeSpec 改动。

## 已核实的契约事实（SDK dist models/operations/billing.d.ts + types.d.ts 6981，2026-08-29）

- `updateProfile` body 类型 `UpsertBillingProfileRequestInput = { name(1-256),
  description?(≤1024), labels?, supplier: Party, workflow: WorkflowInput,
  default: boolean }`——**无 apps 字段**（apps 仅在 CreateBillingProfileRequest
  中必填；Profile 响应携带 apps 供展示）。
- `Workflow` 与 `WorkflowInput` 为同构全可选结构（collection?/invoicing?/
  payment?/tax? 各 Settings(.Input)）；`Workflow = profile.workflow` 直接作为
  input 传给 update 在结构上成立（SDK AcceptDateStrings 处理 Date 序列化）。
- `deleteProfile({id})` → void；受限删除（非默认、无 customer override 引用）
  由后端裁决返回错误问题详情——handleServerError 现有行为为原文透出。
- 既有 UI 事实：form dialog 的 zod schema/EMPTY_VALUES/提交映射在
  billing-profile-form-dialog.tsx（apps 三选必填 min(1) 仅创建校验；编辑态
  三选禁用但值非空即可过校验）；index.tsx renderProfile 现为 5 列无操作列，
  表头需同步加第 6 列；`formatDateTime` 已存在。
- `Profile.apps = { tax: {id}, invoicing: {id}, payment: {id} }`；编辑回填
  用 `profile.apps.tax.id` 等赋给表单值即可（appNameMap 反查展示名仅列表用）。

## 任务拆分（每任务独立 commit + `cd web && pnpm build && pnpm lint`）

- T1 API 层：useUpdateBillingProfile + useDeleteBillingProfile。
- T2 编辑模式 + 操作列 + 删除确认 + i18n（zh/en 同构）。

## 测试与验收命令

- 每 commit：`cd web && pnpm build && pnpm lint`（routeTree 零 diff）。
- 终态追加：`pnpm test:e2e`（与并行轨串行）；locale parity 真实求值；
  prettier 新文件 clean、修改文件差异恰为插入块。
- 浏览器走查（全端点 mock）：编辑打开即回填全部字段；apps 三选禁用且有
  「创建后不可变」提示（Issue 验收项 1）；提交 wire 体 deep-equal
  `{id, body:{name, description, supplier(全地址映射), workflow: 原值,
  default}}` 且**不含 apps 键**；删除确认后 wire 体 `{id}`；受限删除
  （mock 409/422 problem+json）时错误原文出现在 toast（Issue 验收项 2）；
  成功删除后列表刷新；创建模式行为与 #25 完全一致（回归）。

## 全局约束

- 只动 web/ 下上述文件；三 Go 模块不可误改。
- 与并行轨 #22/#24/#28 共享面仅 hooks.ts 追加段与 locale 文件互不相交子树。
- 文案全部 i18n（zh-CN + en 同步），术语沿 CONTEXT.md 词表（账单档案）。
- 删除为破坏性操作，必须 ConfirmDialog；错误原文透出，不得吞错改写。
- 未经用户批准不得 push/merge/close/改 GitHub 状态。
- implementer 不得自行派生 subagent；不得跳过任务审查/修复复审/终审/台账。
