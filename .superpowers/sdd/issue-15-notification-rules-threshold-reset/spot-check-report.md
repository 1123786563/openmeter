# Issue #15 独立抽查报告（attended 轮 2026-08-29T11:27Z 锁）

- 抽查性质：控制器执行的 attended 抽查（三派生通道被无人值守允许列表拒绝，非全新
  上下文独立审查——欠账部分保留）。
- 工作树 /Users/wuyongjun/trea/openmeter-issue-15 @ 51a2b10b1（=台账 tip），受控
  文件改动=0；merge-base(main)=ec85f6871=main tip。

## 门禁证据（本轮新鲜重跑）

| 门禁 | 命令 | 结果 |
|---|---|---|
| build | `pnpm --silent build`（web/） | exit 0 |
| routeTree | `git diff --exit-code -- web/src/routeTree.gen.ts` | exit 0（零 diff） |
| eslint（改动面 6 文件） | `pnpm exec eslint <files>` | 0 errors / 1 warning（rule-form-dialog.tsx:172 `form.watch` react-compiler 信息性=Ruling-Q1 接受类） |
| locale 奇偶 | node 真实求值 | en=982 zh=982 onlyEn=0 onlyZh=0（与台账 982=982 一致） |

## Ruling 第一手核实

- Ruling-PD1 ✓：rule-form-dialog.tsx:462 `(form.formState.errors as
  FieldErrors<ThresholdFormValues>).thresholds?.[index]?.value` —— 判别联合下
  FieldErrors 按分支类型自准收窄 + 可选链，处方缺陷修复落实。
- Ruling-PD2 ✓：第 3-8 行 `type Control, type FieldErrors` 均为 type-only import。
- Ruling-Q1 ✓：useFieldArray 在文件内 2 处引用（import+使用）；eslint 0 errors，
  仅同类信息性 warning 1 条（172:23），与接受记录一致。
- Ruling-1（本轨）✓：rule-form-dialog.tsx:150 `useFeatures({ page: 1, pageSize: 100 })`
  + 139 行 pageSize:100；未造零参 useFeatures。
- Ruling-2 ✓：hooks.ts:949 `queryClient.invalidateQueries({ queryKey:
  nsPrefix('notification.events') })` —— #16 键前缀契约（本轮 #16 分支已落地同域
  键 `ns('notification.events', params)` 并验证续约成立）。

## 规格符合性

- 四类型表单（含阈值/重置型）、testNotificationRule 触发链、i18n 982 键族与台账
  一致；query-keys/features/channels 零触碰（diff --stat 实测 7 文件全在通知域+locale）。

## 处方步骤 6 人工走查（欠账）状态

本轮尝试：现存 :5173 vite 服务（pid 47701）经 lsof cwd 核实服务于主工作树
（main tip，无 #15 表单）；自起 dev server 被会话策略拒绝（无人值守不允许后台
进程，run_in_background 探测被拒记录在轮次台账）。**走查欠账保留**，待具备后台
进程能力的 attended 轮执行。

## 结论

**SPOT-CHECK PASS（代码级）** —— 门禁全绿、五项 Ruling 全部第一手证实、无新发现。
遗留（非阻断）：浏览器真实走查（上方）；features 下拉 size=100 上限（与 #2/#14 同限）。
