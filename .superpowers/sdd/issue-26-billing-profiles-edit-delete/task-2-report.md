# Task 2 report — issue #26 T2 编辑模式 + 操作列 + 删除确认 + i18n

- 状态：DONE
- 提交：`871606d13` `feat(admin): 账单档案编辑（apps 不可变）与删除 (issue #26)`（分支 codex/admin-config-26，基于 T1 `433993e4a`）
- 改动文件（仅 brief 允许的 4 个）：
  - `web/src/features/config/billing-profiles/billing-profile-form-dialog.tsx`
  - `web/src/features/config/billing-profiles/index.tsx`
  - `web/src/i18n/locales/zh-CN.ts`（billingProfiles 子树，纯插入）
  - `web/src/i18n/locales/en.ts`（同构，纯插入）

## 实现要点（对照 brief 交付物）

1. **form dialog 编辑模式**
   - Props 新增 `profile?: Profile | null`（`@openmeter/client` 导入）；`editing = Boolean(profile)`。
   - 打开时 reset：editing 回填 name/description/supplierName/supplierKey/supplierTaxId
     （`taxId.code`）/地址六项+电话（按 `Party.addresses.billingAddress` 字段名
     country/line1/line2/city/state/postalCode/phoneNumber，缺省 ''）/default；
     `appTax=profile.apps.tax.id`、`appInvoicing=…invoicing.id`、`appPayment=…payment.id`
     （非空保证通过 zod min(1)）；非 editing 走 `form.reset(EMPTY_VALUES)`，#25 行为不变。
   - apps 三选：editing 时 `disabled`（Select disabled 透传）+ 每个 FormItem 条件渲染
     FormDescription `t('config.billingProfiles.appsImmutable')`（Issue 验收项 1）。
   - 标题：editing 用新键 `form.editTitle`，否则 `form.createTitle`；提交按钮沿用
     `common.confirm/submitting`（pending = create.isPending || update.isPending）。
   - 提交分叉：先构造共享 `body = {name, description?, supplier{…既有映射…}, default}`，
     再分叉——
     - editing：`updateMutation.mutate({ id: profile.id, body: { ...body, workflow: profile.workflow } })`，
       **workflow 回填 profile.workflow（非空对象）**，body **不含 apps 键**；成功
       toast `toast.updated`。
     - 创建：`createMutation.mutate({ ...body, workflow: {}, apps: {tax/invoicing/payment}, })`
       ——`workflow:{}` 注释与原 create 语义保留，原 toast/关闭行为不变。
2. **index.tsx 列表**
   - 表头第 6 列 `list.actions`（`pr-6 text-right`；createdAt 列头让出 pr-6）；
     空态 colSpan 5→6。
   - 行尾操作 cell（`pr-6`，右对齐）：编辑（Pencil ghost sm）→ `setEditTarget(profile)`；
     删除（Trash2 destructive ghost sm）→ `setDeleteTarget(profile)`（样式沿
     apps/index.tsx 既有行操作模式）。
   - 状态：`editTarget`/`deleteTarget` 均 `useState<Profile | null>(null)`；创建按钮
     `setFormOpen(true)` 不变。
   - 同一 dialog 实例复用：`open={formOpen || Boolean(editTarget)}`，
     `onOpenChange` 里 `setFormOpen(open)` 且关闭时 `setEditTarget(null)`；
     `profile={editTarget}`（创建/编辑互斥）。
   - 删除 ConfirmDialog（destructive）：desc 含 `{{name}}` 插值；确认 →
     `deleteMutation.mutate({id}, { onSuccess: toast(toast.deleted)+关闭, onError: handleServerError })`
     ——错误原文透出（handleServerError 现有行为，Issue 验收项 2），未吞错改写。
3. **i18n（两 locale 同构，只追加）**：`edit`、`delete`、`appsImmutable`、
   `form.editTitle`、`list.actions`、`deleteConfirm.{title,description}`、
   `toast.{updated,deleted}`；既有键零改动。

## 验证（真实运行，worktree web/ 下）

| 命令 | 退出码 |
|---|---|
| `pnpm build` | 0（✓ built in 353ms） |
| `pnpm lint` | 0 |
| routeTree.gen.ts | 无改动（无需还原） |
| zh/en billingProfiles 键 parity（tsx 实际求值） | 39 = 39，无单侧键 |
| locale diff 纯插入检查（grep 删除行） | 0 删除行 |

## 备注 / 已知事项

- 环境事实：worktree 内安装的 prettier 3.8.3 对 CJK 字符串宽度的度量与历史版本不同，
  **4 个目标文件在 HEAD 上本就不过 `prettier --check`**（与本任务无关的既有状态）。
  处理：两个 tsx（本轨道独占文件）按 3.8.3 规范化（顺带 3 处既有行 reflow，使文件
  通过 format:check）；两个 locale 文件（与并行轨道共享）**严格保持纯插入 diff**，
  未引入任何既有行重排，避免合并冲突。

## 修复轮 1

- 提交：`0ddf6a15893ad30c476f6d6b12d306dd4e4baec5`
  `fix(admin): 账单档案更新回显 labels 防清空 (issue #26)`
  （1 file changed, 8 insertions(+), 1 deletion(-)）
- 改动（仅 `web/src/features/config/billing-profiles/billing-profile-form-dialog.tsx`）：
  update 提交分叉的 body 在 `workflow: profile.workflow` 回显旁增加 labels 回显
  `...(profile.labels && Object.keys(profile.labels).length > 0 ? { labels: profile.labels } : {})`
  （无标签时不发该键，SDK `UpsertBillingProfileRequestInput.labels?` 可选），并加一行
  注释「服务端 update 为全量替换且无 labels 回填保护，须回显避免清空」。创建分支与
  其他逻辑零改动。
- 依据（控制器裁定，服务端实证）：api/v3/handlers/billingprofiles/convert.gen.go 将
  缺省 Labels 转 nil Metadata，openmeter/billing/adapter/profile.go 无条件
  SetMetadata；服务端对 workflow 有回填保护、对 labels 没有，客户端必须回显。
  已核 SDK dist types.d.ts：`Profile.labels?: Labels`（5277）、
  `UpsertBillingProfileRequestInput.labels?: Labels`（6981）、
  `Labels = Record<string, string>`。
- 验证（真实运行，worktree web/ 下）：`pnpm build` exit 0（✓ built in 732ms）、
  `pnpm lint` exit 0；`git status` 仅目标文件改动，routeTree.gen.ts 无 churn
  （无需还原）。
