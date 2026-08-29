# Quality Review Brief — Issue #4 Task 1

你是独立代码质量审查员。审查 worktree `/Users/wuyongjun/trea/openmeter-issue-4` 分支 `codex/admin-config-04` 上 `96e1111c0..45f89e50d` 的**代码质量**（spec 符合性由另一位审查员负责，你专注质量维度）。**只读审查 + 允许重跑验证命令；禁止改源文件、禁止派生子代理、禁止 GitHub 写操作。**

## 审查维度

1. **类型安全**：无 any/as 滥用、无 @ts-ignore/eslint-disable 新增（除非逐字对齐仓库既有模式且有先例）；switch 穷尽性（RateCardPriceSummary 5 分支）；空值路径（feature?/description?/duration?/billingCadence?）。
2. **React 质量**：hooks 规则、key 使用（phase.key ?? phaseIndex、card.key）、不必要的重渲染（queryKey 含 params）、受控/非受控状态。
3. **仓库约定**（AGENTS.md + #1–#3 先例）：命名导出、t() 双语、i18n 键无重复/无死键、prettier 格式、最小 diff（无关文件零改动——用 `git diff 96e1111c0..45f89e50d --stat` 全量核对）、与 features/config/features/ 同类实现的模式一致性。
4. **健壮性**：列表页空数据/加载中行为；详情页 404/加载中兜底；分页边界（total undefined 时 ServerTable 行为）；状态筛选与分页交互（切筛选重置页码）；时区/日期渲染（formatDateTime）。
5. **性能/安全**：无新增依赖、无危险 API（eval/dangerouslySetInnerHTML/innerHTML）、无秘密、XSS 面（所有插值经 t() 或 React 文本节点）。
6. **可维护性**：InfoRow/RateCardPriceSummary 命名与 JSDoc 是否表达领域意图；死代码；未使用导出（前瞻 #5/#9 复用可接受但需点名）。
7. **回归面**：routeTree.gen.ts 再生成是否波及既有路由（对比 96e1111c0 版本，差异应仅 $planId 相关）；sidebar/既有页面零改动验证。

## 自行验证（必须做）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-4/web && pnpm lint 2>&1 | tail -3
node --experimental-strip-types 或 tsx 一次性脚本：程序化比对 zh-CN.ts/en.ts 的 config.plans 与 plan 键树 + 全文件重复键扫描 + 新增 t() 引用是否全部可解析（两份 locale 均可）
git -C /Users/wuyongjun/trea/openmeter-issue-4 diff 96e1111c0..45f89e50d --stat
```

## 输出

写入 `.superpowers/sdd/issue-4-plans-list-detail/quality-review-report.md`：各维度发现按 Critical（必须修：会坏功能/坏类型/安全）/Important（应修：明显缺陷或风险）/Minor（可接受，点名记录）分级，每条给文件:行号与证据；自查命令输出摘要；总裁定 PASS（可带 Important 进入 fix round）/FAIL。只写这个文件，不改 progress.md。
