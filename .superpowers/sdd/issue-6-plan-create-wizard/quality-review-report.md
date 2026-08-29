# Quality Review Report — Issue #6 计划创建向导（free/flat 价卡）Task 1

- 审查员：独立代码质量审查员（quality reviewer，规格符合性不在本报告范围）
- 审查对象：commit `6457afe9f`（`feat(admin): 计划创建向导（free/flat 价卡）`），diff 基线 `a319e1b38`
- Worktree：`/Users/wuyongjun/trea/openmeter-issue-6`（分支 `codex/admin-config-06`）
- 日期：2026-08-28（独立复跑当日）

## 独立复跑结果（全部亲跑，退出码）

| 命令 | 退出码 | 备注 |
|---|---|---|
| `cd web && pnpm lint` | **0** | eslint 全绿 |
| `cd web && pnpm test:e2e` | **0** | 2 passed（sign-in smoke / customers smoke）；跑前 :9999/:4173 空闲（无 PID），跑后无残留 |
| `cd web && pnpm exec prettier --check <6 个触碰文件>` | **1** | 4 文件新增违规（3 个新文件 + en.ts）；hooks.ts 违规为基线且未恶化（见轴 4） |
| `cd web && pnpm exec tsc --noEmit -p tsconfig.json` | **0（空检查）** | 根 tsconfig 为 solution-style（`files: []` + references），该命令实际不检查任何文件；已弃用此结果 |
| `cd web && pnpm exec tsc -b` | **0** | 权威类型门面（与 `pnpm build` 的 `tsc -b` 相同；项目无独立 typecheck script，build 已含 tsc）。另抽查实现者 `/tmp/issue6-build.log` 结论一致 |

附加运行时实证（非 brief 强制项，为轴 2 取证）：以 e2e 同款 mock 栈（`node e2e/mock-idp.mjs` :9999 + `pnpm preview` :4173，复用 e2e 构建的 dist）+ `/tmp` 下 Playwright 库脚本驱动向导（不触碰 worktree；脚本与截图均在 /tmp：`issue6-wizard-probe.cjs`、`issue6-wizard-probe2.cjs`、`issue6-probe-*.png`）。跑后端口已清理、`git status` 仅余未跟踪的报告目录。

## 每轴结论

### 轴 1 类型安全 — PASS

- 权威检查 `pnpm exec tsc -b` exit 0。注：brief 建议的 `tsc --noEmit -p tsconfig.json` 在本仓库是**空检查**（根 tsconfig `files: []`，纯 references 容器），本报告以 `tsc -b` 为准并在此记录。
- 实现者 7 条机械修正逐一独立复核，**全部正确**：
  1. `createPlan.mutate(toCreatePlanRequest(values), ...)` — 实测 SDK `dist/sdk/plans.d.ts`：`create(request: CreatePlanRequest, options?)`，且 `dist/models/operations/plans.d.ts` 中 `CreatePlanRequest = AcceptDateStrings<CreatePlanRequestInput>`（扁平、无 `body` 包装），对照 `UpdatePlanRequest = { planId, body }` 确有包装。处方原样 `{ body: ... }` 既过不了 tsc 也会在运行时发错载荷；修正是必要且正确的（与 `feature-form-dialog.tsx` 既有注释先例同构）。
  2. 移除 7 处 `as FieldPath<PlanWizardValues>` cast — RHF 7.72.1（实测安装版本）`FieldPath` 原生含 `` `phases.${number}.xxx` `` 模板路径，内联表达式推断精确；cast 反而把 `TName` 推成全路径联合导致 `field.value` 联合类型不兼容。移除是**类型安全性的提升**（cast 是类型谎言的温床），方向正确。
  3. `useFieldArray` name 去 cast — `FieldArrayPath` 原生接受该模板路径，正确。
  4/5. 两处 `useWatch` name 去 `as never` — `as never` 会让 useWatch 落入 `name?: undefined` 重载（返回 `DeepPartialSkipArrayKey`），内联模板字面量才保住精确类型。保留 price-editor 尾部 `as PriceFormValue['kind']` 恒等 cast，合法且维持处方 import 用途，可接受。
  6. `type FieldPath` import 随之删除 — eslint `no-unused-vars` 强制，正确。
  7. `set-state-in-effect` scoped disable + `-- reason` — 仓库有同款先例（`src/stores/auth-store.ts:85`、`user-auth-form.tsx:27`），lint exit 0 证实有效。备选（条件挂载/`key` 重挂）确实偏差更大，取最小修正合理。
