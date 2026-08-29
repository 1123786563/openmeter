# SDD ledger — plan: docs/superpowers/plans/issue-24-portal-tokens-invalidate.md

- 轨道：issue #24 门户令牌列表+失效；worktree /Users/wuyongjun/trea/openmeter-issue-24；
  分支 codex/admin-config-24；base f60cb90b0。
- Issue: https://github.com/1123786563/openmeter/issues/24

## Preflight 冲突扫描（dispatch 前）

| 交叠对 | 检查结论 |
|---|---|
| T1 legacy/hooks ↔ T2 列表 | T1 产出 listPortalTokens/invalidatePortalTokens/usePortalTokens/useInvalidatePortalToken，T2 消费；命名已固定 |
| T2 ↔ index.tsx 既有发放流 | 仅在 PageHeader 下追加列表区，发放 dialog/一次性明文零触碰 |
| T2 i18n ↔ 既有 config.portalTokens 子树 | 只追加 list/invalidateConfirm/toast.invalidated/status.expired 子键 |
| 本轨 ↔ 并行轨 #22/#26/#28 | 不同 worktree；locale 子树互不相交，合并期按 Issue 号序并集 |
| 计划文本自洽 | v1 列表裸数组（无分页包装）已在契约事实标注，T2 不做分页 |

Ruling: 失效入口按行 id（spec 支持 id|subject 二选一；Issue 仅要求逐令牌失效）。
代价：无 subject 批量入口，功能隐藏，无正确性风险。

## SDD 模式

subagent 工具本轮可用（#22 T1 dispatch 探测成功 14:2x）→ 完整 SDD：
每任务新上下文 implementer + 分遍规格/质量审查 + ≤5 轮修复 + 终审。

## BASE

- T1 BASE: f60cb90b0

## T1 API 层

- Implementer #1（d6df3bdf）异常终止、零改动（git 干净、无报告）——重派。
- Implementer #2（4523ac12）DONE_WITH_CONCERNS @ 4048d1f64
  （+listPortalTokens/+invalidatePortalTokens/+usePortalTokens/+
  useInvalidatePortalToken；build/lint exit 0）。
- Concerns：SDK dist 环境事项 + tsr routeTree 导入重排 churn（提交前还原）——
  均环境非代码，与他轨一致。
- Ruling（环境）：file: 依赖 @openmeter/client 在新 worktree 需先构建 SDK dist
  （api/spec/packages/aip-client-javascript pnpm build）；控制器曾误入其
  node_modules 修复与 implementer 撞车，后续控制器不得触碰运行中轨的 node_modules。
- 审查中：review-65fb00977..4048d1f64.diff。

- 审查（200f185c）：SPEC PASS 无发现；QUALITY PASS 无发现（失效前缀链路、
  apiFetch 204 语义 lib/api.ts:68、legacy 层无 signal 惯例均独立核实）。
- T1 complete（4048d1f64，clean，零修复轮）。

## T2 列表 + 失效确认 + i18n

- Implementer: fresh subagent（13a8c61f）DONE @ d4c63dea0（index.tsx+两 locale；
  build/lint 双 0；portalTokens 子树 32=32 零差、12 新键齐备）。
- 自查备忘：① 失效按钮文案复用 invalidateConfirm.title；② zh unrestricted
  用「不限 meter」。均待审查独立裁定。
- 审查中：review-4048d1f64..d4c63dea0.diff。

- 审查（789b5b6c）：SPEC PASS 无发现；QUALITY PASS 无发现。Minor 备忘×2
  （不要求修复）：① 查询失败显示空态行非错误态（与 apps 参考模式一致的既有
  局限）；② 确认框标题与按钮文案复用同一键（已接受决策）。明文防御独立
  核实：全文零 .token 引用。自查两事项（按钮复用键/「不限 meter」）均裁定接受。
- T2 complete（d4c63dea0，clean，零修复轮）。

## 终态门禁

- e2e（串行，/tmp/issue-round/e2e-24.log）：sign-in smoke PASS；customers smoke
  FAIL——与原始基线 f60cb90b0 逐字同签名（/tmp/issue-round/e2e-base.log：同样
  1 passed + customers 同定位器超时）→ 裁定既有环境问题非本轨回归（轮约定）。
- locale parity：子树 32=32 双端同构（审查+实现者双证）。

- 全分支终审（ad223b40）：FINAL RELEASE_READY 无发现（接缝/验收/回归/一致性/
  发布就绪/非目标六项逐项通过；2 条任务级 Minor 复核确认非阻塞）。
- 轨道终态：T1 4048d1f64 + T2 d4c63dea0，门禁全绿，零未决发现。
