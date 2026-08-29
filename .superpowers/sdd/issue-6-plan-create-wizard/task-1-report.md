# Task 1 Report — Issue #6 计划创建向导（free/flat 价卡）

接管模式：前任 implementer 已写完 7 个文件未提交（死于基础设施故障）。本次接手核对 → 修复 11 个 TS 错误 → 三连验证 → 提交。

## What was implemented

逐文件（commit `6457afe9f`，7 files changed, 906 insertions）：

| 文件 | 动作 | 核对结论 |
|---|---|---|
| `web/src/features/config/plans/plan-form-schema.ts` | Create | 与处方评论 1 代码块逐字一致（两段拼接完整：schema/类型 + 提交映射）。导出齐全：`priceFormSchema`、`PriceFormValue`（评论 2 跨工单契约 ✓）、`rateCardSchema`、`RateCardFormValues`、`phaseSchema`、`PhaseFormValues`、`phasesSchema`、`planWizardSchema`、`PlanWizardValues`、`defaultRateCard()`、`defaultPhase()`、`EMPTY_PLAN`、`toPlanPhases()`、`toCreatePlanRequest()`。wire 契约 ✓：`toRateCardInput` 不发送 `type`/`featureId`；`billingCadence: null → undefined`；金额为字符串（SDK `RateCardInput.billingCadence?: string`、`price: Price` 实测核对）。未改动（前任实现零偏差） |
| `web/src/features/config/plans/price-editor.tsx` | Create | 与处方逐字一致，仅 useWatch name 一处机械修正（见 Deviations #5/#6） |
| `web/src/features/config/plans/plan-form-wizard.tsx` | Create | 与处方逐字一致（含缩进细节），修正 mutate 调用 + 7 处 cast 移除 + 1 处 lint disable（见 Deviations） |
| `web/src/api/hooks.ts` | Modify | `useCreatePlan` 置于 `usePlan` 之后，与处方逐字一致。未改动 |
| `web/src/features/config/plans/index.tsx` | Modify | 新建按钮 + `createOpen` state + PageHeader actions + 末尾挂 `<PlanFormWizard>`，与处方步骤 3 逐字一致。未改动 |
| `web/src/i18n/locales/zh-CN.ts` | Modify | `config.plans.wizard` 子树插在 `toast` 之后，与处方逐字一致。未改动 |
| `web/src/i18n/locales/en.ts` | Modify | 同上，两 locale 同键。未改动 |

关键契约自查：zod message 全部为 i18n key（FieldError 翻译）✓；`open→true` 时 `form.reset(EMPTY_PLAN)` + 回 `basics` 步 ✓；提交成功 toast `config.plans.wizard.toast.created` + 关闭 dialog + invalidation 刷新 ✓。

## Verification evidence（全部亲跑）

| 命令 | 退出码 | 日志 |
|---|---|---|
| `pnpm build` | 0 | `/tmp/issue6-build.log` |
| `pnpm lint` | 0 | `/tmp/issue6-lint.log` |
| `pnpm test:e2e` | 0 | `/tmp/issue6-e2e.log` |

- e2e 通过条数：**2 passed**（`sign-in smoke: OIDC round-trip lands on the dashboard`、`customers smoke: route reachable and renders list data`），既有冒烟无回归。
- 端口检查：`:9999`/`:4173` 跑前空闲（无 PID），跑后无残留进程。
- `web/src/routeTree.gen.ts`：本会话 3 次 build 后均与 HEAD 逐字节一致（未复现控制器记录的 +369/−369 重排），commit 不含该文件。
- build 中的 11 个 TS 错误全部修复，最终态重跑 build 确认（日志为最终提交态）。

## Commit

```
6457afe9f feat(admin): 计划创建向导（free/flat 价卡）
a319e1b38 docs(admin): issue #6 计划创建向导实施计划
a6ff556ef feat(admin): 计划发布/归档/克隆新版本
```

`git status --porcelain` 输出（提交后）：

```
?? .superpowers/sdd/issue-6-plan-create-wizard/
```

（仅报告目录未跟踪、按约定不提交；除此外工作树干净。）

Commit 恰含 7 个任务文件（`git show --stat`：hooks.ts +13 / index.tsx +11 / plan-form-schema.ts +185 / plan-form-wizard.tsx +504 / price-editor.tsx +105 / en.ts +44 / zh-CN.ts +44）。

## Deviations / concerns

### (a) 前任遗留偏差

