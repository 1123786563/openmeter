# Browser Walkthrough Brief — Issue #4 Task 1

你是浏览器走查员。对 worktree `/Users/wuyongjun/trea/openmeter-issue-4`（分支 `codex/admin-config-04` @ 45f89e50d）构建产物做真实浏览器验证。**禁止改源文件、禁止派生子代理、禁止 GitHub 写操作。**

## 准备

```bash
cd /Users/wuyongjun/trea/openmeter-issue-4/web
# 端口卫生
lsof -ti :4173 :9999 | xargs kill -9 2>/dev/null
# 构建产物已存在（implementer pnpm build）；如需重建：pnpm build
# 起 mock IdP + preview（参照既有 e2e 配置 web/playwright.config.ts 的 webServer 做法，后台运行）
node e2e/mock-idp.mjs &  # :9999
pnpm exec vite preview --port 4173 &  # :4173
```

用 Playwright（web/node_modules 已有）写临时 spec（/tmp/issue4-walkthrough.spec.ts，参照 web/e2e/smoke.spec.ts 的登录流程拿到会话）验证以下场景。e2e 目录的既有 spec 不要改。

## 控制器已核实的 wire 事实（直接采用，勿再推导）

- 拦截端点：`**/api/v3/openmeter/plans*`（列表）与 `**/api/v3/openmeter/plans/<id>`（详情）——client baseUrl=`/api/v3` + SDK 路径 `openmeter/plans`。
- 请求查询串（SDK encodings.js serializeDeepObject + encodeSort 实测逻辑）：筛选 → `filter[status]=draft`；分页 → `page[number]=2&page[size]=10`；排序 → `sort=created_at+desc`（空格分隔，URL 编码后 `+` 或 `%20`）。
- **mock 响应体必须用 snake_case wire 字段**（fromWire 无条件转换）：`id/name/description?/created_at/updated_at/key/version/currency/billing_cadence/status/phases[]`，phase=`{name,description?,key,duration?,rate_cards[]}`，card=`{name,key,feature?:{id},currency?,billing_cadence?,price,payment_term}`，price oneOf `{type:'free'}|{type:'flat',amount}|{type:'unit',amount}|{type:'graduated',tiers[]}|{type:'volume',tiers[]}`，tier=`{up_to_amount?,flat_price?:{type:'flat',amount},unit_price?:{type:'unit',amount}}`。
- web client 未开启 SDK zod wire 校验（client.ts 无 validate 选项），mock 不必 100% 字段齐全，但 camelCase 字段会导致 UI 显示 undefined——务必 snake_case。
- 认证：登录流程参照 web/e2e/smoke.spec.ts（mock IdP :9999 OIDC round-trip）。

## 场景（逐项记录 DOM 断言证据 + 截图）

a. **登录后访问 /config/plans**：列表渲染（Header/PageHeader/表格/侧边栏配置分组）；空数据或 mock 数据两种形态按实际后端而定——preview 连的是无后端环境，用 page.route() mock `GET /openmeter/plans*`（v3 wire 格式：data: Plan[] + meta.page.total）注入 ≥3 条计划（不同 status：draft/active/archived/scheduled、不同 currency/cadence/version/name/key）。断言：7 列表头、名称链接 href=/config/plans/<id>、StatusBadge 渲染 4 种状态、v+版本、币种、周期。
b. **状态筛选**：切到 draft → 断言发起新请求且 filter[status]=draft（或 wire 等价形式）、列表只剩 draft 行、页码重置；切回 all 恢复。
c. **分页**：mock 25 条 pageSize 10 → 断言 3 页、翻页请求 page[number]=2、行数变化。
d. **详情-数据路径**：mock `GET /openmeter/plans/<planId>` 返回 ≥2 阶段（阶段 1 duration ISO8601、末阶段无 duration）×每阶段 ≥2 价目卡（覆盖 free/flat/unit/graduated(2 档)/volume(3 档) 至少各一张，多阶段可拆开覆盖）。点击列表名称进入详情：断言返回链接、名称+状态徽章+版本、InfoRow 基本信息全部字段、阶段标题「阶段 N · name」+期限/无限期、价目卡表 6 列、EnumBadge 价格类型 5 种渲染、价格摘要（免费/固定费金额+币种/单价 per unit/「N 档阶梯价」）、cadence 回退（card 无 cadence 时用 plan 的）。
e. **详情-loading/兜底**：slow mock 断言 skeleton；404 mock 断言 notFound 文案；planId 不存在同上。
f. **i18n 切换**：en locale 下列表+详情关键文案为英文（localStorage 或 UI 切换，参照 #3 先例 openmeter-admin.language）。
g. **健壮性**：garbage 注入（price.type 未知值/极长名称/空 phases 的 plan）不崩溃——未知 price.type 走 EnumBadge defaultValue、空 phases 渲染无阶段卡片且无 JS 错误；pageerror 监听全程零错误。
h. **截图**：每主要场景 1440x900 截图存 /tmp/issue4-shots/（≥6 张：列表 zh、筛选后、分页第 2 页、详情 zh、详情 en、404 兜底）。像素校验（Pillow，参照 #1–#3 的 /tmp/verify_shots.py 模式）：尺寸、非空白、成对差异。

## 已知部署限制

read_image/describe_image 视觉通道可能损坏（#1–#3 同况）：以 DOM 断言 + Playwright trace + 像素指标作为程序化视觉证据，如实标注。

## 输出

写入 `.superpowers/sdd/issue-4-plans-list-detail/browser-walkthrough-report.md`：场景 a–h 逐项 PASS/FAIL + 断言明细、请求 wire 记录（POST/GET body 关键字段）、pageerror 计数、截图清单 + 像素指标表、总裁定。原始 JSON 断言记录存 /tmp/issue4-walkthrough-results.json。结束后清理：kill mock-idp/vite preview 进程、删除临时 spec、确认 worktree 树干净（git status）。只写报告文件，不改 progress.md。
