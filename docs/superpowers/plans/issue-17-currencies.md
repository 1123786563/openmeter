# Issue #17 货币：法币与自定义列表/创建 — 实施计划

- **Issue**: https://github.com/1123786563/openmeter/issues/17（[admin-config 17/29] 货币：法币与自定义列表/创建）
- **Branch / base**: `codex/admin-config-17`，base `f6e767dc3`（= main = origin/main）
- **Worktree**: `/Users/wuyongjun/trea/openmeter-issue-17`
- **上游计划**: `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 9 的 #17 子集（不含 #18 的 cost-bases 面板）；处方式细节以 Issue 首条评论为准（逐文件完整代码）。

## 范围（Scope）

1. `web/src/api/query-keys.ts`：追加 `fiatCurrencies` / `currencies` 两个 key。
2. `web/src/api/legacy.ts`：追加 `FiatCurrency` 接口与 `listFiatCurrencies()`（v1 `GET /api/v1/info/currencies`，camelCase）。
3. `web/src/api/hooks.ts`：新增 Currencies 段（`useFiatCurrencies` 24h staleTime / `useCurrencies(params)` 走 v3 `api.internal.currencies.list` / `useCreateCustomCurrency()`，mutation 成功后 `invalidateQueries(nsPrefix('currencies'))`）。
4. 新建 `web/src/features/config/currencies/index.tsx`：双 tab（法币只读表 / 自定义货币表 + 不可变警示 Alert + 新建按钮）；自定义 tab 请求 `filter[type]=custom` + `expand=cost_basis`，行内展示成本基准条数（为 #18 预置数据，不加操作列）。
5. 新建 `web/src/features/config/currencies/custom-currency-dialog.tsx`：仅创建（API 无更新/删除端点）；动态 schema 对活数据校验 code 冲突（法币 + 既有自定义，大小写不敏感）；字段 name(1-256)/code(4-24)/precision(0-12 整数)/decimalMark(1 字符)/thousandSeparator(1 字符)/symbol?(≤16)。
6. `web/src/routes/_authenticated/config/currencies/index.tsx`：替换 #1 占位为 `CurrenciesPage` 挂载。
7. `web/src/i18n/locales/{zh-CN,en}.ts`：新增 `config.currencies.*` 全量子树；补 `common.optional`（两份 locale 的 common 段当前无此 key，已核实）。

## 非目标（Non-goals）

- 不做 cost-bases 面板/追加表单（#18 范围；本任务仅请求 expand 并显示条数）。
- 不做自定义货币更新/删除（API 无端点，页面明示不可变）。
- 不改 sidebar（#1 已含「货币」条目）、不新增 e2e（配置域冒烟由收尾 issue #29 统一补）。
- 不动 v3 SDK 与后端。

## 任务拆分

- **Task 1（唯一实施任务）**：按 Issue 评论的逐文件处方案落地上述 7 项；单 feat 提交 `feat(admin): 货币管理——法币列表与自定义货币创建`。
- **Task 2**：规格符合性审查（对照 Issue 验收标准与处方案逐条）。
- **Task 3**：代码质量审查（约定、回归、locale 完整性程序化比对）。
- **Task 4**：浏览器走查（wire 级证据：v1 法币列表渲染、创建 POST body、冲突校验拦截）。
- **Task 5**：全分支最终审查。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e   # 三连，既有 2 条冒烟不回归
```

浏览器手测（Issue 验收）：`/config/currencies` 法币 tab 出 v1 列表；自定义 tab 创建 `CREDIT_POINTS` 成功入列；与法币/既有自定义冲突的 code 被前端拦截；后端错误 toast 透出原文。

## 全局约束

- 文案全走 i18n，zh-CN 与 en 同步；术语按 CONTEXT.md 词表（自定义货币）。
- 所有请求经 `web/src/api/client.ts` / `legacy.ts`（自动注入 Authorization + X-Namespace）。
- v3 响应 snake_case（SDK 已转 camel）；v1 手写层 camelCase。
- 端点能力以 `api/v3/openapi.yaml` / `api/openapi.yaml` 为准，禁止臆造字段（已核实：自定义货币仅 create + get，code 4-24，`costBasis` 仅在 expand 时返回）。
- 自定义货币端点带 `x-internal` → SDK 归入 `api.internal.currencies`（已核实存在于 dist/sdk/internal.d.ts）。
- 写操作确认后果：创建前 dialog 明示「创建后不可编辑/删除」。
