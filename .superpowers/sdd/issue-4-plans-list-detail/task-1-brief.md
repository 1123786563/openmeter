# Task 1 Brief — Implementer: 计划列表与详情（只读）

你是 implementer subagent。严格按以下要求实施，**禁止**：派生任何子 subagent、任何 GitHub 写操作（gh issue/pr/push/merge/close/comment 一律禁止）、改动本任务范围外的文件。

## 上下文

- Worktree（在这里工作，勿动主 checkout /Users/wuyongjun/trea/openmeter）: `/Users/wuyongjun/trea/openmeter-issue-4`，branch `codex/admin-config-04`，HEAD `96e1111c0`（plan doc 已提交）。
- Issue: https://github.com/1123786563/openmeter/issues/4（用 `gh issue view 4 -R 1123786563/openmeter --json body,comments` 只读读取正文与评论；评论含处方化完整代码——评论代码是主要规范）。
- 实施计划: `docs/superpowers/plans/issue-4-plans-list-detail.md`（已提交，必读——尤其「SDK 契约核实记录」与「已知偏差风险」节）。
- 仓库规范: AGENTS.md（worktree 根）。#1–#3 已确立的 web 模式是你的基准（features/config/features/ 是最近的同类实现）。

## 任务（issue 评论步骤 1–7 的忠实实施）

1. `web/src/api/query-keys.ts`：`queryKeys` 内新增 `plansPage` 与 `plan` 两行（评论原文）。
2. `web/src/components/status-badge.tsx`：`tones` 新增 `plan` 域（评论原文，4 个状态）。
3. `web/src/api/hooks.ts`：`usePlans` 之后新增 `PlanListParams`/`usePlansPage`/`usePlan`（评论原文；契约已核实类型合法）。
4. 新建 `web/src/features/config/plans/index.tsx`：`PlansPage`（评论完整代码）。
5. 新建 `web/src/features/config/plans/plan-detail.tsx`：`PlanDetail`（评论完整代码）。
6. 路由：`web/src/routes/_authenticated/config/plans/index.tsx` 整文件替换；新建 `$planId.tsx`（评论原文）。
7. i18n：zh-CN.ts 与 en.ts——`config.plans` 扩展为评论完整块（替换现有仅 title/description 的占位块），顶层新增 `plan` 域（与 `subscription` 平级）。两份必须键结构完全一致。

## 已知偏差处理规则（plan「已知偏差风险」节）

- 若 `useParams({ from })` 或路由 `component:` 写法触发 eslint/TS 报错：改用 #3 的仓库规范模式（路由文件内 `Route.useParams()` + named `RouteComponent` + 同款 eslint-disable pragma，参照 `web/src/routes/_authenticated/config/features/$featureId.tsx`），把 featureId 换成 planId。**任何偏差都必须写入报告**。
- prettier 按仓库配置格式化新文件；hooks.ts 预存 recharge hunk 违规不要动（最小 diff 原则）。
- `tsr generate` 由 `pnpm build` 自动跑；routeTree.gen.ts 的改动限于新增 `$planId` 节点相关。

## 验证（必须全绿才算完成）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-4/web
pnpm build 2>&1 | tee /tmp/issue4-build.log
pnpm lint 2>&1 | tee /tmp/issue4-lint.log
# e2e 前先确认端口卫生：lsof -ti :4173 :9999 | xargs kill -9 2>/dev/null（若有残留）
pnpm test:e2e 2>&1 | tee /tmp/issue4-e2e.log
```

三项全部 exit 0（e2e 2 passed）。任何失败：先诊断、修复、重跑；环境阻塞如实记录（命令+错误+已完成的验证），禁止伪报通过。

## i18n 校验（必须做）

用 node/tsx 写一次性脚本程序化比对两份 locale：`config.plans` 与顶层 `plan` 键树结构完全一致、无重复键、interpolation 变量（{{index}}/{{duration}}/{{count}}）一致。结果写入报告。

## 提交

- 单次 commit：`feat(admin): 计划列表与详情（只读）`（只含本任务文件，git status 干净后提交）。
- 提交前 `git diff --stat` 自查：改动文件 = 范围 1–7 列出的文件 + routeTree.gen.ts，无其他。

## 报告（写入 .superpowers/sdd/issue-4-plans-list-detail/task-1-report.md）

内容：改动文件清单（含 +/- 行数）、三项验证命令输出摘要（exit code + 关键行）、i18n 比对结果、偏差清单（若有，逐条：评论原文→实际采用+原因+证据）、git log -1 与最终 git status。报告只写这个文件，不改 progress.md。
