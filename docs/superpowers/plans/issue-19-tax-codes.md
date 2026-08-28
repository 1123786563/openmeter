# Issue #19 税码：CRUD 与 app 映射 — 实施计划

- **Issue**: https://github.com/1123786563/openmeter/issues/19（[admin-config 19/29] 税码：CRUD 与 app 映射）
- **Branch / base**: `codex/admin-config-19`，base `f6e767dc3`（= main = origin/main）
- **Worktree**: `/Users/wuyongjun/trea/openmeter-issue-19`
- **上游计划**: `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 10 的 #19 子集（不含 #20 的组织默认卡片）；处方式细节以 Issue 首条评论为准（逐文件完整代码）。

## 范围（Scope）

1. `web/src/api/query-keys.ts`：追加 `taxCodes: (params) => ns('tax-codes', params)`。
2. `web/src/api/hooks.ts`：新增 Tax codes 段（`useTaxCodes(includeDeleted=false)` 走 v3 `api.tax.listCodes`（page size 100）/ `useCreateTaxCode` / `useUpsertTaxCode` / `useDeleteTaxCode`，mutation 成功后 `invalidateQueries(nsPrefix('tax-codes'))`）。注：Issue 评论的「在 Currencies 段之后」是假定 #17 先落地的排序锚点；本轨道与 #17 并行，实际插入在当前文件 Features 段之后、Helpers 段之前，最终排序由合并产生（无代码依赖）。
3. 新建 `web/src/features/config/tax-codes/tax-code-form-dialog.tsx`：创建/编辑复用（编辑时 key 只读禁用 + 提示；upsert body 无 key 字段）；appMappings 动态行（useFieldArray：app_type 三枚举 Select + 映射码 Input + 删行）；zod 校验 name(1-256)/key(ResourceKey `^[a-z0-9]+(?:_[a-z0-9]+)*$`)/description(≤1024)/映射行必填 + appType 去重。
4. 新建 `web/src/features/config/tax-codes/index.tsx`：列表（key/name/appMappings 徽章/updatedAt/操作）；`include_deleted` 开关（Switch，重新查询）；编辑/删除按钮对已删除行禁用；删除走 ConfirmDialog；已删除行显示 destructive 徽章。
5. `web/src/routes/_authenticated/config/tax-codes/index.tsx`：替换 #1 占位为 `TaxCodesPage` 挂载。
6. `web/src/i18n/locales/{zh-CN,en}.ts`：新增 `config.taxCodes.*` 全量子树；复用 `common.optional`（#17 同轮补入；若本轨道先合并则由本轨道补——见并行说明）。

## 非目标（Non-goals）

- 不做组织默认税码卡片（`GET|PUT /openmeter/defaults/tax-codes` 归 #20；#20 复用本任务 `useTaxCodes` 与 `nsPrefix('tax-codes')` 约定）。
- 不改 sidebar（#1 已含「税码」条目）、不新增 e2e（收尾 issue 统一）。
- 不动 v3 SDK 与后端。

## 任务拆分

- **Task 1（唯一实施任务）**：按 Issue 评论的逐文件处方案落地上述 6 项；单 feat 提交 `feat(admin): 税码管理（CRUD 与 app 映射）`。
- **Task 2**：规格符合性审查（对照 Issue 验收标准与处方案逐条）。
- **Task 3**：代码质量审查（约定、回归、locale 完整性程序化比对）。
- **Task 4**：浏览器走查（wire 级证据：listCodes 查询参数、创建 POST body、key 只读、重复 appType 拦截、删除确认）。
- **Task 5**：全分支最终审查。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e   # 三连，既有 2 条冒烟不回归
```

浏览器手测（Issue 验收）：创建税码（name=数字服务、key=`digital_services`、sandbox=`txcd_00000000`、stripe=`txcd_10000000`）出现在列表；编辑时 key 只读、映射行可增删且重复 appType 被拦截；删除走确认弹窗；「显示已删除」可见已删除行且操作禁用。

## 全局约束

- 文案全走 i18n，zh-CN 与 en 同步；术语按 CONTEXT.md 词表（税码）。
- 所有请求经 `web/src/api/client.ts`（自动注入 Authorization + X-Namespace）。
- 端点能力以 v3 spec 为准（已核实：`UpsertTaxCodeRequest` 无 key → key 创建后不可改；已删除税码 PUT 410 由 `handleServerError` toast 透出；delete 返回 204）。
- 删除一律 ConfirmDialog；删除失败/受限错误 toast 原文透出。
- ResourceKey 正则与 #2 features 的 FEATURE_KEY 逐字一致（`^[a-z0-9]+(?:_[a-z0-9]+)*$`）。
