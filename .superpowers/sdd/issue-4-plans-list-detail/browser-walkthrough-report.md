# Browser Walkthrough Report — Issue #4 Task 1（计划列表 + 详情）

- 审查对象：worktree `/Users/wuyongjun/trea/openmeter-issue-4`，分支 `codex/admin-config-04` @ `45f89e50da349027daf280c8549763466163ea73`
- 走查时间：2026-08-27 13:26–13:45（本机真实 Chromium，Playwright 1.62.1）
- 服务拓扑：mock OIDC IdP `http://127.0.0.1:9999`（`web/e2e/mock-idp.mjs`）+ vite preview `http://127.0.0.1:4173`（serve `web/dist`），视口 1440×900
- 临时 spec/脚本/trace 全部位于 `/tmp`，worktree 源文件零改动
- 原始断言记录：`/tmp/issue4-walkthrough-results.json`；像素校验：`/tmp/issue4/pixel-check.json`；Playwright trace：`/tmp/issue4/test-results/*/trace.zip`（7 份）

## 构建产物说明（重要）

worktree 内预存在的 `web/dist` 无法完成 OIDC 登录：`index.html` 实际加载的 chunk 未烘焙任何 `VITE_CASDOOR_*` 配置，控制台报 `[auth] VITE_CASDOOR_ISSUER or VITE_CASDOOR_CLIENT_ID is not set` / `No authority or metadataUrl configured`（dist 目录中另遗留一个含 `127.0.0.1:9999` 的旧 chunk，hash 与 index.html 引用不一致，属过期残留）。

处置：按仓库自有 e2e 流程（`web/playwright.config.ts` webServer 的原样 env）重建 dist ——
`VITE_CASDOOR_ISSUER=http://127.0.0.1:9999 VITE_CASDOOR_CLIENT_ID=openmeter-admin-e2e VITE_CASDOOR_REDIRECT_URI=http://127.0.0.1:4173/auth/callback VITE_CASDOOR_LOGOUT_REDIRECT_URI=http://127.0.0.1:4173/sign-in pnpm build`。
`web/dist` 为 gitignored 产物，源码树不受影响；被测代码仍为 45f89e50d 的源。重建后用独立调试脚本验证 OIDC 往返（discovery → authorize 302 → /auth/callback → POST /token → dashboard）完整通过。

## 场景裁定总览

| 场景 | 内容 | 裁定 |
|---|---|---|
| a | 列表渲染（结构/7 列/4 状态徽章/链接） | **PASS**（15 项断言） |
| b | 状态筛选（wire + 行过滤 + 页码重置） | **PASS**（8 项断言） |
| c | 分页（25 条 → 3 页，page 2 请求） | **PASS**（7 项断言） |
| d | 详情数据路径（2 阶段 × 5 价格类型） | **PASS**（28 项断言） |
| e | loading skeleton / 404 兜底 | **PASS**（5 项断言） |
| f | en locale 切换 | **PASS**（17 项断言） |
| g | 健壮性（未知 price.type/空 phases/超长名称） | **PASS**（6 项断言） |
| h | 截图 + 像素校验 | **PASS**（9 张，全部通过） |

**合计 86 项断言，0 失败；全程 pageerror = 0；捕获 plans 相关 GET 26 次。**

## 场景断言明细

### a. 登录后访问 /config/plans — PASS

OIDC 登录参照 `web/e2e/smoke.spec.ts`（mock IdP round-trip 落到 dashboard）后访问 `/config/plans`。mock 注入 4 条计划（draft/active/archived/scheduled；EUR/USD/JPY/GBP；monthly/yearly/quarterly；v3/v5/v1/v2）。

- PageHeader 标题「计划」+ 描述「管理产品计划：阶段、价目卡与版本状态。」可见
- 侧边栏「配置」分组与「计划」链接（href=`/config/plans`）可见
- `<html lang>` 含 `zh`（默认 locale 生效）
- 表头恰好 7 列，逐一等于 `['名称','Key','版本','状态','币种','计费周期','创建时间']`
- 4 条数据行渲染；名称链接 href=`/config/plans/plan-active-2`（按 id 拼接）
- 每行断言：草稿/已发布/已归档/待生效 4 种 StatusBadge + 币种 + 周期 + `v{version}` 全部正确
- 初始 wire：`GET /api/v3/openmeter/plans?page[number]=1&page[size]=10&sort=created_at desc`，无 filter