- `Price` 判别联合映射穷尽性：`toPriceInput` 对 `kind` 两分支 switch 无 default，漏分支会以返回类型错误暴露（`RateCardInput['price']`）。#7/#8 扩展 kind 时若漏映射将编译报错——契约安全。

### 轴 2 运行时正确性 — FAIL（发现 C1、I1，详见发现清单）

**正向证据（这些是对的）：**

- **行级错误路径成立**（探针 2 实证）：两阶段、阶段 1 期限留空 → 「仅最后一个阶段可以无期限」精确出现 1 次于该行 duration 下。zod superRefine `path: [index, 'duration']` → `errors.phases[0].duration` → FormField(`phases.0.duration`) 的 `fieldState.error` 链路真实落行。
- **步骤 1→2 转换正常**（探针实证）：basics 填写后 `trigger(['name','key','currency','billingCadence'])` 放行。
- **Select 受控绑定**：rateCardType 直连 `value/onChange`；billingCadence `value={field.value ?? 'one_time'}` + `onChange` 中 `'one_time' → null`、`'P1M'/'P1Y' → 字面量`，null⇄one_time 转换正确；price kind 切换整体替换对象（`{kind:'free'}` / `{kind:'flat', amount:''}`），判别联合重置语义正确。
- **invalidation 链**（静态核实，端到端被 C1 阻断无法探针）：`nsPrefix('plans') = ['api', ns, 'plans']` 与 `queryKeys.plans()` 全等；`nsPrefix('plans-page')` 前缀匹配 `queryKeys.plansPage(params)`；`queryKeys.plan(plan.id)` 精确匹配 `usePlan`。`onSuccess: (plan)` 类型为 `Plan`（`CreatePlanResponse = Plan`），`plan.id` 类型安全。链路正确。
- **useWatch 未挂载行为**：`defaultValues: EMPTY_PLAN` 使 `_formValues` 全量存在（RHF 默认 `shouldUnregister: false`）；步骤 3 挂载时 `useWatch('...price.kind')` 立即返回默认值 `'flat'`（`defaultRateCard()`），amount 输入框随挂载即渲染。`phase?.name ?? ''` 空值兜底齐备。
- **reset 语义无共享变异/键残留**：RHF 7.72.1 `_reset` 源码实证 `updatedValues = cloneObject(formValues)` 深拷贝——模块级 `EMPTY_PLAN` 常量不会被表单编辑污染；重开时 `reset(EMPTY_PLAN)` + 回 `basics` 步，field array 以新 id 重建，且数组字段在 basics 步本就未挂载，无键闪烁可见面。

**负向证据（C1，探针 1/2 双证实）**：步骤 2→3 静默死锁，见发现清单。

### 轴 3 回归风险 — PASS

- `git diff a319e1b38..HEAD -- web/src/api/hooks.ts web/src/features/config/plans/index.tsx` **纯增量**：hooks.ts 仅在 `usePlan` 与 `usePublishPlan` 之间插入 `useCreatePlan`（+13）；index.tsx 仅加 4 个 import、`createOpen` state、PageHeader `actions`、末尾挂载 `<PlanFormWizard>`。既有 `usePlans/usePlansPage/usePlan`、发布/归档/克隆三个 mutation、列表列与筛选零改动（diff 全文核对）。
- i18n：两 locale 均只插入 `config.plans.wizard` 子树（紧跟 `toast` 之后），既有键零改动；wizard 子树键集 zh/en 完全对齐（各 34 键，无差集，脚本比对）；向导引用的既有键 `plan.priceType.free/flat`、`config.plans.detail.noDuration/rateCardName/cadence` 均存在（#4 产物）。
- commit 不含 `routeTree.gen.ts`，无生成文件扰动。

