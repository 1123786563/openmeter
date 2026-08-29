# Task 2 report — issue #22 T2 目录区 + 安装 dialog + 挂载 + i18n

- Status: DONE
- Commit: `a99dd981b` feat(admin): 应用目录浏览与安装表单 (issue #22)
- Branch: codex/admin-config-22（worktree /Users/wuyongjun/trea/openmeter-issue-22）

## 文件清单（2 新建 2 修改，约束内）

| 文件 | 变更 |
| --- | --- |
| `web/src/features/config/apps/install-app-dialog.tsx` | 新建。安装 dialog |
| `web/src/features/config/apps/app-catalog-section.tsx` | 新建。目录区 |
| `web/src/features/config/apps/index.tsx` | 修改。+import、页头注释补一句目录区语义、已装列表 `</div>` 后挂载 `<AppCatalogSection />`（+6/-1，已装列表/卸载/换 Key 逻辑零触碰） |
| `web/src/i18n/locales/zh-CN.ts` | 修改。`config.apps` 子树末尾只追加 `catalog` + `install`（+28/-0，纯插入块） |
| `web/src/i18n/locales/en.ts` | 修改。同构追加（+29/-0，纯插入块） |

## 关键实现决策

1. **zod 条件必填（Issue 验收项）**：`createInstallSchema(isStripe)` 用
   `.superRefine` 在 `type==='stripe'` 且 apiKey 为空（trim 后）时向
   `path:['apiKey']` 加 issue；错误文案统一在 `FormMessage` 手动渲染
   `config.apps.install.validation.apiKeyRequired`（照抄 stripe-key-dialog
   手动消息模式）。因 RHF 的 resolver 在首挂载固化、而 `item` 首次为 null，
   父组件用 `key={installTarget?.type ?? 'none'}` 按类型重挂 dialog，
   保证 resolver 与所选目录项类型一致（组件内有注释说明）。
2. **提交体三分支**：stripe → `{type:'stripe',name,createBillingProfile,apiKey}`；
   sandbox / external_invoicing → 表单不渲染开关、固定
   `createBillingProfile:false`（SDK 契约三分支均必填 boolean，Ruling 笔误
   已按 SDK 事实执行）。ternary 直接内联进 `mutate(...)` 首参，借上下文
   类型保住 `type` 字面量判别。
3. **打开即重置**：`useEffect(open)` 内 `form.reset({name:item?.name,apiKey:'',
   createBillingProfile:true})`——name 预填可改、apiKey 清空、开关默认开。
4. **目录区**：`useAppCatalog()`，目录项取 `data.data`；Skeleton×3 loading、
   空态行 colSpan=5。每项一行 Table（风格对齐已装列表：`bg-hover/50` 表头、
   pl-6/pr-6、ghost sm 按钮）：type 徽章（复用 `config.apps.type.*`）、
   name+description（description 作 muted 小字）、capabilities 徽章（复用
   `config.apps.capability.*`，key=capability.key）、installMethods 徽章（新键
   `config.apps.catalog.installMethod.*`）。该列表头无对应 i18n 键且键集封闭，
   故此列表头为空（徽章自说明）。
5. **安装按钮可用性**：`installMethods.includes('with_api_key') ||
   includes('no_credentials_required')` 才可用；否则禁用并在按钮旁显示
   `config.apps.catalog.oauthUnsupported`（不实现 OAuth，非目标）。
6. **i18n**：仅追加 brief 列举的键，zh/en 完全同构；description 讲清后果
   （安装会在当前命名空间创建应用关联；Stripe 可同时创建计费档案）。

## 验证命令与输出摘要（均真实运行）

| 命令（cwd=worktree/web） | 退出码 |
| --- | --- |
| `pnpm build` | **0**（vite ✓ built in 383ms，tsc -b + tsr generate 通过） |
| `pnpm lint` | **0**（eslint 无输出） |
| brief 的 locale 粗查 node -e | 0（两文件 `catalog:{` 计数均 1） |
| 递归键比对（加强）：eval `config.apps` 子树比较叶键集合 | zh 52 = en 52，only-in 差集为空；`catalog`/`install` 新子树键序完全一致（title/empty/installMethod.{with_oauth2,with_api_key,no_credentials_required}/installAction/oauthUnsupported；title/description/name/apiKey/apiKeyHint/createBillingProfile/createBillingProfileHint/validation.{nameRequired,apiKeyRequired}/submit/toast.installed） |
| `git status` | 仅 4+1 文件；**无 routeTree.gen.ts 改动**（无需还原） |
| `pnpm exec prettier --check`（两个新文件） | 0（clean；已 --write 归一 import 排序） |
| prettier（修改文件） | 未 --write：两 locale 与 index.tsx 在 HEAD 本就非 prettier-clean（历史状态，非本次引入）；本次 diff 保持纯插入块 |

## 疑虑

- `AppCatalogSection` 目录表的 installMethods 列表头为空：键集按 brief 封闭
  （无该列表头键），徽章文案自说明。若审查希望有表头，需增键（超出 brief）。
- 修改文件未做 prettier 全文重排（会污染插入块 diff）；仓库该三文件历史上
  即非 prettier-clean，终态 prettier 检查若按整仓口径会仍报（与 #21 交付时
  状态一致）。
- 浏览器 mock 走查属计划终态任务（全端点 mock），不在 T2 验证范围。

## 修复轮 1

- Commit: `86b831bc8` fix(admin): 安装表单错误文案本地化与目录表头补键 (issue #22)
- 修复文件（5，均在授权范围）：install-app-dialog.tsx / app-catalog-section.tsx /
  zh-CN.ts / en.ts / docs/superpowers/plans/issue-22-apps-catalog-install.md。

### Q1（Important）schema message 本地化

- `createInstallSchema(isStripe, t)` 注入 `TFunction`（i18next 直接依赖，已在
  package.json）；`name` 的 `min(1)` 与 apiKey superRefine message 均改为
  `t('config.apps.install.validation.nameRequired' | 'apiKeyRequired')`（复用
  既有键，零新增校验键）。
- 组件内 `const schema = useMemo(() => createInstallSchema(isStripe, t),
  [isStripe, t])`，resolver 用该 schema；per-type 重挂键逻辑
  （app-catalog-section.tsx:141-148）未动，语言切换时 t 引用变化亦会重建。
- 两处 FormMessage 死翻译 children 删除，改 `<FormMessage />`（form.tsx:137
  error 优先渲染 error.message，children 永不生效——对齐
  external-invoice-dialog 模式）；ui/form.tsx 未动。
- 工厂 docstring 补一句：message 必须在 schema 构造期经 t 解析，否则裸键
  泄漏到 UI。

### Q2（计划修订补键）installMethods 列表头

- zh-CN/en 在 `config.apps.catalog.installMethod.*` 块后同构插入
  `installMethodsLabel`（zh '安装方式' / en 'Install methods'），各 +1 行
  纯插入。
- app-catalog-section.tsx 空 TableHead（原注释「badges self-describe」）改填
  `t('config.apps.catalog.installMethodsLabel')`。
- 计划文档 i18n 键清单（Scope 第 26-28 行）catalog 括号列表追加
  「installMethodsLabel 安装方式列表头」一项。

### 验证（均真实运行，cwd=worktree/web）

| 命令 | 退出码 |
| --- | --- |
| `pnpm build` | **0**（vite ✓ built in 397ms，tsc -b + tsr generate 通过） |
| `pnpm lint` | **0**（eslint 无输出） |
| `node /tmp/issue-round/locale-parity.mjs <web>` | **0**（zh=843 en=843，parity OK） |
| `pnpm exec prettier --check` 两组件文件 | 0（--write 归一后 clean；locale/计划文档保持纯插入不重排） |
| `git status` | 提交后 clean；routeTree.gen.ts 本次零 churn（无需还原） |

- 提交体：5 files changed, +21/-18。
- 修复后疑虑清单第 1 条（installMethods 空 表头）与审查 Q1 均已闭环。