### b. 状态筛选 — PASS

12 条 mock（3 draft + 9 混合状态）。先翻到第 2 页，再切筛选：

- 翻页 wire：`page[number]=2`；第 2 页呈现「混合条目 08/09」
- 切到「草稿」后新请求 wire：**同一次请求**携带 `filter[status]=draft` + `page[number]=1`（页码重置的直接证据）
- 行过滤：仅剩 3 行草稿计划，非草稿行（如「混合条目 01」）计数归零；分页显示「Page 1 of 1」
- 切回「全部状态」：行数恢复 10、触发器回到「全部状态」；10s 内不发新请求（app 全局 `staleTime: 10 * 1000`，缓存命中，属产品合法行为，已如实记录）

### c. 分页 — PASS

25 条 mock、pageSize 10：

- 「Page 1 of 3」；第 1 页呈现 01–10
- 点击「Go to page 2」→ wire `page[number]=2`（同请求 `page[size]=10`、`sort=created_at desc`）；行变为 11–20，01 消失
- 「Page 2 of 3」可见；再翻第 3 页 → 剩余 5 行（21–25）

### d. 详情数据路径 — PASS

列表点击「综合演示计划」进入 `/config/plans/plan-demo-1`（客户端路由）。mock 返回 2 阶段：阶段 1（`duration: P30D`，free + flat 卡）、阶段 2（无 duration，unit + graduated(2 档) + volume(3 档) 卡）。

- 返回链接「返回计划列表」href=`/config/plans`
- 标题区：h1 名称 + StatusBadge「已发布」+ `v2`
- InfoRow 全字段：key=`demo-plan`、币种=USD、计费周期=monthly、创建/更新时间（2024 格式化日期）、描述
- 阶段标题：「阶段 1 · 试用阶段」+「期限 P30D」；「阶段 2 · 正式阶段」+「无限期（最后阶段）」
- 每阶段价目卡表 6 列头 = `['价目卡','价格类型','功能','价格','计费周期','Key']`；共 2 张阶段表
- EnumBadge 5 类：免费 / 固定费 / 单价 / 阶梯（累计）/ 阶梯（整单）
- 价格摘要：免费；`USD 29.00`（flat）；`EUR 0.10 / 单位`（unit，卡级 EUR 覆盖）；「2 档阶梯价」「3 档阶梯价」
- cadence 回退：无卡级 cadence 的「平台费」行回退 plan 的 monthly；「API 调用」行用卡级 yearly
- feature.id 渲染（`01FEATFREE…`）与未关联卡的 `—` 占位

### e. 详情 loading / 404 兜底 — PASS

- slow mock（2.5s 延迟）直访 `/config/plans/plan-slow`：`[data-slot="skeleton"]` ≥2 块可见（skeleton 截图取证）
- 404 mock 直访 `/config/plans/plan-missing` 与不存在 id `/config/plans/01UNKNOWNID`：均渲染「计划不存在或已被删除。」（react-query 生产环境按 `retry>3` 重试退避后落定，30s 断言窗内通过；该场景共捕获 11 次 GET，即重试序列本身）

### f. en locale — PASS

`page.addInitScript` 预置 `localStorage['openmeter-admin.language']='en'`（#3 先例键名）：

- `<html lang>` 切为 `en`
- 列表：标题「Plans」、英文描述、7 列英文表头、徽章「Published」
- 详情：「Back to plans」「Phases & rate cards」「Phase 1 · 试用阶段」「Duration P30D」「Indefinite (last phase)」、徽章 Free/Flat fee/Per unit/Graduated/Volume、「2-tier pricing」「3-tier pricing」「USD 29.00」「EUR 0.10 / unit」

### g. 健壮性 — PASS

- 未知 `price.type: 'mystery'`：EnumBadge defaultValue 渲染出「mystery」文本，价格摘要单元格为空（switch 无命中返回 undefined，React 渲染空），同表正常 flat 卡仍显示 `USD 5.00`，页面不崩溃
- 空 `phases: []`：「阶段与价目卡」标题在、阶段表计数 0、无「阶段 N」文本
- 300+ 字符超长名称：列表行与详情 h1 均渲染
- 以上全部场景 `pageerror` 计数为 0（每个用例收尾显式断言）

## 请求 wire 记录（全部 GET，无 POST）

按场景去重后的查询串（URL 解码后）：

