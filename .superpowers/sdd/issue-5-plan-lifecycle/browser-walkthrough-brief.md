# Browser Walkthrough Brief — Issue #5 Task 1

You are the browser walkthrough verifier for OpenMeter issue #5 (计划：发布/归档/克隆新版本). Verify the REAL built artifact in a headless browser with in-DOM assertions plus screenshots.

Worktree: /Users/wuyongjun/trea/openmeter-issue-5 (branch codex/admin-config-05). The web app: web/ (React 19 + TanStack Router + TanStack Query).

## 端口隔离（重要）

Two independent code reviewers may run `pnpm test:e2e` concurrently (it hard-occupies 127.0.0.1:9999 and :4173 with strictPort). You MUST use different ports:

```bash
cd /Users/wuyongjun/trea/openmeter-issue-5/web
# 1. build with mock IdP baked in on :9989, redirect to :4183
VITE_CASDOOR_ISSUER=http://127.0.0.1:9989 \
VITE_CASDOOR_CLIENT_ID=openmeter-admin-e2e \
VITE_CASDOOR_REDIRECT_URI=http://127.0.0.1:4183/auth/callback \
VITE_CASDOOR_LOGOUT_REDIRECT_URI=http://127.0.0.1:4183/sign-in \
  pnpm build
# 2. mock IdP on :9989
MOCK_IDP_PORT=9989 node e2e/mock-idp.mjs &
# 3. preview on :4183
pnpm exec vite preview --host 127.0.0.1 --port 4183 --strictPort &
```

Kill stale listeners on 9989/4183 first (`lsof -ti :9989,:4183 | xargs kill` style). Kill what you started when done. Login flow: mirror web/e2e/smoke.spec.ts (Casdoor button → mock IdP 302 round-trip) against http://127.0.0.1:4183.

## Wire 事实（控制器已核实，直接采用）

- v3 SDK client baseUrl `/api/v3`；拦截 `**/api/v3/openmeter/plans*`（列表，query `filter[status]=`、`page[number]/page[size]`、`sort=created_at+desc`）、`**/api/v3/openmeter/plans/<id>`（详情）、`**/api/v3/openmeter/plans/<id>/publish`（POST）、`**/api/v3/openmeter/plans/<id>/archive`（POST）。列表响应 `{data:[planWire],meta:{page:{number,size,total}}}`；详情/publish/archive 响应为单个 planWire。
- planWire 必须 snake_case：`id/name/key/version/currency/billing_cadence/pro_rating_enabled/status/created_at/updated_at/phases[]`，phase=`{name,key,duration?,rate_cards[]}`，card=`{name,key,feature?:{id},price,payment_term}`，price oneOf `{type:'free'}|{type:'flat',amount:'10'}|{type:'unit',amount:'2'}|…`。camelCase 字段会渲染 undefined。
- v1 克隆端点：`**/api/v1/plans/<idOrKey>/next`（POST，无请求体），201 响应为 **v1 camelCase**（`{id,name,key,version,currency,billingCadence,status,createdAt,updatedAt}`）。二次克隆错误路径：响应 409（或 400）`{"message":"..."}`——错误原文经 handleServerError toast 透出。
- web client 未开 SDK zod 校验，mock 字段无需 100% 齐全，但务必 snake_case（v1 除外）。

## 状态化 mock 设计（spec 内维护计划存储）

初始两条计划：
- planA `{id:'plan-draft-1', key:'starter', version:1, status:'draft', name:'起步版'}`
- planB `{id:'plan-active-1', key:'pro', version:2, status:'active', name:'专业版'}`

路由处理器按 method+path 变更存储：publish→status:'active'（effective_from 设 now）；archive→status:'archived'（effective_to 设 now）；v1 next→若该 key 已有 draft 则 409 `{"message":"there is already a plan in draft status"}`，否则新建 `{id:'plan-draft-2', key:<同源>, version:<源+1>, status:'draft'}` 并返回 v1 camelCase。列表 GET 尊重 `filter[status]`。

## 场景（逐项 DOM 断言证据 + 截图，1440x900）

a. **列表基线**：/config/plans 渲染两行；planA 行状态徽章「草稿/Draft」、planB「生效/Active」；名称链接 href=/config/plans/<id>。
b. **draft 详情发布**：点 planA 名称 → 详情 h1=起步版 + 草稿徽章 + v1；断言「发布」按钮可见、「克隆新版本」与「归档」不可见。点「发布」→ ConfirmDialog（标题「发布计划」、描述含「起步版」）；先点「取消」断言弹窗关闭且未发请求；重新点「发布」确认 → 断言发出 `POST /api/v3/openmeter/plans/plan-draft-1/publish`；toast「计划已发布」出现；徽章变为「生效」；此时「归档」+「克隆新版本」可见、「发布」消失。回列表 → draft 筛选下无 planA（filter[status]=draft 请求返回空），all 视图下 planA 为生效。
c. **active 克隆（成功路径）**：进 planB（专业版 v2 生效）→「克隆新版本」+「归档」可见、「发布」不可见。点克隆 → 弹窗描述含 v2；确认 → 断言 `POST /api/v1/plans/plan-active-1/next`；toast「已创建新草稿 v3」；URL 变为 /config/plans/plan-draft-2；详情为「专业版」草稿 v3 + 「发布」按钮可见；回列表 all 视图出现新草稿行。
d. **二次克隆 4xx 透出**：再次进 planB → 点「克隆新版本」确认 → mock 409 → 断言 toast 显示「there is already a plan in draft status」原文；弹窗保持打开（confirming 未重置）；页面不崩溃。
e. **归档**：planB（或 planA）详情 →「归档」→ 确认弹窗（destructive 样式）→ 确认 → POST archive → toast「计划已归档」→ 徽章「已归档」；列表同步。
f. **i18n en**：localStorage `openmeter-admin.language='en'`（#3 先例）后重载：按钮 Publish / Clone next version / Archive；弹窗标题 Publish plan / Clone next version；toast Plan published / New draft v3 created。
g. **全程 pageerror 零错误**（page.on('pageerror') 计数）。
h. **截图**（≥7 张，/tmp/issue5-shots/）：list-baseline-zh、draft-detail-zh、publish-confirm-zh、published-active-zh、clone-confirm-zh、clone-navigated-draft-zh、clone-4xx-toast-zh、archived-zh、detail-en。像素校验（Pillow 可用则做，参照 /tmp/verify_shots.py 模式）：尺寸、非空白（unique colors>500）、成对差异。

## 已知部署限制

read_image/describe_image 视觉通道此前会话曾损坏（#1–#4 同况，由控制器自行复核截图）；你以 DOM 断言 + trace + 像素指标作为程序化证据，如实记录。

## 输出与清理

写入 `.superpowers/sdd/issue-5-plan-lifecycle/browser-walkthrough-report.md`：场景 a–h 逐项 PASS/FAIL + 断言明细（引用实际 DOM 文案）、请求 wire 记录（POST 端点/body）、pageerror 计数、截图清单 + 像素指标表、总裁定行 `Ruling: PASS|FAIL`。原始 JSON 断言记录存 /tmp/issue5-walkthrough-results.json。结束清理：kill mock-idp/preview、删临时 spec、确认 `git status` 树净（仅 gitignored .superpowers/sdd/ 与 dist/）。不改 progress.md，不改任何 tracked 文件，无 GitHub 写操作，不得派生 subagent。
