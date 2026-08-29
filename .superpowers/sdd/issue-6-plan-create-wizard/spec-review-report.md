# Spec Review Report — Issue #6 计划创建向导（free/flat 价卡）Task 1

- 审查员：独立 spec reviewer（规格符合性：实现是否逐字符合 Issue #6 处方，不多不少）
- Worktree：`/Users/wuyongjun/trea/openmeter-issue-6`（分支 `codex/admin-config-06`）
- 实现 commit：`6457afe9f`（父 = 计划 commit `a319e1b38`，其父 = `a6ff556ef` = main）✓ 已核实
- diff 基线：`a319e1b38`；权威 spec：`gh issue view 6`（评论 1 = 逐文件处方，评论 2 = PriceFormValue 导出契约）✓ 已通读
- 方法：处方 8 个代码块程序化提取到 `/tmp/spec-blocks/`，与实现文件做 `diff -u` 字节级对比；i18n 键集用括号配对 + 键提取脚本做程序化对照；复跑 `pnpm lint`；抽查三份 `/tmp/issue6-*.log`。

## 清单 1–10 逐项裁定

### 1. 文件集精确 — PASS

`git diff --stat a319e1b38..6457afe9f` 输出（摘录）：

```
 web/src/api/hooks.ts                               |  13 +
 web/src/features/config/plans/index.tsx            |  11 +
 web/src/features/config/plans/plan-form-schema.ts  | 185 ++++++++
 web/src/features/config/plans/plan-form-wizard.tsx | 504 +++++++++++++++++++++
 web/src/features/config/plans/price-editor.tsx     | 105 ++++++
 web/src/i18n/locales/en.ts                         |  44 ++
 web/src/i18n/locales/zh-CN.ts                      |  44 ++
 7 files changed, 906 insertions(+)
```

按控制器勘误的 7 文件口径（3 create + 4 modify，zh/en 为同一 i18n 项两份 locale）：恰好 7 个文件，全部纯插入（0 删除），无 web/ 外文件，无范围蔓延、无遗漏。工作树除报告目录（约定不提交）外干净（`git status --porcelain` 仅 `?? .superpowers/sdd/...`）。commit subject 逐字 = `feat(admin): 计划创建向导（free/flat 价卡）`。

### 2. plan-form-schema.ts — PASS（字节级逐字一致，零偏差）

处方代码块 1+2 拼接后与实现 `diff -u` **exit 0（零差异）**。要点逐项核对：

- 导出全集 14 个：`priceFormSchema`(L21)、`PriceFormValue`(L31，满足评论 2 跨工单契约)、`rateCardSchema`(L33)、`RateCardFormValues`(L64)、`phaseSchema`(L66)、`PhaseFormValues`(L85)、`phasesSchema`(L87)、`planWizardSchema`(L103)、`PlanWizardValues`(L119)、`defaultRateCard`(L121)、`defaultPhase`(L132)、`EMPTY_PLAN`(L136)、`toPlanPhases`(L165)、`toCreatePlanRequest`(L174)。
- 四条正则逐字一致（L8–12：RESOURCE_KEY / ISO8601_DURATION / AMOUNT / CURRENCY）。
- superRefine path 定位：phasesSchema `path: [index, 'duration']`（L96）✓；rateCardSchema `path: ['type']`（L58）✓。
- zod message 全部为 i18n key（`config.plans.wizard.errors.*`，8 处）✓。
- null→undefined 映射：`billingCadence: card.billingCadence ?? undefined`（L159）✓；`duration: phase.duration.trim() || undefined`（L169）✓。
- trim 语义：key/name/currency/description 均先 trim（L156–157, 167–168, 178–181）✓。
- wire 契约：`toRateCardInput` 不发送 `type`/`featureId`；金额为字符串 ✓。

### 3. price-editor.tsx — PASS（恰 1 处偏差 = 报告 Deviation #5）

`diff -u` 处方块 3 vs 实现：唯一 hunk 为 useWatch name（见第 9 项裁定）。其余逐字一致：

