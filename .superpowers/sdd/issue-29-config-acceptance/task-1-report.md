# Task 1 Report — e2e 冒烟第 3 用例

- 状态：DONE
- Commit：d93e912c3（单提交，仅 web/e2e/smoke.spec.ts，+74 行）
- 测试：`pnpm test:e2e` → **3 passed (6.3s)**（sign-in ✓ customers ✓ config plans ✓）
- 定向 eslint e2e/smoke.spec.ts：0 输出（干净）

## 实现摘要

- `MOCK_PLAN_NAME` 常量按处方位置插入。
- 第 3 用例逐字采用处方代码（含全部注释与占位兜底断言）。
- **处方授权偏差（第二读端点）**：/config/plans 首屏除 plans 外还触发 GET /api/v3/openmeter/namespaces（认证布局 Header 的 namespace-switcher）。未 mock 时该请求经 vite preview 的 /api 代理（preview 继承 server.proxy → 127.0.0.1:8888）打到本地任意监听者（CI 无监听、本机为用户 dev shim）；shim 恒回 `{"data":[]}`，缺 NamespaceList wire 形状（strictObject `{default, namespaces}`），switcher 在 namespace-switcher.tsx:55 读 `namespaces.length` 抛 TypeError → 500 错误边界 → 用例失败。诊断证据：debug spec 捕获 request/response/pageerror 链（namespaces 200 → TypeError at header bundle），debug spec 已删除。
- **同根因扩展（customers 用例）**：customers 冒烟 20+ 轮的「环境性失败」真因相同（Header namespaces 崩溃）。处方步骤 2 自身预期「3 条用例全部通过」，故以共享 `mockNamespaces(page)` helper 同时修复两用例。修复后 customers 由恒败转恒过。
- 约束遵守：playwright.config.ts / mock-idp.mjs / 后端 / spec / SDK 零触碰。

## 审查记录（Standing DOWNGRADE，控制器分遍）

- Spec 遍：PASS——处方代码逐字落位；偏差均在处方明文授权框架内（「按 customers 用例的双路径 mock 模式补一条 page.route，并在 PR 里注明端点」；步骤 2 的 3 绿预期）；Produces 契约（用例名/总数 2→3）达成。
- Quality 遍：PASS——eslint 0；helper 文档注释说明失效模式与 wire 形状；断言真实（数据或占位二择一可见，占位兜底为处方明令）；无测试卫生问题；与文件既有风格一致。
- Minor（记录不修）：处方注释称 plans.listAll 分页，实际管理页走 plans.list——注释为处方逐字文本，保留原样。

## 剩余关注

无 Critical/Important。namespaces 真因的澄清建议写入 issue 完成评论（外化轮）。
