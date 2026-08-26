# Issue #1 配置分组骨架与九个路由占位 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: subagent-driven-development. Implement task-by-task with isolated git worktree, per-task implementer/reviewer subagents, and an SDD ledger at `.superpowers/sdd/issue-1-config-group-skeleton/progress.md`.

**Issue:** https://github.com/1123786563/openmeter/issues/1 — `[admin-config 01/29] 配置分组骨架与九个路由占位`（label `ready-for-agent`，无 Blocked by）
**Authoritative detail:** the issue's comment carries the fully prescriptive implementation plan (exact file contents for every change); `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 1 is the master-plan section. Where any wording differs, the issue comment wins; this file adds scope control only.

**Goal:** 侧边栏新增「配置」分组（计划/功能/附加组件/通知/货币/税码/应用/门户令牌/账单档案九入口），一次性建齐 11 个路由占位文件（notification 域含 channels/rules/events 三个），命令面板（⌘K）经既有 `sidebarData` 消费自动可导航。

**Tech Stack:** React 19 + TanStack Router 文件路由 + i18next（zh-CN/en 双语）+ Vite + Playwright（mock OIDC IdP + vite preview）。

## Scope

- Parameterize `web/src/components/placeholder-page.tsx`（新增可选 `titleKey`/`descriptionKey` props，默认值保持现行为；该组件当前无调用方）。
- `web/src/i18n/locales/zh-CN.ts` / `en.ts`：新增 `sidebar.groups.config`、9 个 `sidebar.*` 条目 key、顶层 `config.*` 命名空间（11 个页面的 title/description）。
- `web/src/components/layout/data/sidebar-data.ts`：导入 7 个新 lucide 图标，`navGroups` 末尾追加「配置」分组（9 条目，顺序=计划/功能/附加组件/通知/货币/税码/应用/门户令牌/账单档案）。
- 创建 11 个路由文件 `web/src/routes/_authenticated/config/{plans,features,addons,notification/{channels,rules,events},currencies,tax-codes,apps,portal-tokens,billing-profiles}/index.tsx`，全部渲染参数化 `PlaceholderPage`。
- `web/src/routeTree.gen.ts` 由 `pnpm build` 内的 `tsr generate` 重写，随本次提交。

## Non-goals

- 不实现任何真实功能页面（列表/表单/请求 hooks 均属 #2–#28）。
- 不改动现有五个侧边栏分组（overview/billing/metering/credits/commerce）的任何内容。
- 不改动 `command-menu.tsx`（自动消费 sidebarData）。
- 不新增 e2e 用例（既有 2 条冒烟不得回归）。
- 不创建 `web/src/features/config/` 空目录（git 不跟踪空目录，由 #2 起自然创建）。
- 无后端/API/SDK 改动。

## Task split

单任务（Task 1）：上述全部改动为**一个原子提交**（issue 步骤 8 的 `git add` 清单），由一个 implementer subagent 完成。拆得更细反而会产生中间态（如 i18n key 无路由引用或路由引用缺 key），审查无法独立成立。

## Verification & acceptance commands（在 worktree 的 `web/` 下）

```bash
pnpm install          # worktree 首次安装（@openmeter/client 为 file: 依赖，如 build 报模块缺失先构建其 package）
pnpm build            # tsr generate 重写 routeTree.gen.ts + tsc -b + vite build
pnpm lint             # eslint，0 error
pnpm test:e2e         # 既有 sign-in + customers 冒烟（自动起 mock IdP:9999 + preview:4173）
```

验收（issue Acceptance criteria）：

1. 侧边栏出现「配置」分组，九个条目均可点击进入占位页；
2. 命令面板（⌘K）可搜索并导航到全部新页面；
3. zh-CN 与 en 文案齐全；
4. `pnpm build` / `pnpm lint` / `pnpm test:e2e` 全绿。

浏览器走查（headless 截图留证 + 人工目视复核）：侧边栏分组与 9 条目、任一新页面标题/描述与 `config.*` key 一致、命令面板搜索「计划」「账单」可导航、语言切换 en 后分组标题变「Configuration」。

## Global constraints

- 慢命令/大输出：完整日志落 `/tmp/issue1-*.log`（仓库外），报告退出码与日志路径，检查只看有界切片；不得因显示截断而重跑。
- 所有命令在 worktree `/Users/wuyongjun/trea/openmeter-issue-1` 的 `web/` 下执行；**绝不触碰主检出** `/Users/wuyongjun/trea/openmeter`（其处于未完成上游合并中）。
- 提交信息按 issue 步骤 8：`feat(admin): 配置分组骨架与九个配置路由占位`。
- i18n zh-CN 与 en 两份同步维护；文案 key 与 issue 评论逐字一致。
- 不 push、不 merge、不改 GitHub 状态（外部副作用需用户批准）。
- 遵守仓库 AGENTS.md 约定（web 子域无 Go 相关约束适用）。

## Branch / worktree

- Worktree: `/Users/wuyongjun/trea/openmeter-issue-1`（base `7cd8ae172` = main 尖端）
- Branch: `codex/admin-config-01`
- SDD ledger: `/Users/wuyongjun/trea/openmeter/.superpowers/sdd/issue-1-config-group-skeleton/progress.md`（主仓库，gitignored）
