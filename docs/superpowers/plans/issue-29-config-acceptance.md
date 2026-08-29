# Issue #29 — 配置域全链路验收（实施计划）

- **Issue**: https://github.com/1123786563/openmeter/issues/29 `[admin-config 29/29] 配置域全链路验收`
- **处方来源**: issue 评论（2026-08-26，处方级 7 步骤）+ 总计划 `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 15
- **分支**: `codex/admin-config-29`（base = main @ 601fe0b6e），worktree `/Users/wuyongjun/trea/openmeter-issue-29`

## 范围

1. `web/e2e/smoke.spec.ts` 新增第 3 条用例 `config plans smoke: /config/plans reachable and renders list data`（用例总数 2→3）：`MOCK_PLAN_NAME` 常量 + 完整用例（拦截 `**/api/v3/openmeter/plans*` 返回 wire 格式 `BillingPlan` 固定计划；登录后断言侧边栏「计划」链接可见；`goto /config/plans` 断言 URL；计划单元格或「建设中/Under Construction」占位兜底可见）。
2. 浏览器全链路走查四大块，各完成一次写操作并整窗截图×4：
   - ① `/config/features` 新建功能（name `验收-功能`、key `acceptance_feature`）→ 列表出现该行 → `01-config-features-created.png`
   - ② `/config/notification/channels` 新建渠道（name `验收-Webhook`、url `https://example.com/webhook`）→ 列表出现且未禁用 → `02-config-channel-created.png`
   - ③ `/config/tax-codes` 新建税码（name `验收税码`、key `acceptance_tax_code`）→ 列表出现 → `03-config-tax-code-created.png`
   - ④ `/config/apps` 目录区安装 sandbox 应用（name `验收-Sandbox`）→ 已装列表出现 sandbox 条目 → `04-config-app-installed.png`
   - 截图需同时可见侧边栏「配置」分组与操作结果；产物不入仓库，附到 issue 完成评论（外发需用户批准）。
3. 全量回归：`web/` 下 `pnpm build && pnpm lint && pnpm test:e2e`（3 条全绿）；仓库根 `go build ./...`（无输出）；`git status` 核验 Go 文件零 diff（本系列无后端改动）。
4. `web/README.md` 两处补充：(a)「E2E 冒烟测试」用例清单追加第 3 条；(b) 其后新增「配置域（/config）」小节（四大块路由清单 + i18n/SDK 说明），逐字采用处方文案。

## 非目标

- 不改任何后端/Go 代码、openapi spec、SDK 生成物。
- 不改 `playwright.config.ts` 与 `e2e/mock-idp.mjs`（处方明令）。
- 不为列表首屏写操作端点加 mock（除非首屏确有第二个读端点导致失败，届时按 customers 双路径模式补并在台账注明）。
- 不在本轮执行外发（push/merge/issue 评论+关闭）——等待用户批准。
- 走查后清理（税码删除等可选项）不做。

## 全局约束

- 本系列全部工单无后端改动；验收须同时证明前端全绿且 Go 后端零误改。
- 命令在 `web/` 下执行（Go 回归在仓库根）；工具缺失时 `nix develop --impure .#ci -c <cmd>`（禁止并发 nix develop）。
- Consumes（不得改名）：路由 `/config/plans`、i18n key `config.plans.title`（zh「计划」）、侧边栏「配置」分组（计划/功能/附加组件/通知/货币/税码/应用/门户令牌/账单档案）、v3 SDK wire 格式 `{data, meta.page}`（snake_case，`BillingPlan` schema，必填 `id/name/created_at/updated_at/key/version/currency/billing_cadence/status/phases`，`status` 枚举 `draft|active|archived|scheduled`）。
- 复用既有测试设施：`login(page)` helper、`page.route(...).fulfill(...)` mock 模式、playwright 双 webServer（mock IdP + preview）。
- Produces：smoke.spec.ts 第 3 用例（名称 `config plans smoke: ...`）；README 用例清单第 3 条 + 「配置域（/config）」小节。
- 仓库约定：e2e 冒烟不依赖真实 OpenMeter/Casdoor；root `go build ./...` 不含 e2e 模块。
- **环境裁定（本计划前置 Ruling，详见领取台账）**：真实后端走查在本环境不可达（:8888 为无状态 shim、web/.env 缺失、compose 栈未起）→ 走查以有状态持久化 mock（POST 真实写入内存存储、GET 回读）驱动真实 UI 全链路完成四写操作并截图；尽力对 :5173+shim 做真实探查；shortfall 记为剩余风险，随外化批准一并提请用户裁决。

## 任务拆分

- **T1 e2e 冒烟第 3 用例**：smoke.spec.ts 按处方逐字实现 + `pnpm test:e2e` 3 条全绿。
- **T2 README 补充**：两处逐字落档；`pnpm build`+`pnpm lint` 复核（README 改动不碰代码路径）。
- **T3 浏览器全链路走查**：四块各一写操作+截图+DOM 断言（stateful mock 模式；登录走 mock IdP）；产出走查报告（含每步断言与截图清单）入 SDD 工作区；尽力真实环境探查（:5173）并如实记录结果。
- **T4 全量回归**：web 三连 + `go build ./...` + Go 零 diff 核验 + `git status` 干净度（除计划文档与 smoke/README 改动外无受控文件变化）。
- **最终全分支审查**：最强可用模型，全分支 diff 五角度审查（规格符合/质量/回归面/一致性/发布就绪）。

## 测试与验收命令

```bash
cd web && pnpm test:e2e          # 预期 3 passed（sign-in/customers/config plans）
cd web && pnpm build             # 预期 exit 0
cd web && pnpm lint              # 预期 0 errors
go build ./...                   # 预期无输出
git status --porcelain           # 预期仅 web/e2e/smoke.spec.ts、web/README.md、docs 计划文档
git diff --stat main -- '*.go'   # 预期空（后端零误改）
```

验收（issue Acceptance criteria 映射）：

| 标准 | 证据 |
| --- | --- |
| e2e 冒烟覆盖至少一个新配置页 | smoke.spec.ts 第 3 用例 + 3 passed 输出 |
| 四大块各完成一次写操作并留证 | 4 张整窗截图 + 走查报告（DOM 断言 + wire 记录） |
| 全量回归绿 | build/lint/test:e2e exit 0 + go build 无输出 + Go 零 diff |