- `PriceEditorProps` 契约 `control/phaseIndex/cardIndex/currency`（L19–24）✓。
- `useWatch` 取 kind 并保留处方尾部 `as PriceFormValue['kind']`（L49–52）✓。
- kind 切换重置：free→`{ kind: 'free' }`，flat→`{ kind: 'flat', amount: '' }`（L64–70）✓。
- `FieldError`：`t(message, { defaultValue: message })` 翻译行为 + 无 message 返回 null（L27–35）✓。
- `kind === 'flat'` 才渲染 amount 字段（L86）✓；处方中两处异常缩进的 `<FieldError>` 行（处方 82/215 行风格）原样保留。

### 4. plan-form-wizard.tsx — PASS（偏差全部 = 报告 Deviations #1/#2/#3/#4/#6/#7，无未报告差异）

`diff -u` 处方块 4 vs 实现的全部 hunks 逐一对应报告的机械修正（裁定见第 9 项），其余 500+ 行逐字一致。处方要求的结构点：

- `STEPS = ['basics','phases','rateCards']`（L51）✓。
- `zodResolver(planWizardSchema)` + `defaultValues: EMPTY_PLAN` + `mode: 'onChange'`（L68–72）✓。
- open→`form.reset(EMPTY_PLAN)` + 回 `basics`（L74–80；`setStep` 行前为 D7 的 scoped eslint-disable 注释）✓。
- 每步 trigger 字段集：basics `['name','key','currency','billingCadence']`（L89）；phases `'phases'`（L93）✓。
- `PhaseRateCardsSection` 抽组件、hooks 不进循环（L343–361：useFieldArray/useWatch 在子组件内）✓。
- 仅最后阶段渲染 durationHint（L267–271）✓。
- 删除禁用：阶段 `disabled={phaseFields.length === 1}`（L280）；价目卡 `disabled={fields.length === 1}`（L477）✓。
- 提交按钮 `disabled={createPlan.isPending}`（L330）✓。
- `RATE_CARD_TYPES = ['flat_fee']`（L55）✓。

### 5. hooks.ts — PASS（逐字一致）

`git diff a319e1b38..6457afe9f -- web/src/api/hooks.ts`：唯一 hunk 为 +13 行的 `useCreatePlan`，插入位置在 `usePlan`（L196–202）之后、既有 `usePublishPlan` 之前（L204–215）✓。mutationFn 签名 `(input: Parameters<typeof api.plans.create>[0]) => api.plans.create(input)` 与三处 invalidation（`nsPrefix('plans')` / `nsPrefix('plans-page')` / `queryKeys.plan(plan.id)`）逐字对齐处方块 5 ✓。

### 6. index.tsx — PASS（逐字一致）

git hunks 与处方步骤 3 完全对应：4 个精确 import（`Plus` L5 / `useState` L1 既有 / `PlanFormWizard` L22 / `Button` L9）、`const [createOpen, setCreateOpen] = useState(false)`（L31）、PageHeader `actions=` 新建按钮（Plus 图标 + `t('config.plans.wizard.createTitle')`，L107–112）、JSX 末尾挂 `<PlanFormWizard open={createOpen} onOpenChange={setCreateOpen} />`（L161，fragment 内最后元素）✓。无其他改动。

### 7. i18n — PASS（程序化验证）

- **追加块逐字**：zh 追加块（git diff 提取）与处方 zh 块做基准缩进 −4 归一后 `diff -u` **exit 0**；en 同样 **exit 0**。缩进差 4 空格源于处方块以 `plans:` 深层嵌套书写、实际插入层级为 `plans` 的直接子键——结构位置正确：两份 locale 均插在 `config.plans` 对象内、`toast` 兄弟键之后（zh L162–205 / en L165–208）。
- **键集一致（程序化）**：括号配对提取 `wizard` 子树后做键提取对照 — zh 35 键 = en 35 键，集合完全相等（`KEY SETS IDENTICAL: True`，zh-only/en-only 均空）；结构骨架（缩进/键名/是否对象，37 个条目）逐项相同（`STRUCTURE IDENTICAL: True`）。
- 文案：zh 中文 / en 英文，与处方逐字一致（含 `errors` 全部 9 键、`steps`/`cadence`/`rateCardType`（含 `usage_based` 标签，处方明列）/`fields`/`toast.created`）。

