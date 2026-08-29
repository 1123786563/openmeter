# 最终全分支审查报告 — issue #11（codex/admin-config-11，6aaa5b916..26eeb9e5a）

审查人：控制器终审（Standing DOWNGRADE 模式；attended 独立抽查待补）。
分支 tip 26eeb9e5a；+797/-95，7 文件（4 追加共享文件 + 2 新组件 + 1 组件改挂）。

## 角度 1：规格追溯（Issue AC → 代码锚点）

- AC「在计划详情内可直接增删改该计划的附加组件」：
  - 增：PlanAddonsTab create 按钮 → PlanAddonFormDialog（create 分支提交
    CreatePlanAddonRequest：name/addon{id}/fromPlanPhase + 可选 description/
    maxQuantity）；
  - 删：行删除 → ConfirmDialog(destructive) → useDeletePlanAddon + toast；
  - 改：行编辑 → dialog 编辑态（addon 锁定 disabled，提交 UpsertPlanAddonRequest
    无 addon 字段）。
- 处方契约（hooks 四件套/query key/i18n 键集/组件结构）逐字落地（T1/T2 分遍
  diff 证据）；PlanAddon 无价格字段 → tab 不含 PriceForm，改价经行内链接跳
  /config/addons（与 spec 事实一致，总纲 Task 6 笔误以 spec 为准——处方已裁定）。

## 角度 2：回归面

- #10 addons 独立页零触碰（diff -- addons/ = 0 行）。
- plan-detail.tsx 唯一既有组件改动：prettier 两侧对照证明差异恰为 Tabs 包裹 +
  移入块重缩进（+114/-70），overview 信息/阶段视图内容等价（1 处 JSX 换行为
  prettier 深缩进产物）。
- 共享文件 append-only：hooks +65/-0、query-keys +2/-0、zh +44/-0、en +47/-0。

## 角度 3：证据审计（全部落盘 /tmp/issue-round3/）

- t11-t1-build/lint.log、t11-t2-build2/lint2.log（amend 后重验）双 0；
- t11-t2-e2e.log + t11-final-e2e.log：sign-in ✓（631ms/511ms）；customers ✘
  与 pristine 基线（/tmp/issue-round/e2e-base.log）同测试行同错误类同耗时
  （5.6s/5.5s）→ 环境性非回归（历轮共识）；
- t11-t1/t2-eslint-targeted/eslint.log exit 0；prettier 新文件 clean + 基线
  对照全为插入；
- locale 奇偶分支 tip 复核 zh=993 en=993 零差集；routeTree 零 diff。

## 角度 4：约定

- eslint 0 / 新增行反模式 0（eslint-disable·console·debugger·: any·@ts- 全零）；
- i18n 双语同构；术语按词表（附加组件/阶段）；
- 写操作 ConfirmDialog（destructive 标记）+ toast + handleServerError 透出；
- API 全经 @openmeter/client SDK（无手写 fetch/无 legacy）；
- ConfirmDialog props 契约实测吻合（title/desc/confirmText/cancelBtnText/
  destructive/isLoading/handleConfirm + spread open/onOpenChange）。

## 角度 5：对抗边界

- single 实例：maxQuantity 输入 disabled 且提交一律省略（spec 约束）；
- 编辑态换绑：addon Select disabled（PUT 无 addon 字段，UI 不给发）；
- 空计划阶段：fromPlanPhase 默认 phases[0]?.key ?? ''，schema min(1) 拦截提交；
- addon 本体被删：addonNameById 未命中 → 行内退化为 id 码展示（不崩溃）；
- 分页：pageSize 变更重置 page=1；total 加载期 undefined（ServerTable 契约允许）；
- maxQuantity 非法输入：POSITIVE_INT refine 拦截 + FormMessage 翻译键。

## RULING

**PASS（本地完成，等待外部化批准）。**

## 剩余风险

- e2e customers 冒烟环境性失败（基线同签名，非本轨回归；:4173/:8888 shim 环境
  问题，跨轨共性）；
- 后端对 plan-addon 状态机/重复挂载的裁决（如同 addon 同 phase 二次挂载）由
  服务端执行，UI 不预判（错误经 handleServerError 原文透出）；
- addons 首页 size=100 上限（Ruling-1 代价，与 #10 页同限）。
