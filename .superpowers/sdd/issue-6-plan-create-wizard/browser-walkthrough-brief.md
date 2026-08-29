# Browser Walkthrough Brief — Issue #6 计划创建向导（free/flat 价卡）

你是浏览器走查员。用真实浏览器（Playwright chromium）对**构建产物**走通 Issue #6 的手测脚本，产证截图与 wire 记录。

## 环境搭建（先看 web/tests/e2e 既有基建怎么跑——mock-idp :9999 + vite preview :4173 是既有模式）

- Worktree：`/Users/wuyongjun/trea/openmeter-issue-6`；先 `cd web && pnpm build`（若产物已在则跳过）。
- 参考既有 e2e 的启动方式（web/tests/e2e/ 下 spec 与 package.json 的 test:e2e script），用同样的 mock IdP 登录往返进入 /config/plans。
- 你的 spec 放 /tmp/issue6-walkthrough/（不进仓库树），服务器手动管理（或复用脚本），结束不留进程、不改仓库任何文件。

## 走查用例（Issue 规定的手测脚本，全部必须覆盖）

**正向 a**：登录 → /config/plans → 点「新建计划」→ 步骤 1 填 name=测试专业版/key=test_pro_wiz/currency=CNY/cadence=P1M/description → 下一步 → 步骤 2 默认阶段改名「标准期」key=standard duration=P1M + 添加阶段 2「无限期」key=forever duration 留空 → 下一步 → 步骤 3：阶段 1 价目卡 name=平台费 key=platform_fee cadence=P1M price=free；阶段 2 价目卡 name=年费 key=yearly_fee cadence=one_time price=flat amount=99.00 → 创建计划 → toast「计划已创建（草稿）」出现 → dialog 关闭 → 列表刷新出现 test_pro_wiz（draft 徽章）→ 点进详情，阶段结构与输入一致（两阶段、各自价目卡、free/flat、期限/无限期、一次性 cadence）。
**反向 b**：新建 → 步骤 2 两个阶段、阶段 1 duration 清空 → 点下一步 → 「仅最后一个阶段可以无期限」出现在**阶段 1 的 duration 输入下方**（行级定位，截图证明）→ 补回 P1M 可继续。
**反向 c**：flat 金额输入 `abc` → 「请输入非负数字金额」行内错误（amount 输入下方）→ 改 99.00 错误消失。
**反向 d**（若 UI 可达）：尝试让某阶段价目卡数 <1（删除按钮在仅 1 张时禁用——记录禁用态截图与提示语义；若不可达则记录「删除按钮 disabled 防御」为覆盖方式）。
**回归 e**：列表页既有功能（状态筛选、分页、既有计划行链接）不回归；zh/en 切换后向导文案正确（步骤 1 各标签 + 错误文案各截一张）。

## 产证要求

- 每用例 ≥1 张 1440x900 截图（存 /tmp/issue6-shots/，文件名含用例号）；toast/dialog/行级错误必须可见。
- DOM 断言（expect ... toBeVisible/toHaveText）逐用例写在 spec 里，全绿退出码 0。
- wire 证据：拦截 POST /openmeter/plans 请求体（page.on('request')），记录一次成功提交的完整 JSON body 到 /tmp/issue6-walkthrough/create-body.json（含两阶段、free/flat price、billingCadence null→缺省字段）。
- zero pageerror：记录 page 'pageerror' 事件计数为 0。

## 输出

报告写到 `.superpowers/sdd/issue-6-plan-create-wizard/browser-walkthrough-report.md`：用例 a–e 各 PASS/FAIL + 截图路径 + DOM 断言摘要 + wire body 摘录 + pageerror 计数 + Playwright 退出码。结尾一行：`WALKTHROUGH: PASS|FAIL (n/m)`。不改仓库文件、不改 progress.md、不派生 subagent、不做 GitHub 写操作。最终回复：PASS/FAIL + 一句要点。