**0 条。** 逐文件对照处方核对：前任 7 个文件全部逐字一致，无笔误。11 个 TS 错误全部源于处方代码与实际类型面（SDK d.ts / react-hook-form 7.72.1 / TS 模板字面量拓宽规则）的不匹配，非前任引入。

### (b) 类型/lint 强制机械修正（7 条）

全部以 SDK `web/node_modules/@openmeter/client/dist/**/*.d.ts` 实际类型与安装版 RHF 类型为准（issue #2/#4 既定裁定）：

1. **wizard L101 mutate 参数**（修 build2.log 错误 1）：`createPlan.mutate({ body: toCreatePlanRequest(values) }, ...)` → `createPlan.mutate(toCreatePlanRequest(values), ...)`。SDK `plans.create(request: CreatePlanRequest, options?)` 扁平收参：`CreatePlanRequest = AcceptDateStrings<CreatePlanRequestInput>` 无 `body` 包装（对照 `UpdatePlanRequest = { planId, body }` 才有）。与既有 `feature-form-dialog.tsx` 的注释先例（"The v3 SDK flattens CreateFeatureRequest fields at the top level (no `body` wrapper), unlike update"）完全同构。
2. **移除 7 处 `as FieldPath<PlanWizardValues>` cast**（phases 步 name/key/duration 3 处 + rateCards type/name/key/billingCadence 4 处；修错误 L238/251/264/414/429/387/446）：cast 使 RHF 泛型 `TName` 推断为**全部路径的联合**，`field.value` 随之成为所有字段值的联合（含 `null`/对象），Input/Select 的 `value` 不兼容。RHF 7.72.1 的 `FieldPath` 原生包含 `` `phases.${number}.xxx` `` 模板路径（`dist/types/path/eager.d.ts` 实测），内联表达式无需 cast 即精确推断。
3. **useFieldArray name（原 L356）**：移除同款 cast——`useFieldArray` 要求 `FieldArrayPath`，全路径联合不满足；`` `phases.${phaseIndex}.rateCards` `` 原生合法。
4. **useWatch name（原 L360）**：`as never` → 直接内联 `` `phases.${phaseIndex}` ``，返回精确 `PhaseFormValues`（`as never` 使 useWatch 落入 `name?: undefined` 重载返回 `DeepPartialSkipArrayKey`，导致 L372/387/446 三处 `.duration`/`.name`/Select value 错误）。
5. **price-editor useWatch name（原 L49）**：`` `${pricePath}.kind` as never `` → 内联 `` `phases.${phaseIndex}.rateCards.${cardIndex}.price.kind` ``。根因（tsc 最小实验证实）：模板字面量一经 `const` 变量中转即拓宽为 `string`（`as const` 亦无效），只有内联表达式在泛型推断位保留模板字面量类型。保留处方尾部的 `as PriceFormValue['kind']`（现为恒等 cast，合法且保持 `PriceFormValue` import 的用途）。
6. **import 清理（处方授权 + eslint 强制）**：上述 cast 移除后 `type FieldPath` 不再使用，从 `react-hook-form` import 中删除（`@typescript-eslint/no-unused-vars` = error）。
7. **lint：`set-state-in-effect`**：eslint-plugin-react-hooks 7.1.1（recommended, error）禁止 effect 内直接 setState，命中处方 `setStep('basics')`。加 scoped 注释 `// eslint-disable-next-line react-hooks/set-state-in-effect -- reopen must return to the first wizard step`。理由：`form.reset` 部分不触规且与 `feature-form-dialog.tsx` 既有同款 effect 惯用法一致；备选方案（条件挂载/`key` 重挂）需改 2 个文件、删除处方 effect 并损失 Radix 关窗动画，偏差更大，故取最小修正。仓库有 scoped disable + `-- reason` 先例（`server-table.tsx`、`auth-store.ts` 等）。

### (c) 其他偏差

**0 条。** i18n 两份、hooks.ts、index.tsx 未做任何改动；price-editor 保留处方的 FormField `as never` name（不报错，不动）。

### 环境干预与未决事项

- 端口：无残留需清理（跑前跑后 9999/4173 均空闲）。
- 依赖：无需安装（控制器已就绪 SDK 注入与 node_modules），未重复处理。
- routeTree.gen.ts：无重排发生、无需恢复；一次在 `web/` 子目录误用根相对路径的 `git checkout` 报 pathspec 错，经从根目录核实文件本就 clean，无实际影响。
- 未决：处方步骤 5 的「浏览器手测」（两阶段计划真实创建、反向校验用例）不在本任务验证三连范围，未执行；后续如需可人工补测。向导运行时行为以 build/lint/e2e 三连全绿为准。
