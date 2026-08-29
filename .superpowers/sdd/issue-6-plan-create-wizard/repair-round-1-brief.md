# Repair Round 1 Brief — Issue #6 计划创建向导（free/flat 价卡）

你是修复轮 implementer（独立新 subagent）。对象：Issue #6 Task 1 已提交的实现
`6457afe9f`（分支 `codex/admin-config-06`）。质量审查发现 1 Critical + 1 Important +
1 Minor，全部为**处方内在缺陷**（实现逐字忠实已由 spec review 证明），现按审查协议
授权修复。你只修这 3 条，不做任何其他改动。

## 环境

- Worktree：`/Users/wuyongjun/trea/openmeter-issue-6`（所有命令在此目录执行；web 命令在 `web/` 下）。
- 环境（SDK dist 注入、node_modules）已由控制器就绪，勿重装依赖。
- 端口 9999/4173 当前空闲；e2e 前后自查无残留。
- 先读：`.superpowers/sdd/issue-6-plan-create-wizard/quality-review-report.md`（发现 C1/I1/M1
  的完整根因分析与修复建议）、`task-1-report.md`（实现偏差先例）。

## 修复项（仅此 3 条）

### C1（Critical）：向导步骤 2→3 门禁死锁

`web/src/features/config/plans/plan-form-wizard.tsx` 的 `next()`：

```ts
} else if (step === 'phases') {
  if (await form.trigger('phases')) {   // 恒 false：resolver 返回完整错误树，默认价目卡尚未挂载
    setStep('rateCards')
  }
}
```

改为**只校验该步已挂载的叶子路径**（审查员建议语义）：

```ts
} else if (step === 'phases') {
  const phaseFieldNames = phaseFields.flatMap((_, i) => [
    `phases.${i}.name`,
    `phases.${i}.key`,
    `phases.${i}.duration`,
  ])
  if (await form.trigger(phaseFieldNames)) {
    setStep('rateCards')
  }
}
```

**已知类型坑**（task-1-report.md 偏差 #4/#5 的教训）：模板字面量经 `const` 变量中转即拓宽为
`string`，`form.trigger` 需要 `FieldPath<PlanWizardValues>[]`。若直接照抄报 TS 错，取**最小**
类型修正（例如给 `flatMap` 显式类型参数，或等价写法），保持内联模板字面量的上下文类型推断；
允许必要的类型标注，逐条记录。禁止 `as never`；禁止把门禁改回整子树 trigger；禁止改动
zod schema 或步骤 3 逻辑。行级 superRefine 错误恰好落在 `phases.${i}.duration` 叶子路径，
仍在门禁内；价目卡合法性由步骤 3 挂载后的字段校验 + 最终 `handleSubmit` 全量校验兜底
（彼时字段已挂载、错误可见）。`phases` 数组级 min(1) 在该步不可触发（仅剩 1 行时删除按钮
disabled），不受影响——这些结论审查员已核实，你复跑验证即可。

### I1（Important）：description 超 1024 字符最终提交静默失败

两处**都做**（各一行）：

1. `plan-form-wizard.tsx` 步骤 1 门禁列表加入 `'description'`：
   `form.trigger(['name', 'key', 'currency', 'billingCadence', 'description'])`
   （description 字段在步骤 1 已挂载，错误可见）。
2. 同文件 description 的 `<Textarea>` 加 `maxLength={1024}`（输入期即拦截）。

### M1（Minor）：prettier 新增违规 4 文件

```bash
cd web && pnpm exec prettier --write \
  src/features/config/plans/plan-form-schema.ts \
  src/features/config/plans/price-editor.tsx \
  src/features/config/plans/plan-form-wizard.tsx \
  src/i18n/locales/en.ts
```

纯格式化、零语义变化（审查员明确建议）。zh-CN.ts 干净勿动；hooks.ts 基线违规勿动
（计划明令不得使既有 prettier 状况变化，也不修基线）。

## 约束

- 只允许触碰上述 3 个语义点 + prettier 重排的 4 文件；不得改 zod schema、i18n 文案、
  hooks.ts、index.tsx、e2e、routeTree.gen.ts（若 build 再生成它，提交前恢复到 HEAD：
  `git checkout -- web/src/routeTree.gen.ts`，从 worktree 根执行）。
- 不得派生 subagent；不得 push / merge / 改 GitHub 状态；不得改 progress.md。
- 提交前 `git status --porcelain` 必须仅含预期文件（`.superpowers/sdd/` 报告目录除外，不提交）。
- commit 单条，subject：`fix(admin): 计划向导步骤门禁与描述长度校验（审查修复轮 1）`。

## 验证（全部亲跑，日志存 /tmp）

1. `cd web && pnpm build` → /tmp/issue6-fix1-build.log
2. `cd web && pnpm lint` → /tmp/issue6-fix1-lint.log
3. `cd web && pnpm test:e2e` → /tmp/issue6-fix1-e2e.log（既有 2 条冒烟必须仍绿）
4. `cd web && pnpm exec prettier --check src/features/config/plans/plan-form-schema.ts src/features/config/plans/price-editor.tsx src/features/config/plans/plan-form-wizard.tsx src/i18n/locales/en.ts` → exit 0
5. （C1 定向自证，可选但推荐）写一个 /tmp 下的一次性 Playwright 探针复用 e2e mock 栈
   （参照 /tmp/issue6-wizard-probe2.cjs 的方式），证明步骤 2→3 现在可通行；脚本与截图放
   /tmp，不进仓库。

## 输出

报告写到 `.superpowers/sdd/issue-6-plan-create-wizard/repair-round-1-report.md`：
修复内容逐条（C1/I1/M1 → 实际 diff 位置）、typing 修正细节（若偏离审查员建议片段）、
验证证据表（命令/退出码/日志路径）、commit hash 与 `git show --stat`、`git status --porcelain`
终态。结尾一行：`REPAIR ROUND 1: DONE|BLOCKED`。最终回复：一行结论 + commit hash。
