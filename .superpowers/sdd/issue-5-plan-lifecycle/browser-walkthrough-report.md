# Browser Walkthrough Report — Issue #5 Task 1（计划：发布/归档/克隆新版本）

- 日期：2025-08-27（本机时区）
- 验证对象：REAL built artifact（`pnpm build` 产物，由 `vite preview` 服务）
- Worktree：`/Users/wuyongjun/trea/openmeter-issue-5`（branch `codex/admin-config-05`）
- 端口隔离：mock IdP `http://127.0.0.1:9989`（`MOCK_IDP_PORT=9989`），preview `http://127.0.0.1:4183`（build env `VITE_CASDOOR_ISSUER=http://127.0.0.1:9989`、`VITE_CASDOOR_CLIENT_ID=openmeter-admin-e2e`、`VITE_CASDOOR_REDIRECT_URI=http://127.0.0.1:4183/auth/callback`、`VITE_CASDOOR_LOGOUT_REDIRECT_URI=http://127.0.0.1:4183/sign-in`）。未占用 9999/4173。
- 驱动方式：plain Node 脚本（Playwright library API，`web/node_modules` 内的 Chromium 1234，headless，viewport 1440x900），登录流程镜像 `web/e2e/smoke.spec.ts`（goto `/` → Casdoor 按钮 → mock IdP 302 → `/auth/callback` → dashboard）。
- 断言记录：`/tmp/issue5-walkthrough-results.json`（43 项 check 全 PASS + 20 条 wire 记录 + pageerror 计数 + 像素指标）。截图：`/tmp/issue5-shots/`（9 张）。
- 状态化 mock：spec 内维护 planWire 存储，初始 planA `plan-draft-1/starter/v1/draft/起步版`、planB `plan-active-1/pro/v2/active/专业版`；publish→`status:'active'`+`effective_from`；archive→`status:'archived'`+`effective_to`；`POST /api/v1/plans/<id>/next` 同 key 已有 draft 则 409 `{"message":"there is already a plan in draft status"}`，否则新建 `plan-draft-2`（v3 draft）并返回 v1 camelCase（201）；列表 GET 尊重 `filter[status]`。v3 响应 snake_case（`billing_cadence/pro_rating_enabled/rate_cards/payment_term`…），v1 克隆响应 camelCase。

## 场景结果

### a. 列表基线 — PASS
- `/config/plans` 渲染 2 行（`tbody tr` count = 2）。
- planA 行状态徽章实际 DOM 文案为「草稿」（Draft），planB 为「已发布」（Active）。注：brief 预期写的是「生效」，实际 locale `plan.status.active` 的 zh-CN 值是「已发布」（`web/src/i18n/locales/zh-CN.ts`）；en 为「Published」。按真实产物记录，判定 PASS。
- 名称链接 href：`/config/plans/plan-draft-1`、`/config/plans/plan-active-1`。
- 列表 wire：`GET /api/v3/openmeter/plans?page%5Bnumber%5D=1&page%5Bsize%5D=10&sort=created_at+desc`（即 `page[number]=1&page[size]=10&sort=created_at desc` 的 URL 编码形式）。
- 截图：`list-baseline-zh.png`。

### b. draft 详情发布 — PASS
- 点击「起步版」进入详情：`h1` = 起步版；徽章「草稿」；版本 `v1`；「发布」按钮可见，「克隆新版本」「归档」不存在（count 0）。截图 `draft-detail-zh.png`。
- 点「发布」→ ConfirmDialog：标题「发布计划」、描述含「起步版」与「确定要发布」。截图 `publish-confirm-zh.png`。
- 先点「取消」：弹窗关闭，wire 记录确认未发出任何 publish POST。
- 重新确认发布：wire `POST /api/v3/openmeter/plans/plan-draft-1/publish`（无请求体）；toast「计划已发布」出现；徽章变为「已发布」；此时「归档」+「克隆新版本」可见、「发布」消失。截图 `published-active-zh.png`。
- 回列表 all 视图 planA 为「已发布」；切 draft 筛选（Select 选项「草稿」）→ wire 出现 `GET /api/v3/openmeter/plans?...&filter%5Bstatus%5D=draft` 且响应为空（空态单行 colSpan 渲染「暂无计划」，无 planA 行）；切回「全部状态」恢复 2 行。

### c. active 克隆（成功路径） — PASS
- planB 详情（专业版 v2 已发布）：「克隆新版本」+「归档」可见、「发布」不存在。
- 点克隆 → 弹窗标题「克隆新版本」、描述含 `v2`（「将以『专业版』v2（最新发布版）为底本…」）。截图 `clone-confirm-zh.png`。
- 确认 → wire `POST /api/v1/plans/plan-active-1/next`（无请求体，201）；toast「已创建新草稿 v3」；URL 变为 `/config/plans/plan-draft-2`；详情「专业版」+「草稿」徽章 + `v3` +「发布」按钮可见。截图 `clone-navigated-draft-zh.png`。
- 回列表 all 视图出现新草稿行（3 行，含 专业版 v3 草稿）。

### d. 二次克隆 4xx 透出 — PASS
- 再次进 planB → 点「克隆新版本」确认 → mock 409 → toast 原文「there is already a plan in draft status」（`handleServerError` 透出 `ApiError.message` = body `message`）。
- wire 第二次 `POST /api/v1/plans/plan-active-1/next` 记录在案；弹窗保持打开（`alertdialog` 仍可见，confirming 未重置）；页面不崩溃（后续交互正常）。截图 `clone-4xx-toast-zh.png`。
- 浏览器 console 仅 1 条预期内的 fetch 日志（`Failed to load resource: 409 (Conflict)`），无 JS 异常。

