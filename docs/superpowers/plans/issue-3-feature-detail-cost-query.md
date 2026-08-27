# Issue #3 功能目录：详情与成本查询 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

- **Issue:** https://github.com/1123786563/openmeter/issues/3 `[admin-config 03/29] 功能目录：详情与成本查询`
- **主计划对应:** `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 2 的详情页部分（该主计划 Task 2 前半——列表/CRUD——已由 #2 交付）。
- **Worktree / 分支:** `/Users/wuyongjun/trea/openmeter-issue-3`，`codex/admin-config-03`，base = `80d27c7b6`（main，含 #2 成果）。

## 目标

功能详情页：基本信息卡（key/描述/创建/更新时间）+ 成本查询区（客户选择器 + 时间窗 → `POST /openmeter/features/{featureId}/cost/query` → 结果表格）。列表页 key 列加详情链接。

## 范围

1. `web/src/api/hooks.ts`：Features 分节末尾追加 `FeatureCostQueryParams` + `useFeatureCostQuery(featureId, params, options?)`（评论处方代码为准；query key 用已就绪的 `queryKeys.featureCostQuery`，from/to 走 `toISOString()` 保证可序列化；body 为 `MeterQueryRequest`，客户过滤用保留维度 `customer_id: { eq }`，`timeZone: 'UTC'`）。
2. `web/src/routes/_authenticated/config/features/$featureId.tsx`：新路由挂 `FeatureDetailPage`（`Route.useParams()` 传 featureId）。
3. `web/src/features/config/features/feature-detail.tsx`：`FeatureDetailPage`——`useFeature` 基本信息 + 成本查询区（`CustomerPicker` + datetime-local from/to 默认近 30 天 + 「查询」按钮提交式触发 `submitted` 状态，模式照抄 `features/meters/meter-detail.tsx`）。结果表列严格对应 `FeatureCostQueryRow`：时段 `from~to`、`usage`、`cost`+`currency`（null → `detail` 原文，无 detail → 「无定价」占位）、额外维度（剔除保留键 `subject`/`customer_id`）。costQuery 空/加载态处理。顶部 import 补 `import type { Customer } from '@openmeter/client'`。
4. `web/src/features/config/features/index.tsx`：key 列 cell 改为 `<Link to='/config/features/$featureId'>`（其余不动）。
5. i18n：`config.featureDetail.costQuery.*`（title/customer/from/to/run/period/usage/cost/dimensions/noPrice/empty）+ `config.features.fields.updatedAt`（若 #2 已有则跳过——已核实 **#2 未加 updatedAt**，需新增），zh-CN 与 en 同结构。

## 非目标

- 不改 features CRUD hooks（#2 产物，评论明确不再动）。
- 不做 granularity/groupByDimensions UI（评论契约未包含；不传即整段聚合）。
- 不动后端 / API spec。
- 不动 e2e 冒烟既有 2 条用例（仅保证不回归）。
- routeTree.gen.ts 由 `tsr`/构建自动再生，不手改。

## 已核实的接口事实（实施时直接引用）

- SDK `api.features.queryCost(request: QueryFeatureCostRequest, options?): Promise<FeatureCostQueryResult>`（`dist/sdk/features.d.ts:65`）。
- `MeterQueryRequest`（`dist/models/types.d.ts:4044`）：`from?/to?: Date; granularity?; timeZone: string; groupByDimensions?: string[]; filters?`。
- `FeatureCostQueryRow`（`types.d.ts:1095`）：`usage: string; cost: string | null; currency: string; detail?: string; from/to: Date; dimensions: Record<string,string>`（subject/customer_id 为保留键）。
- `CustomerPickerProps = { value: Customer | null; onChange: (c: Customer | null) => void; className? }`（customer-picker.tsx:23）。
- `formatDateTime/formatNumber`（`web/src/lib/format.ts:35/28`）。
- `queryKeys.featureCostQuery(id, params)` 已存在（query-keys.ts:45，#2 落位）。
- 前车之鉴（#2 教训）：issue 评论的 SDK 调用形态若与 `dist/*.d.ts` 冲突，以 SDK 实际类型为准，偏差记录进台账；本 issue 处方代码已经对照 SDK 核实一致。

## 任务拆分（单任务垂直切片）

- [ ] Task 1：上述范围 1–5 全部（一个 commit：`feat(admin): 功能详情与成本查询`）。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e   # 三连全 exit 0
```

- e2e 前清理残留端口进程（:4173 vite preview / :9999 mock-idp，前两次会话遗留问题）。
- 浏览器走查（Playwright，built dist + mock 或真实后端）：
  1. `/config/features` 列表 key 为链接 → 点击进入 `/config/features/$featureId`；
  2. 详情页基本信息渲染（name/key/描述/时间）；面包屑回列表；
  3. 默认近 30 天窗点「查询」→ 出数或空态文案；
  4. 选择已知客户再查询 → 行 usage/cost/currency 正常；cost null 行显示 detail 原文；
  5. zh/en 双语渲染无缺 key。
- 验收（Issue 原文）：详情页正确渲染功能信息；成本查询按客户+时间窗出数或显示空态。

## 全局约束

- 文案全部 i18n（zh-CN + en 同步维护，禁止硬编码 UI 文案；detail 原文透出除外）。
- API 仅经 `web/src/api/client.ts` 单例 SDK（自动 Bearer + X-Namespace）。
- 提交式查询：未点「查询」前不发 cost query 请求（`enabled` 控制）。
- 遵循 AGENTS.md Go/前端惯例；不引入 `any/as`；命名遵循 Type Translation Naming。
- 禁止 push/merge/关 Issue/GitHub 写操作——外部副作用需用户另行批准。
- implementer subagent 禁止再派生 subagent、禁止 GitHub 写、改动限 worktree + /tmp。