| 场景 | 请求 |
|---|---|
| a | `/api/v3/openmeter/plans?page[number]=1&page[size]=10&sort=created_at desc` |
| b | 同上初始；`?page[number]=2&page[size]=10&sort=created_at desc`；`?filter[status]=draft&page[number]=1&page[size]=10&sort=created_at desc` |
| c | `?page[number]=1/2/3&page[size]=10&sort=created_at desc` |
| d | 列表初始 + `GET /api/v3/openmeter/plans/plan-demo-1`（无查询串） |
| e | `GET …/plans/plan-slow`、`…/plans/plan-missing`、`…/plans/01UNKNOWNID`（后两者含 react-query 重试序列，共 11 次） |
| f | 列表初始 + `…/plans/plan-demo-1` |
| g | 列表初始 + `…/plans/plan-garbage`、`…/plans/plan-empty-phases`、`…/plans/plan-long-name` |

实测编码形态与控制器核实一致：`page%5Bnumber%5D=2`、`sort=created_at+desc`、`filter%5Bstatus%5D=draft`。mock 响应体全程 snake_case wire 字段（`billing_cadence/created_at/rate_cards/…`），UI 无 undefined 渲染。

## pageerror 计数

**0**（7 个用例 × 每 400ms 收尾窗口监听 `pageerror`，逐用例显式断言为零）。

## 截图清单与像素指标（Pillow 11.3.0）

9 张 1440×900 PNG 存于 `/tmp/issue4-shots/`：

| 文件 | 场景 | std | 唯一色 | 背景占比 | 判定 |
|---|---|---|---|---|---|
| 01-list-zh.png | 列表 zh（4 状态） | 22.27 | 1216 | 0.944 | PASS |
| 02-filter-draft.png | 筛选 draft 后 | 21.60 | 1177 | 0.917 | PASS |
| 03-page2.png | 分页第 2 页 | 25.72 | 1091 | 0.920 | PASS |
| 04-detail-zh.png | 详情 zh（2 阶段全价格类型） | 26.43 | 1064 | 0.918 | PASS |
| 05-detail-en.png | 详情 en | 23.45 | 1146 | 0.922 | PASS |
| 06-404.png | 404 兜底 | 13.77 | 620 | 0.980 | PASS |
| 07-skeleton.png | loading skeleton | 14.32 | 612 | 0.748 | PASS |
| 08-robust-unknown-type.png | 未知 price.type | 22.14 | 909 | 0.945 | PASS |
| 09-list-en.png | 列表 en | 18.65 | 1161 | 0.956 | PASS |

- 尺寸校验：9/9 = 1440×900
- 非空白校验：9/9（std>2 且唯一色>500）
- 成对差异：36 对全部通过（meanAbsDiff 最小 2.039 / changed 最小 3.1%，无任何两张实质相同）
- 校验脚本：`/tmp/issue4/verify_shots.py`，结果 JSON：`/tmp/issue4/pixel-check.json`

## 视觉通道限制（如实标注）

本部署 read_image/describe_image 通道确认损坏（`model "glm-5.3" does not declare image input`，与 #1–#3 同况）。本次视觉证据为程序化：DOM 断言（86 项）+ Playwright trace（7 份 zip，含每步快照与网络）+ Pillow 像素指标（上表）。未做人工目视复核。

## 缺陷清单

- **Critical：无**
- **Important：无**
- **Minor（观察项，均为共享组件既有行为，非本任务回归，不阻塞）：**
  1. 分页控件文案 `Page x of y` / `Rows per page` 硬编码英文，zh 界面中英混排（`components/data-table/pagination.tsx`）。
  2. zh 界面日期按 date-fns 默认 enUS locale 格式化（如 `May 8, 2024, 6:00:00 PM`，`lib/format.ts` 未传 zh locale）。
  3. 环境备注：worktree 预存在 dist 未烘焙 `VITE_CASDOOR_*`（且目录内混有旧 env chunk 残留），任何直接 `pnpm preview` 的走查都会卡登录；建议后续交付前按 e2e env 重建或在交付说明中标注。

## 总裁定

**PASS** —— 场景 a–h 全部通过；86/86 断言通过；pageerror=0；wire 事实（snake_case 响应、`filter[status]`/`page[number]`/`sort=created_at desc` 请求串、`**/api/v3/openmeter/plans*` 拦截面）与控制器核实完全一致；9 张截图通过全部像素校验。
