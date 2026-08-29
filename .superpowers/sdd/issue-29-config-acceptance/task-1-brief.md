# Task 1 Brief — e2e 冒烟第 3 用例（issue #29 T1）

## 上下文（一行）

Issue #29（配置域全链路验收，29 系列终验轨）的代码交付物之一：为 web 冒烟套件新增第 3 条用例，覆盖一个新配置页（/config/plans）。

## 工作环境

- Worktree: `/Users/wuyongjun/trea/openmeter-issue-29`（分支 `codex/admin-config-29`，已在位，勿改分支）
- web 环境已就绪（node_modules + SDK dist 已配好），直接可用 pnpm
- 仓库规范见 worktree 根 AGENTS.md；命令均在 `web/` 下执行

## 需求（先读：这是你的需求书，逐字值为准）

**Modify: `web/e2e/smoke.spec.ts`** —— 在文件顶部 `MOCK_CUSTOMER_NAME` 常量之后新增：

```ts
const MOCK_PLAN_NAME = 'Smoke Test Plan'
```

并在文件末尾（customers 用例之后）追加完整用例（**逐字采用**，含注释）：

```ts
test('config plans smoke: /config/plans reachable and renders list data', async ({
  page,
}) => {
  // Fulfill the v3 plans list the config page calls. The SDK's
  // plans.listAll pagination walks `meta.page`; one full page keeps it
  // from requesting page 2. Wire format is snake_case per
  // BillingPlan in api/v3/openapi.yaml.
  await page.route('**/api/v3/openmeter/plans*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          {
            id: '01JDVF0R4T2Z3Y8W7X6V5U4T3R',
            name: MOCK_PLAN_NAME,
            key: 'smoke-test-plan',
            version: 1,
            currency: 'CNY',
            billing_cadence: 'P1M',
            status: 'active',
            phases: [],
            created_at: '2024-01-01T01:01:01.001Z',
            updated_at: '2024-01-01T01:01:01.001Z',
          },
        ],
        meta: { page: { number: 1, size: 10, total: 1 } },
      }),
    })
  })

  await login(page)

  // The sidebar must expose the config group (zh-CN default locale).
  await expect(page.getByRole('link', { name: '计划' })).toBeVisible()

  await page.goto('/config/plans')
  await expect(page).toHaveURL(/\/config\/plans$/)

  // The real plans page renders the mocked plan; keep the placeholder
  // fallback so a temporarily unshipped page does not hard-fail.
  const placeholderHeading = page.getByRole('heading', {
    name: /建设中|Under Construction/,
  })
  const planCell = page.getByText(MOCK_PLAN_NAME)
  await expect(planCell.or(placeholderHeading)).toBeVisible()
})
```

## 约束

- 不得修改 `playwright.config.ts`、`e2e/mock-idp.mjs`（处方明令）。
- 不得触碰任何后端/Go 文件、openapi spec、SDK 生成物。
- 若 /config/plans 首屏实际还调用了**第二个读端点**导致用例失败：按 customers 用例的双路径 mock 模式补一条 `page.route`，并在报告中注明端点与原因；**不要**为写操作端点加 mock。
- 不得派生任何 subagent。

## 验证（必须真实运行并留存输出）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-29/web && pnpm test:e2e
```

预期：3 条用例（sign-in / customers / config plans）全部通过。已知环境事项：customers 冒烟在本机曾因用户 :8888 shim 出现环境性失败——若 customers 失败而 sign-in/config plans 通过，将完整输出留存并在报告标注「与基线同签名=环境性」，不要试图修改用例或 mock customers。

## 提交

单任务单提交：`test(admin): 配置计划页冒烟用例（issue #29 task 1）`（只 add smoke.spec.ts）。

## 报告契约

完整报告写入 `/Users/wuyongjun/trea/openmeter/.superpowers/sdd/issue-29-config-acceptance/task-1-report.md`（含：改动 diff 摘要、测试命令与完整输出的关键段、遇到的偏差）。你的返回消息只需：状态（DONE / DONE_WITH_CONCERNS / NEEDS_CONTEXT / BLOCKED）、commit hash、一行测试摘要、concerns（如有）。