### e. 归档 — PASS
- planB 详情 →「归档」→ 弹窗标题「归档计划」，确认按钮为 destructive 样式（class 含 `destructive`）。
- 确认 → wire `POST /api/v3/openmeter/plans/plan-active-1/archive`（无请求体）→ toast「计划已归档」→ 徽章「已归档」。截图 `archived-zh.png`。
- 回列表：planB 行「已归档」同步。

### f. i18n en — PASS
- `localStorage['openmeter-admin.language']='en'` 后 reload（本场景开始时将 mock 存储重置为 planA active / planB active v2、无草稿，以复现与 zh 相同的生命周期文案路径）。
- planB（active）：按钮 `Archive`、`Clone next version` 可见，`Publish` 不存在。
- 克隆弹窗标题 `Clone next version`、描述含 `v2`；确认 → toast `New draft v3 created` → 跳转 `/config/plans/plan-draft-2`，详情 `Draft` 徽章 + `v3` + `Publish` 按钮可见。截图 `detail-en.png`。
- 点 `Publish` → 弹窗标题 `Publish plan` → 确认 → toast `Plan published`。

### g. pageerror 零错误 — PASS
- 全程 `page.on('pageerror')` 计数 = **0**（覆盖登录 + a–f 全部页面与交互）。

### h. 截图与像素校验 — PASS
- 9 张截图（≥7 要求），全部 1440x900，全部唯一颜色数 1004–1422（>500 非空白），成对差异均 > 0（无重复图）。含弹幕遮罩的两张（4xx toast / archived）均值明显变暗（alert dialog overlay 变暗背景所致），进一步证明截图捕获了弹窗态。

## Wire 请求记录（POST / 关键 GET）

| 场景 | Method & Path | Body | 响应 |
|---|---|---|---|
| a/b/c/e | `GET /api/v3/openmeter/plans?page[number]=1&page[size]=10&sort=created_at desc` | — | `{data:[…], meta:{page:{number,size,total}}}` |
| b | `GET /api/v3/openmeter/plans?…&filter[status]=draft` | — | 空 data |
| b | `POST /api/v3/openmeter/plans/plan-draft-1/publish` | 无 | planWire(active) |
| c | `POST /api/v1/plans/plan-active-1/next` | 无 | 201 v1 camelCase（plan-draft-2, v3, draft） |
| d | `POST /api/v1/plans/plan-active-1/next` | 无 | 409 `{"message":"there is already a plan in draft status"}` |
| e | `POST /api/v3/openmeter/plans/plan-active-1/archive` | 无 | planWire(archived) |
| f | `POST /api/v1/plans/plan-active-1/next` → `POST /api/v3/openmeter/plans/plan-draft-2/publish` | 无 | 201 / planWire(active) |

（完整 20 条 wire 记录见 `/tmp/issue5-walkthrough-results.json` 的 `wireLog`。）

## 截图清单与像素指标

| 截图 | 尺寸 | 唯一颜色 | 平均 RGB |
|---|---|---|---|
| /tmp/issue5-shots/list-baseline-zh.png | 1440x900 | 1085 | 252.2, 252.4, 252.6 |
| /tmp/issue5-shots/draft-detail-zh.png | 1440x900 | 1004 | 251.1, 251.3, 251.7 |
| /tmp/issue5-shots/publish-confirm-zh.png | 1440x900 | 1018 | 251.0, 251.2, 251.6 |
| /tmp/issue5-shots/published-active-zh.png | 1440x900 | 1156 | 251.2, 251.4, 251.7 |
| /tmp/issue5-shots/clone-confirm-zh.png | 1440x900 | 1238 | 237.5, 237.7, 238.0 |
| /tmp/issue5-shots/clone-navigated-draft-zh.png | 1440x900 | 1013 | 250.9, 251.1, 251.6 |
| /tmp/issue5-shots/clone-4xx-toast-zh.png | 1440x900 | 1401 | 133.0, 133.2, 133.6 |
| /tmp/issue5-shots/archived-zh.png | 1440x900 | 1422 | 135.8, 135.7, 136.0 |
| /tmp/issue5-shots/detail-en.png | 1440x900 | 1077 | 251.3, 251.5, 251.9 |

成对平均像素差（mean |Δ|, 0-255）：draft-detail vs published-active 0.675；publish-confirm vs clone-confirm 15.086；clone-navigated-draft vs detail-en 3.077；clone-confirm vs clone-4xx-toast 106.313；draft-detail vs clone-navigated-draft 1.355。

## 已知限制与备注

- 视觉通道（read_image/describe_image）此前会话损坏（brief 已知部署限制）：本报告以 DOM 断言 + wire 记录 + pageerror 计数 + Pillow 像素指标作为程序化证据；截图保留于 /tmp 供控制器自行复核。
- 术语偏差记录：brief 预期 active 徽章「生效」、toast 后英文案等，实际产物 zh-CN 为「已发布」「已归档」（en：Published / Archived）；所有断言按实际 DOM 文案执行并如实记录。
- 清理已完成：mock-idp/preview 已 kill（9989/4183 无监听），临时 spec（/tmp/issue5-walkthrough/）已删除；`git status --porcelain` 仅余 `.superpowers/sdd/issue-5-plan-lifecycle/`（gitignored/untracked），无任何 tracked 文件改动、无 commit、无 GitHub 操作。

Ruling: PASS
