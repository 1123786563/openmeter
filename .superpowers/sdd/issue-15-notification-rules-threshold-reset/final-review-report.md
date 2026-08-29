# 最终全分支审查报告 — issue #15（codex/admin-config-15，1aae6dacc..51a2b10b1）

审查人：控制器终审（Standing DOWNGRADE 模式；attended 独立抽查待补）。
分支 tip 51a2b10b1；+470/-50，6 文件（legacy/hooks/i18n×2 追加 + 两组件完整
替换）。

## 角度 1：规格追溯（Issue AC → 代码锚点）

- AC1「阈值型/重置型规则可创建与编辑」：schema 四类型判别联合
  （thresholds `z.array(...).min(1).max(10)` L90）；阈值行编辑 useFieldArray
  （≤1 禁删/≥10 禁增）；features MultiSelect（选项=功能目录 featureOptions）；
  编辑全类型放开 + type Select `disabled={!isCreate}`（类型不可换）；
  分支渲染按 watchedType 门控（L391/487-488）。
- AC2「test 触发与结果展示」：useTestRule + POST
  /v1/notification/rules/{id}/test（legacy 封装，201→NotificationEvent）；
  禁用规则按钮 disabled；ConfirmDialog 二次确认；toast.testSent({id: event.id})
  展示生成事件 id；事件缓存失效 nsPrefix('notification.events')（#16 落地前
  no-op，Ruling-2 钦定保留）。
- spec 语义：空 features 数组提交省略（L254/261 `...(length ? {features} : {})`）；
  threshold 值 NUMERIC_PATTERN refine（可负、可小数）。
- 处方逐字性：T1 双块 IDENTICAL；T2 全量 diff 偏差恰四类（Ruling-1 适配 /
  处方缺陷修复 PD1·PD2 / prettier / import 排序），零语义偏离。

## 角度 2：回归面

- channels 页、features 目录页零触碰（diff 0 行）。
- query-keys.ts 零触碰（Ruling-1 落实：复用 #2 的 useFeatures 契约）。
- 两组件为完整替换（非叠加），旧 invoice-only 表单死代码零残留
  （+#301/-50 dialog 中删除行全为旧表单/旧 import）。
- i18n 纯追加（+25/+27，删除 0），既有键值零改动。

## 角度 3：证据审计（全部落盘 /tmp/issue-round3/）

- t15-t1-build2/lint2、t15-t2-build3/lint4（最终 tip）双 0；
- t15-t2-e2e.log + t15-final-e2e.log：sign-in ✓（558ms/630ms）；customers ✘
  与 pristine 基线同测试行同错误类同耗时（5.5s/5.6s）→ 环境性非回归；
- 定向 eslint：T1 四文件 0/0；T2 0 error + 1 信息性 warning（Ruling-Q1）；
- prettier：T1 四文件基线对照纯插入；T2 两文件 clean（复写复验）；
- locale 奇偶分支 tip 982=982 零差集；routeTree 提交版零 diff（build 重排
  产物两次恢复）。

## 角度 4：约定

- 反模式全分支新增行 0；API 走 legacy apiFetch（v1 端点）+ v3 SDK
  useFeatures，无手写 fetch；
- 写操作 ConfirmDialog + toast + handleServerError；i18n 双语同构、术语按
  词表（阈值/额度重置/功能范围）；
- 注释承载域语义（deprecated 阈值类型标注、payload oneOf 注记、分支收窄
  原因注释）。

## 角度 5：对抗边界

- threshold 值非法输入：NUMERIC_PATTERN refine + 翻译错误提示（FieldErrors
  分支收窄后行级渲染）；
- 全部阈值行删除：remove 按钮 fields.length ≤1 disabled（保持 min(1)）；
- 超 10 项：append 按钮 ≥10 disabled；
- 空功能目录：featureOptions 空 + MultiSelect 占位 + noFeatures 提示；
- 空渠道：channels min(1) 拦截（既有）；
- 禁用规则误触发 test：按钮 disabled；
- features 目录翻页上限：useFeatures({page:1,pageSize:100})（Ruling-1 契约，
  与 #2 目录页同限）；
- PUT 全量替换语义：edit 回填从 rule 实体（features 回填用 id）重建提交体。

## RULING

**PASS（本地完成，等待外部化批准）。**

## 剩余风险

- e2e customers 冒烟环境性失败（基线同签名，非本轨回归，跨轨共性）；
- useFieldArray 触发 react-compiler 信息性 warning（Ruling-Q1，不阻断）；
- 阈值型/重置型表单无浏览器自动化覆盖（e2e 仅冒烟；处方步骤 6 的人工走查
  由 attended 补查承担——Standing DOWNGRADE 欠账的一部分）；
- features 目录 size=100 上限（与 #2 页同限，超出时选项不全）。
