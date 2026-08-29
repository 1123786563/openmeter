# 最终全分支审查 — issue #16 通知事件流与 resend（2026-08-29）

审查域：codex/admin-config-16 = 51a2b10b1（#15 tip，串行链基）+ 本轨 3 提交
（3a3a2f2c6 计划 / eab5a1d67 T1 / 5f9e18274 T2）。本轨自有 diff =
3a3a2f2c6..5f9e18274（7 文件，779+/9-）。#15 提交已在其轨道终审 PASS，
不在本报告域内重复审查。

## 五角度对抗审查

### 1. 验收标准（AC）锚定

- 「事件列表按过滤条件出数」：from/to（datetime-local 草稿→应用时转
  ISO）+ rule/channel 单选 Select→应用时升数组→`listNotificationEvents`
  重复参数→setPage(1)。锚点：applyFilters/NotificationEventsParams/
  satisfies NotificationEventListParams。✓（代码锚定；浏览器真实出数走查
  归 #29 与 attended 欠账）
- 「resend 后投递状态可见变化（或有明确错误 toast）」：POST 202 →
  toast.resent 受理提示 → invalidate nsPrefix('notification.events') →
  1.5s refetch（RESENDING/状态变化可见）；onError handleServerError 透出
  后端拒绝原文（如渠道已删除）。锚点：useResendEvent/ConfirmDialog
  handleConfirm/setTimeout refetch。✓
- Master plan Task 8 与 issue comment 处方步骤 1-6 全部落实（T1/T2 分遍
  归一化比对 IDENTICAL，偏差全数 formatter 强制或 Ruling 授权）。

### 2. 回归面

- 分支自有 diff 仅 7 文件：T1 五文件（append-only + events 子树替换）+
  T2 两文件（新建 + 占位替换）。rules.tsx/rule-form-dialog.tsx/channels/
  addons/plans 域零触碰（grep ZERO 命中）。
- 全分支删除行：zh/en 占位 description ×2 + 路由占位 7 行，共 9 行，
  全部 Ruling-i18n/处方步骤 5 授权。
- routeTree.gen.ts 零 diff；locale 结构奇偶与 base 伪差集 md5 完全一致
  （50b63752…，零新增漂移）；events 子树 zh/en 同为 36 冒号行。

### 3. 证据（全部新鲜命令、落盘）

- final build/lint exit 0（t16-final-*.log）；定向 eslint 0；
  prettier 新增内容合规（Ruling-P1 范围内）。
- e2e：sign-in ✓（1.2s）；customers ✘ 与 base 同机新鲜重跑同签名
  （locator=Acme/建设中 or、5000ms、element(s) not found）→ 环境性
  非回归（#11/#15 跨轨共性）。
- worktree 干净（status 0 行）；三提交信息循 #15 规范。
- routeTree 事故记录：终审轮一次 build 收尾后 routeTree.gen.ts 被生成器
  以 dev 序重写（369+/369- 纯 import 重排，#15 同款先例）→ 恢复提交版后
  新鲜 build 复验 exit 0 + 零 diff（即时与 3s 沉降后双检）+ worktree 全净。
  判定：生成器瞬态产物，非本轨改动（本轨未新增路由路径）。

### 4. 约定

- 处方逐字纪律：T1 四块 + T2 路由归一化 IDENTICAL；events.tsx 差异恰
  五类（import 排序/tailwind class 排序/尾逗号 = formatter 强制；
  PD1 = Ruling 授权修复）。
- 仓库约定：i18n zh/en 同构；命名 `<Type><Value>` 无涉；无 Go/后端/
  TypeSpec 改动（前端轨无 go 门禁，先例一致）；v1 经 legacy apiFetch；
  slog/context 等服务端约定无涉。
- SDD 纪律：分遍 spec/quality 审查 ×2 任务全 PASS、零修复轮（PD1 属
  实施内处方缺陷修复，见 T2 分遍 1 记录）；台账完整。

### 5. 对抗边界（八项核验）

1. 单条刷新失败 → handleServerError + refreshingId 复位（finally）✓
2. resend 至已删渠道 → onError toast 透出后端原文 ✓（AC 替代分支）
3. resendOptions 取 eventOverrides 优先 = 最新投递状态 ✓
4. 空事件/空渠道记录 → empty 行/noChannels 空态文案 ✓
5. 分页边界（page<1、page>totalPages 禁用；应用过滤重置页码）✓
6. DELIVERY_STATE_CLASS 键集 = DeliveryState 联合五值（Record 完整，
   编译期保证）✓
7. payload/attempt body 注入面 → React 转义 + pre break-all ✓
8. 卸载后 refetch/invalidate → TanStack no-op 语义 ✓

## Ruling 全录

- Ruling-base：轨基=#15 tip（类型族硬依赖）；代价：#15 外部化被否则 rebase。
- Ruling-i18n：events 占位子树替换（title 值不变、description 处方文案）。
- Ruling-prefix：ns('notification.events') = #15 useTestRule 预热契约续约。
- Ruling-3：规则/渠道下拉 pageSize:100 上限（处方原文，与 #14/#15 同限）。
- Ruling-P1：prettier 范围=新增内容合规、既有块零重排（base 三文件整体
  --check 本就失败）；代价：整体 --check 仍失败（base 既有债）。
- Ruling-PD1：处方 useState 惰性表达式 Date.now 触发 react-hooks/purity
  error → lazy initializer 等价修复（行为：首渲染求值一次）。

## 剩余风险

- e2e customers 冒烟环境性失败（基线同签名，跨轨共性）。
- 浏览器真实走查（过滤收敛/行展开/重发状态流转/后端拒绝 toast）无自动化
  覆盖——归 #29 全链路验收与 attended 补查欠账（与 #11/#15 同）。
- 规则/渠道下拉 >100 条不全（Ruling-3 代价）。
- 外部化待用户批准；顺序 #11→#15→#16（本轨含 #15 提交，#15 先合则快进
  追加）；#11↔#16 hooks/query-keys/locale 同锚点 append 并集预案。

## RULING: PASS（本地完成，等待外部化批准）
