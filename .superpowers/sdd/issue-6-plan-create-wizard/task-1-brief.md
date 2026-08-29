# Task 1 Brief — Issue #6 计划创建向导（free/flat 价卡）实施

你是 implementer subagent。工作目录：`/Users/wuyongjun/trea/openmeter-issue-6`（git worktree，分支 `codex/admin-config-06`，基底 `a6ff556ef`）。**所有命令在此目录下执行**（web 相关命令在 `web/` 子目录）。

## 硬性约束（违反即失败）

1. **不得派生任何 subagent**（no subagents / no delegation）。
2. **不得 push、merge、不得执行任何 gh 写操作 / 修改 GitHub 状态**（`gh issue view` 只读允许）。
3. 不得改动本任务 6 个文件之外的生产文件（`routeTree.gen.ts` 等 gitignored 产物除外——本任务不应触碰路由）。
4. 处方代码逐字实现：唯一允许的偏差是处方明确要求的「清理上文标注的未用 import」与 lint/构建强制的机械修正；任何偏差必须在报告中逐条记录。
5. 完成后工作树必须干净（改动全部进入唯一功能 commit）。

## 任务

按 Issue #6 评论 1 的处方（**权威 spec，先 `gh issue view 6 --repo 1123786563/openmeter --json body,comments` 通读**）实现：

### 文件清单

| 动作 | 文件 |
|---|---|
| Create | `web/src/features/config/plans/plan-form-schema.ts` |
| Create | `web/src/features/config/plans/price-editor.tsx` |
| Create | `web/src/features/config/plans/plan-form-wizard.tsx` |
| Modify | `web/src/api/hooks.ts`（`useCreatePlan`，放在 `usePlan` 之后） |
| Modify | `web/src/features/config/plans/index.tsx`（PageHeader actions 新建按钮 + 挂载向导） |
| Modify | `web/src/i18n/locales/zh-CN.ts` 与 `en.ts`（`config.plans.wizard` 子树，两 locale 同键） |

Issue 评论中的代码块就是这些文件的完整内容（schema、PriceEditor、PlanFormWizard、useCreatePlan、i18n 两份、列表页挂载步骤）。i18n 的 `wizard:` 子树放在 `config.plans` 对象内（`toast` 之后即可），缩进与邻接键一致。

### 关键契约（处方内已明确，实现时逐条自查）

- `plan-form-schema.ts` 必须导出：`priceFormSchema`、`PriceFormValue`（**评论 2 强制的跨工单契约**）、`rateCardSchema`、`RateCardFormValues`、`phaseSchema`、`PhaseFormValues`、`phasesSchema`、`planWizardSchema`、`PlanWizardValues`、`defaultRateCard()`、`defaultPhase()`、`EMPTY_PLAN`、`toPlanPhases()`、`toCreatePlanRequest()`。
- zod message 一律是 i18n key（`config.plans.wizard.errors.*`），由 `FieldError` 翻译。
- wire 映射：`RateCardInput` **不发送 `type` 字段**；`billingCadence: null` → `undefined`；金额是字符串。
- 向导重置：`open` 变 true 时 `form.reset(EMPTY_PLAN)` + 回到 `basics` 步。
- 提交成功：toast `config.plans.wizard.toast.created` + 关闭 dialog（列表经 `useCreatePlan` 的 invalidation 自动刷新）。

### 验证（全部必须亲跑并在报告附日志路径与退出码）

```bash
cd web
pnpm build      # exit 0
pnpm lint       # exit 0
pnpm test:e2e   # exit 0，既有 2 条冒烟通过（若 9999/4173 端口被占用：先 lsof -ti :9999/:4173 查证并 kill 既有残留，记录 PID）
```

注意：`pnpm build` 前若 node_modules 缺失先 `pnpm install --frozen-lockfile`（记录）。e2e 期间 mock-idp 与 vite preview 由脚本自行管理，结束后不应残留进程。

### 提交

- Commit message（逐字）：`feat(admin): 计划创建向导（free/flat 价卡）`
- 单 commit 包含全部改动（6 个文件；如 prettier/eslint 对既有未触碰文件报红属既有基线，不修不改）。
- 计划文档 commit（`docs(admin): issue #6 计划创建向导实施计划`）已由控制器完成，勿动。

## 报告

把报告写到 `.superpowers/sdd/issue-6-plan-create-wizard/task-1-report.md`（该目录已存在 progress.md；只写自己的报告文件，不改 progress.md），内容：

1. What was implemented：逐文件清单与要点（含相对处方的任何偏差及理由——目标零偏差）。
2. Verification evidence：build/lint/test:e2e 各自的退出码 + 日志文件路径（保存到 /tmp/issue6-{build,lint,e2e}.log）+ e2e 通过条数。
3. Commit：最终 commit hash（`git log --oneline -3` 输出）+ `git status --porcelain` 输出（必须为空）。
4. Deviations/concerns：任何环境干预（端口清理、依赖安装）与未决事项。

完成后最终回复只需给出：commit hash、三连退出码、报告文件路径、偏差数。