### 轴 4 lint/format 基线 — PASS（lint）/ Minor 发现（prettier）

- `pnpm lint` exit 0。
- prettier（`--check` 6 文件）：`plan-form-schema.ts`、`price-editor.tsx`、`plan-form-wizard.tsx`、`en.ts` 4 文件违规；`zh-CN.ts` 干净。
- **hooks.ts 基线裁定（方法论修正后）**：直接对 /tmp 基线副本跑 prettier 会因插件/样式表按路径解析而失真。用 `--stdin-filepath src/api/hooks.txt` 模拟树内路径后：基线 a319e1b38 同样报 recharge hunk 违规（同为 7 行 diff、同一 hunk）→ 与 brief 所述一致，**属基线且未恶化**。
- `src/features/config/**` 其余既有文件 prettier 全部干净 → 上述 4 文件违规是**本次新增**（内容逐字来自处方，属处方内在风格缺陷）。CI web job 仅 `lint + build + e2e`，不跑 `format:check` → 不挡 CI。分级 Minor（M1）。

### 轴 5 安全 — PASS

- 7 个触碰文件 grep 无 `eval`、`new Function`、动态 `import(`、`dangerouslySetInnerHTML`、secret。
- 用户输入路径：RHF 表单值 → zod 校验 → `toCreatePlanRequest` 纯函数映射 → SDK 类型化 `api.plans.create`。无原始 fetch 拼接、无 HTML 注入面。

### 轴 6 可维护性 — PASS

- 结构遵循既有 `features/config/plans` 目录与同域文件命名（`*-form-dialog.tsx` / `*-form-schema.ts` 惯例延伸）；shadcn 组件用法与既有表单一致；`@/` import 顺序由 sort-imports 插件约束（lint 过）。
- 导出清单与处方/issue 评论 1 契约一致（`priceFormSchema`/`PriceFormValue` 等为 #7-#11 前向契约，按 brief 不算死代码）；`flatFeePriceKind` superRefine 在 #6 UI 不可触发，但为 #7 的前向防线且处方明示，不算死代码。
- 注释解释意图（''=无期限、null=一次性 wire 等价、#7/#8/#9 扩展点），符合 AGENTS.md 文档约定。

## 发现清单

### C1（Critical，处方内在缺陷）：向导步骤 2→3 永久阻塞，「创建计划」不可达

- **现象**（探针实证 ×2）：步骤 2 中阶段字段全部合法（名称/key 填写、阶段 1 期限 P1M），点击「下一步」3 次重试，dialog 描述始终停留 `阶段列表 (2/3)`，且**无任何可见错误**（`p.text-destructive` 全空）。向导无法进入价目卡编辑步 → 无法创建任何计划 → 处方自己的验收标准「向导可完整走通并创建含固定费价卡的计划」不可满足。
- **根因**（RHF 7.72.1 + @hookform/resolvers 5.2.2 安装源码实证）：
  1. `trigger(name)` 返回 `!fieldNames.some(name => get(errors, name))`——用的是 resolver 返回的**完整**错误树（`createFormControl.ts` trigger 实现）；
  2. `_runSchema(names)` 把 names 传给 zodResolver，但 zodResolver/toNestErrors **不按 names/已注册字段过滤**（`toNestErrors` 中 `names` 仅用于数组项 root 包装，`fields` 仅附加 ref；zod 对全 schema 解析）；
  3. `planWizardSchema.phases` 深嵌 `rateCardSchema`，而默认价目卡 `defaultRateCard()` 的 `key=''、name=''、price={kind:'flat',amount:''}` 必然非法——这些字段要到步骤 3 才渲染、才可修复；
  4. 故 `await form.trigger('phases')` 恒 false。错误落在未挂载字段上，行级/数组级 FieldError 均不渲染 → 静默死锁。