### 8. 验收标准映射（Issue 正文） — PASS（代码路径可静态证实）

- 「向导可完整走通并创建含固定费价卡的计划」：`onSubmit` → `createPlan.mutate(toCreatePlanRequest(values), { onSuccess: toast.success(t('config.plans.wizard.toast.created')); onOpenChange(false), onError: handleServerError })`（plan-form-wizard.tsx:99–110）→ `api.plans.create` POST（SDK sdk/plans.d.ts:33）→ 三处 invalidation 刷新列表/详情（hooks.ts:209–213）。flat 价卡路径：`toPriceInput` case 'flat' → `{ type:'flat', amount }`（plan-form-schema.ts:149–150），金额字符串直传。
- 「数量/期限校验错误定位到具体行」：
  - 期限：`phasesSchema.superRefine` `path: [index,'duration']`（plan-form-schema.ts:94–98）→ RHF 错误落在 `phases.${index}.duration` → 该行输入下方 `FieldError`（plan-form-wizard.tsx:257–274）。
  - 价目卡数量：`rateCards.min(1, '...rateCardsRequired')`（plan-form-schema.ts:80–82）→ `errors.phases[phaseIndex].rateCards.message` → 阶段卡片下方 `FieldError`（plan-form-wizard.tsx:486–492）。
  - 阶段数量：`phasesSchema.min(1)` 顶层 → 行列表后 `FieldError`（plan-form-wizard.tsx:287–289）；flat 金额非法 → 金额输入下方行内 `FieldError`（price-editor.tsx:86–101）。
  - 浏览器手测属控制器后续 walkthrough 阶段（实施计划明列 Task 1 之后），与本项「静态证实」口径一致。

### 9. 偏差裁定（7 条，逐条） — PASS（全部落在处方允许范围）

| # | 偏差 | 独立核实证据 | 裁定 |
|---|---|---|---|
| D1 | `mutate({ body: X })` → `mutate(X)` | SDK 实测：`create(request: CreatePlanRequest, options?)`（sdk/plans.d.ts:33）；`CreatePlanRequest = AcceptDateStrings<CreatePlanRequestInput>` 扁平无 body 包装，而 `UpdatePlanRequest = AcceptDateStrings<{ planId; body }>`（models/operations/plans.d.ts:24–29）——不对称与报告所述一致 | 编译强制，语义等价，范围内 |
| D2 | 移除 7 处 `as FieldPath<PlanWizardValues>` cast | RHF 7.72.1（package.json 实测）`dist/types/path/eager.d.ts` 含 9 处 `` `phases.${number}` `` 模板路径——内联表达式原生精确推断，cast 反而使 `TName` 拓宽为全路径联合致 `field.value` 类型不兼容 | 编译强制，行为不变，范围内 |
| D3 | useFieldArray name cast 移除 | 同上：`FieldArrayPath` 约束下全路径联合不满足，`` `phases.${n}.rateCards` `` 原生合法 | 编译强制，范围内 |
| D4 | useWatch `as never` 移除（wizard） | `as never` 命中 `name?: undefined` 重载返回 `DeepPartialSkipArrayKey`，内联模板字面量保精确类型 | 编译强制，范围内 |
| D5 | useWatch name 内联（price-editor） | 模板字面量经 const 变量中转即拓宽为 `string`（TS 规则），内联是保类型的唯一写法；处方尾部 `as PriceFormValue['kind']` 保留 | 编译强制，范围内 |
| D6 | `type FieldPath` import 删除 | cast 移除后未用；`@typescript-eslint/no-unused-vars: error`（eslint.config.js:34）强制；处方步骤 1 明文授权「清理上文标注的未用 import」 | 处方明文授权 + lint 强制，范围内 |
| D7 | `set-state-in-effect` scoped disable 注释 | 规则存在于 eslint-plugin-react-hooks 7.1.1（cjs bundle + README 实测），recommended 预设经 eslint.config.js:27 接线为 error；修正为单行 scoped 注释，被 suppress 的 `setStep('basics')` 行为与处方完全一致；替代方案需删除处方 effect/改挂载方式（更大偏差） | lint 强制的最小机械修正，已记录，范围内（Minor 备注见下） |

