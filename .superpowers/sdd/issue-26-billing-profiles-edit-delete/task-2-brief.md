# Task 2 brief — issue #26 T2 编辑模式 + 操作列 + 删除确认 + i18n

Worktree: /Users/wuyongjun/trea/openmeter-issue-26（分支 codex/admin-config-26）
T1 已提供（勿改）：`useUpdateBillingProfile()`（input `{id, body}`，body 类型
SDK `UpsertBillingProfileRequestInput` = `{name, description?, labels?, supplier,
workflow, default}`——**无 apps**）、`useDeleteBillingProfile()`（input `{id}`）。

## 必读参考（worktree 内先读再写）

- `docs/superpowers/plans/issue-26-billing-profiles-edit-delete.md`（契约事实节必读：
  workflow 回填、apps 不可变）
- `web/src/features/config/billing-profiles/billing-profile-form-dialog.tsx`（本任务主改文件：
  zod schema、EMPTY_VALUES、提交映射、apps 三选、打开时 reset）
- `web/src/features/config/billing-profiles/index.tsx`（列表+挂载点）
- `web/src/features/config/apps/index.tsx`（ConfirmDialog + 行操作模式）
- SDK 类型（worktree 内 `web/node_modules/@openmeter/client/dist/models/types.d.ts`）：
  `Profile`（含 supplier.addresses.billingAddress 结构）、`UpsertBillingProfileRequestInput`

## 交付物

1. 修改 `web/src/features/config/billing-profiles/billing-profile-form-dialog.tsx`：
   - Props 增加可选 `profile?: Profile | null`（'@openmeter/client' 导入）。
   - `const editing = Boolean(profile)`；打开时 reset：editing 用 profile 回填
     name/description/供应商字段（supplierName/key/taxId 与地址六项+电话，按 SDK
     `Party.addresses.billingAddress` 字段名逐一对映，缺省 ''）/default；
     appTax=profile.apps.tax.id、appInvoicing=profile.apps.invoicing.id、
     appPayment=profile.apps.payment.id（禁用态仍需通过 zod min(1)）；
     非 editing 维持 #25 原有 EMPTY_VALUES 行为**逐字节不变**。
   - apps 三个选择器：editing 时 `disabled` + 每个 FormItem 下加
     FormDescription `t('config.billingProfiles.appsImmutable')`（**Issue 验收项 1**）。
   - dialog 标题：editing 用新键 `form.editTitle`，否则沿用既有 `form.createTitle`；
     提交按钮文案沿用既有键。
   - 提交分叉：构造 `const body = { name, description?, supplier: {…既有映射…},
     default }`（supplier 映射复用现有 createMutation 调用处的构造，最小重构：
     先构造再分叉，创建分支行为不变）；editing 时走
     `updateMutation.mutate({ id: profile.id, body: { ...body,
     workflow: profile.workflow } }, …)`——**workflow 必须回填 profile.workflow，
     禁止 workflow: {}（会把服务端 workflow 设置重置为缺省；create 的 workflow:{}
     注释仅对创建成立）**；body **不得含 apps 键**（update 契约无此字段）。
     成功 toast 用新键 `toast.updated`；创建分支保持原 toast/行为。
2. 修改 `web/src/features/config/billing-profiles/index.tsx`：
   - 表头加第 6 列 `config.billingProfiles.list.actions`（pr-6 text-right）；
     renderProfile 行尾加操作 cell：编辑（Pencil ghost sm）→
     `setEditTarget(profile)`；删除（Trash2 destructive ghost sm）→
     `setDeleteTarget(profile)`。
   - 状态：`const [editTarget, setEditTarget] = useState<Profile | null>(null)`、
     deleteTarget 同构；创建按钮改为 `setCreateOpen(true)` 等既有行为不变。
   - `<BillingProfileFormDialog open={formOpen || Boolean(editTarget)} …>`——或按你
     读到的现状用最小改法保证：创建与编辑互斥复用同一 dialog 实例，editing 传
     `profile={editTarget}`；关闭时同时清 editTarget。
   - 删除 ConfirmDialog（destructive）：desc 含 {{name}}；确认 →
     `deleteMutation.mutate({ id }, { onSuccess: toast(config.billingProfiles.toast.deleted)
     + 关闭, onError: handleServerError })`——**错误原文透出是 Issue 验收项 2**，
     不得吞错改写。
3. i18n：`config.billingProfiles` 子树内**只追加**：`edit`、`appsImmutable`、
   `delete`、`deleteConfirm: { title, description }`、`toast: { updated, deleted }`
   （并入既有 toast 键旁）、`list.actions`、`form.editTitle`。zh/en 同构；不改既有键。

## 约束

- 只动上述 3 个文件；不碰 hooks（T1 已就绪）/query-keys。
- 创建模式回归零变化（#25 行为逐项保持）。

## 验证（必须真实运行并记录退出码）

```
cd /Users/wuyongjun/trea/openmeter-issue-26/web && pnpm build && pnpm lint
```

双 exit 0；无 routeTree 改动；locale 新键 zh/en 一致。

## 提交

```
git add src/features/config/billing-profiles/ src/i18n/locales/zh-CN.ts src/i18n/locales/en.ts
git commit -m "feat(admin): 账单档案编辑（apps 不可变）与删除 (issue #26)"
```

## 报告

完整报告写入 `.superpowers/sdd/issue-26-billing-profiles-edit-delete/task-2-report.md`，
回复只给四行：状态/提交哈希/一行测试摘要/疑虑。