- **处方内在裁定**：`if (await form.trigger('phases'))` 逐字来自处方（实现零偏差）。这是处方自身的门禁逻辑 defeats 处方自身的三步设计。非实现缺陷；但按 brief 评估是否 Critical——**是**：核心验收路径完全不可用，且实现者的验证三连（build/lint/e2e）结构性无法发现（冒烟不覆盖向导），报告亦如实声明「浏览器手测未执行」。
- **最小修复建议**（`web/src/features/config/plans/plan-form-wizard.tsx:92-95`）：步骤门禁只校验该步已挂载的叶子路径，不触发整个 `phases` 子树：

  ```ts
  } else if (step === 'phases') {
    const phaseFieldsNames = phaseFields.flatMap((_, i) => [
      `phases.${i}.name`,
      `phases.${i}.key`,
      `phases.${i}.duration`,
    ])
    if (await form.trigger(phaseFieldsNames)) {
      setStep('rateCards')
    }
  }
  ```

  行级 superRefine 错误恰好落在 `phases.${i}.duration` 叶子路径，仍在门禁内；价目卡合法性留待步骤 3 的字段渲染 + 最终 `handleSubmit` 全量校验（彼时字段已挂载、错误可见）。`phases` 数组级 min(1) 在该步不可触发（删行按钮在仅剩 1 行时 disabled），不受影响。

### I1（Important，处方内在缺陷）：description 超 1024 字符时最终提交静默失败

- **现象**：步骤 1 的门禁 `form.trigger(['name','key','currency','billingCadence'])` 不含 `description`（处方原文），`Textarea` 也无 `maxLength`。用户粘贴 >1024 字符描述可顺利走到步骤 3，点「创建计划」时 `handleSubmit` 全量校验在 `description.max(1024)` 失败——错误落在未挂载的 basics 字段上，无任何可见反馈，提交按钮「看起来坏了」。
- **处方内在裁定**：trigger 列表与 Textarea 均逐字来自处方。非实现缺陷；用户可见静默失败（与 C1 同类，触发面较窄）。
- **最小修复建议**：二选一（可并用）：① `plan-form-wizard.tsx:89` 步骤 1 trigger 列表加入 `'description'`；② `plan-form-wizard.tsx:213` Textarea 加 `maxLength={1024}`（输入期即拦截，最简）。

### M1（Minor，处方内在风格缺陷）：prettier 新增违规 4 文件

- 3 个新文件的处方代码自身超 printWidth=80（如单行 `.refine((value) => AMOUNT.test(value), 'config.plans.wizard.errors.amount'),`），`en.ts` 的 `keyFormat: 'Lowercase letters, digits and underscores only, e.g. pro_plan',` 同理。CI 不跑 `format:check` 故不挡门；但仓库存在该 script 且同目录既有文件全干净。
- **最小修复建议**：`cd web && pnpm exec prettier --write src/features/config/plans/plan-form-schema.ts src/features/config/plans/price-editor.tsx src/features/config/plans/plan-form-wizard.tsx src/i18n/locales/en.ts`（纯格式化，零语义变化）。

## 结论

实现本身忠实于处方（7 处类型/lint 机械修正全部正确且必要，类型面与回归面干净）；但处方自身的步骤门禁 `form.trigger('phases')` 在 RHF schema-resolver 语义下恒阻塞步骤 2→3，向导核心功能不可用，属 Critical 处方内在缺陷，需修复轮处理后方可交付。

QUALITY REVIEW: FAIL (1C/1I/1M)
