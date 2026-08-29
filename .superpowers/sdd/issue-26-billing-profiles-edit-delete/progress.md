# SDD ledger — plan: docs/superpowers/plans/issue-26-billing-profiles-edit-delete.md

- 轨道：issue #26 账单档案编辑/删除；worktree /Users/wuyongjun/trea/openmeter-issue-26；
  分支 codex/admin-config-26；base f60cb90b0。
- Issue: https://github.com/1123786563/openmeter/issues/26

## Preflight 冲突扫描（dispatch 前）

| 交叠对 | 检查结论 |
|---|---|
| T1 hooks ↔ T2 编辑/删除 | T1 产出 useUpdateBillingProfile/useDeleteBillingProfile，T2 消费；命名已固定 |
| T2 form dialog 编辑态 ↔ 创建态 | 同一 dialog 双模式：无 profile prop=创建（行为与 #25 完全一致），有=编辑；提交映射分叉是本轨核心风险，审查重点 |
| T2 index.tsx 操作列 ↔ 既有 5 列 | 表头加第 6 列「操作」，renderProfile 同步 |
| T2 i18n ↔ 既有 config.billingProfiles 子树 | 只追加 edit/appsImmutable/delete/deleteConfirm/toast.updated/deleted/list.actions |
| 本轨 ↔ 并行轨 #22/#24/#28 | 不同 worktree；hooks 追加段与 locale 子树互不相交 |
| 计划文本自洽 | update body 无 apps + workflow 回填 profile.workflow（禁止 workflow:{} 重置）已在契约事实强调；与 SDK UpsertBillingProfileRequestInput 一致 |

Ruling: 编辑提交回填 profile.workflow 而非空对象——PUT 全量替换语义下 workflow:{}
会把服务端 collection/invoicing/payment/tax 设置重置为缺省；create 的 workflow:{}
注释仅对创建（落默认值）成立。代价：若用户预期「编辑即重置 workflow」则行为不同
——无此预期，正确性优先。

## SDD 模式

subagent 工具本轮可用 → 完整 SDD（同 #24 ledger 记录）。

## BASE

- T1 BASE: f60cb90b0

## T1 API 层

- Implementer: fresh subagent（5d408ac9）DONE_WITH_CONCERNS @ 433993e4a
  （+useUpdateBillingProfile/+useDeleteBillingProfile；build/lint exit 0）。
- Concerns：SDK dist 环境事项（同他轨）。
- 审查中：review-48eb434e3..433993e4a.diff。

- 审查（64484faf）：SPEC PASS 无发现；QUALITY PASS 无发现（含 SDK 源码级实证：
  UpsertBillingProfileRequestInput 无 apps 字段、DeleteBillingProfileRequest={id}）。
- T1 complete（433993e4a，clean，零修复轮）。

## T2 编辑模式 + 操作列 + 删除 + i18n

- Implementer: fresh subagent（53c11dc6）DONE @ 871606d13（dialog+index+两
  locale；build/lint 双 0；billingProfiles 39=39 纯插入）。
- 自查备忘：prettier 3.8.3 CJK 宽度与历史不同——轨道独占 tsx 已按其规范化
  （3 处既有行 reflow），locale 严格纯插入。
- 审查中：review-433993e4a..871606d13.diff。

- 审查（7a54ff02）：SPEC PASS 无发现（验收项 1/2 双达成：apps 禁用+不可变提示、
  workflow 回填非空对象、删除错误原文透出）；QUALITY PASS，3 条 Minor 观察。
- Ruling（修复轮 1）：Minor#2「update 缺省 labels 或清空服务端标签」经控制器
  服务端实证升级为必修——convert.gen.go 缺省 Labels→nil Metadata +
  adapter/profile.go 无条件 SetMetadata（workflow 有回填保护、labels 无）。
  修复：update body 回显 profile.labels（与 workflow 回显同模式）。原实现者
  不可续轮（运行时限制）→ 派新 implementer（99767a1f）。
- 环境裁定：send_message 对已结束 subagent 不可用（本运行时实证）——修复轮
  一律派全新实现者，SDD 语义不变。

- 修复轮 1 复审（4c1f100d）：FIX PASS 无发现（回显表达式/位置/类型兼容
  （SDK types.d.ts:6981 labels?: Labels）/update 语义零破坏/创建分支零改动/
  恰一文件无夹带，全部逐项实证）。
- 轨道实现完成：433993e4a(T1) + 871606d13(T2) + 0ddf6a158(fix1)。

## 终态门禁

- e2e（串行，/tmp/issue-round/e2e-26.log）：sign-in smoke PASS；customers smoke
  FAIL——与原始基线同签名（e2e-base.log）→ 既有环境问题非本轨回归。
- locale parity：billingProfiles 39=39 纯插入。

- 全分支终审（ae4244e6）：FINAL RELEASE_READY 无新发现（含全文件键集
  en/zh 各 833 双向零差集复核；labels 回显对服务端两种语义均幂等正确）。
- 轨道终态：T1 433993e4a + T2 871606d13 + fix1 0ddf6a158，门禁全绿。
