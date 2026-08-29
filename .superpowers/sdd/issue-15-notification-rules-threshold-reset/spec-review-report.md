# Spec 符合性审查报告 — issue #15（分遍 1，新鲜命令）

审查人：控制器分遍审查（Standing DOWNGRADE 模式，见 progress.md SDD 模式节）。
要求源：处方（/tmp/issue-round3/issue-15-comments.md）+ 计划（含 Ruling-1/2）+
任务简报。

## Task 1（787e4032f，BASE 1aae6dacc；提交含一次漏 add locale 的 amend 与
一次 import 修复 + routeTree 恢复提交版）

| # | 要求 | 证据（新鲜命令） | 判定 |
|---|---|---|---|
| 1 | legacy.ts 事件类型族 + testNotificationRule = 处方逐字 | 处方 L41-105 vs diff 新增行 diff：仅 1 行段尾空行差；代码体 IDENTICAL | ✅ |
| 2 | useTestRule = 处方逐字 + import 补行 | 处方 L111-123 vs diff（滤 import 行）：内容行一致（第 13 行为提取窗口栅栏）；import testNotificationRule 为 TS 必需补行 | ✅ |
| 3 | Ruling-1 落实：不改 query-keys、不加零参 useFeatures | numstat 恰 4 文件（legacy/hooks/zh/en），query-keys 零触碰 | ✅ |
| 4 | i18n zh 处方键值精确落地（20 键抽查逐字） | node 求值 20/20 EXACT（初报 1 项 mismatch 复核为 shell 转义伪差异，直查 JSON.stringify 相等） | ✅ |
| 5 | 既有键零删改 | numstat 0 删除；rules 子树 54=54 键；created/createTitle/types/toggleConfirm 等原值保留 | ✅ |
| 6 | zh/en 奇偶 | 全文件 982=982 零差集；rules 子树 54=54；en test/testSent 抽查吻合 | ✅ |
| 7 | Ruling-2 落实：useTestRule 失效 nsPrefix('notification.events') 处方原样保留 | hooks diff 逐字比对覆盖 | ✅ |
| 8 | build/lint | t15-t1-build2/lint2 exit 0（首轮 build TS2552 import 缺失 → 修复 → 0）；routeTree 恢复提交版后零 diff | ✅ |

**Task 1 判定：SPEC PASS，无发现，零修复轮。**

## Task 2（51a2b10b1，BASE 787e4032f；含 prettier/type-import 修正 amend）

要求源：处方步骤 3（rule-form-dialog 完整替换）+ 步骤 4（rules 完整替换）+
步骤 6 手测语义（以代码锚点代人工浏览器走查）。

| # | 要求 | 证据（新鲜命令） | 判定 |
|---|---|---|---|
| 1 | dialog 与处方零语义偏离 | 全量 diff 枚举：偏差恰四类——Ruling-1 三处（useFeatures(params)/featuresData?.data/deps）、处方缺陷修复二处（FieldErrors 分支收窄 TS2339 + consistent-type-imports）、prettier 重排、eslint import 排序；无其他 | ✅ |
| 2 | rules.tsx 同上 | 全量 diff：仅 import 排序 + prettier 换行 | ✅ |
| 3 | AC1 锚点：阈值型（thresholds 1-10 + features 多选） | schema `z.array(thresholdRowSchema).min(1).max(10)`（L90）；useFieldArray 行编辑 + 增删按钮（≤1 禁删/≥10 禁增）；features MultiSelect 选项=功能目录（featureOptions←useFeatures） | ✅ |
| 4 | AC1 锚点：重置型（features 多选） | reset 分支 FeaturesAwareFormValues control 收窄 + MultiSelect | ✅ |
| 5 | spec 语义：空 features 提交省略 | `...(values.features.length ? { features: values.features } : {})` ×2（create/update 提交体 L254/261） | ✅ |
| 6 | AC2 锚点：test 触发 + 结果展示 | useTestRule + 禁用规则按钮 disabled（rules L179）+ ConfirmDialog + toast.testSent({id: event.id}) | ✅ |
| 7 | 编辑放开全类型 + 类型不可换 | type Select `disabled={!isCreate}`（dialog L320）；rules 编辑按钮无条件 | ✅ |
| 8 | 分支渲染正确性 | thresholds/features 字段仅 watchedType 命中分支渲染（L391/487-488） | ✅ |
| 9 | i18n 键覆盖 | 两文件 46/46 t() 键 zh 求值存在（en 由奇偶保证） | ✅ |
| 10 | 门禁 | build3/lint4 exit 0；prettier 两文件 clean；e2e sign-in ✓ + customers ✘ 基线同签名（环境性非回归） | ✅ |

**Task 2 判定：SPEC PASS。**发现并裁定 2 项处方自身缺陷（实施内修复，见 Ruling）：
- Ruling-PD1：处方 L598 `form.formState.errors.thresholds` 在判别联合
  FieldErrors 下 TS2339 编译不过——按处方注记自准的分支收窄手法改为
  `(form.formState.errors as FieldErrors<ThresholdFormValues>).thresholds`
  （语义等价：仅 threshold 分支渲染该行）。若判错代价=阈值行错误提示位置
  不渲染（校验仍由 zod resolver 拦截提交），风险极低。
- Ruling-PD2：处方 import 未按仓库 consistent-type-imports 规则（Control/
  FieldErrors 仅类型使用）——改 type 修饰导入。
- Ruling-Q1：useFieldArray 触发 react-compiler 信息性 warning（Compilation
  Skipped: incompatible library）——RHF 标准数组 API、处方钦定，lint exit 0
  不阻断；替代方案（手写数组状态）偏离处方更大，接受并记录。