报告所称「(a) 前任遗留偏差 0 条」与字节级 diff 结果吻合（plan-form-schema.ts / hooks.ts / index.tsx / 两 locale 零偏差；其余两文件的差异全部落在上表 7 条）。

**处方代码引用的既有键存在性（grep 实测）**：`plan.priceType.free/flat`（zh L364–366 / en L372–374）、`config.plans.detail.cadence`（zh L132 / en L135）、`config.plans.detail.rateCardName`（zh L128 / en L131）、`config.plans.detail.noDuration`（zh L127 / en L130）、`common.back/next/submitting`（zh/en L10–12）、`config.plans.fields.name/key/currency/billingCadence/description`（zh L111–119 / en L114–122）——全部真实存在于两份 locale。✓

### 10. 非目标遵守 — PASS

diff 全量 grep `usage_based|graduated|volume|unit|put|api.plans.update|plans.publish` 的 `+` 行命中均为处方自身内容（注释、`z.enum(['flat_fee','usage_based'])` 契约、i18n 的 `usage_based` 标签）——UI 选项仅 `RATE_CARD_TYPES=['flat_fee']`，price 判别联合仅 free/flat 分支，无 PUT/`api.plans.update` 调用。7 文件全部在 `web/` 下，无后端/API/spec 改动。e2e 无行为改动：未触碰任何 e2e 文件，既有 2 条冒烟原样通过（见下）。

## 复跑与日志抽查

- **复跑 `cd web && pnpm lint`：exit 0**（`$ eslint .` 无任何发现）。
- `/tmp/issue6-lint.log`（11 字节）：仅 `$ eslint .`，零输出 = 干净 ✓。
- `/tmp/issue6-e2e.log`：`Running 2 tests using 2 workers` → `✓ sign-in smoke: OIDC round-trip lands on the dashboard`、`✓ customers smoke: route reachable and renders list data` → **`2 passed (6.3s)`**，与报告的通过条数和测试名逐字一致 ✓。
- `/tmp/issue6-build.log`（147 行）：vite v8.0.8 production build，尾部 `✓ built in 297ms`，tsc -b 无错误输出 ✓。
- **时间戳一致性**：lint 14:22 → e2e 14:23 → build 14:24 → commit `6457afe9f` 14:24:56 → 报告文件 14:26 —— 三份日志均为提交前最终态，时序自洽 ✓。旁证工件（`issue6-build2.log` 8/27 22:09 前次失败 build、`issue6-gh-error.log`、`issue6-install.log`）与报告的环境叙事吻合。
- （注：报告验证表按 build/lint/e2e 顺序罗列，实际执行时序为 lint→e2e→build——表格为呈现顺序而非时间线声明，内容与最终态一致，不影响有效性。）

## 发现清单

- **Critical：0。Important：0。**
- Minor-1：D7 为 eslint-disable **抑制**而非结构性修正（`react-hooks/set-state-in-effect`）。属 lint 强制的最小机械修正、有仓库先例且已按要求记录，不构成处方违背；建议代码质量审查阶段关注该 effect 模式在 #9 编辑回填扩展时的可维护性。
- Minor-2：实现者报告的验证证据表命令顺序（build/lint/e2e）与日志 mtime 反映的实际执行顺序（lint/e2e/build）不一致；纯文档呈现问题，不影响三连全绿的真实性。

## 结论

实现与 Issue #6 处方逐字对齐：7 文件中 5 个（plan-form-schema.ts、hooks.ts、index.tsx、zh-CN.ts、en.ts）字节级零偏差；price-editor.tsx 与 plan-form-wizard.tsx 的全部差异为 7 条已记录的编译/lint 强制机械修正，逐条核实均落在处方允许范围（未用 import 清理 / lint·构建强制机械修正）内，无未报告偏差、无范围蔓延、无非目标越界。验收标准的代码路径可静态证实；lint 复跑 exit 0；三份日志真实且绿。

SPEC REVIEW: PASS (0C/0I/2M)
