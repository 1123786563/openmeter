# Final Whole-Branch Adversarial Review Brief — Issue #4

你是最强模型的终审审查员。对 worktree `/Users/wuyongjun/trea/openmeter-issue-4` 分支 `codex/admin-config-04` 的全分支 `18309b955..HEAD` 做对抗性终审（假设实现有错，努力找出它）。**只读 + 验证命令重跑；禁止改源文件、禁止派生子代理、禁止 GitHub 写操作。**

## 必读输入

1. Issue 正文+评论（规范）：`gh issue view 4 -R 1123786563/openmeter --json body,comments`
2. 实施计划：`docs/superpowers/plans/issue-4-plans-list-detail.md`
3. SDD 台账：`.superpowers/sdd/issue-4-plans-list-detail/progress.md`
4. 已有报告：task-1-report.md / spec-review-report.md / quality-review-report.md / browser-walkthrough-report.md（及任何 fix-round 记录）

## 审查角度（每角度独立执行并给证据）

1. **规范保真**：Issue 验收标准（列表可筛选状态并分页；详情正确渲染所有阶段与价目卡结构）在最终代码里的满足证明；评论处方代码与实现的差异逐一裁定。
2. **SDK 契约再验证**：`api.plans.list/get` wire 形状、Plan/PlanPhase/RateCard/Price 字段名、StringFieldFilterExact——独立从 `api/spec/packages/aip-client-javascript/dist/` 重新推导，不信任何二手结论。
3. **查询语义**：queryKey 构造（params 对象引用稳定性/序列化行为——ns() 实现核对）、usePlan enabled 门、筛选切换的请求-响应一致性、React Query 缓存失效面（本任务只读无 mutation，验证无 stale 陷阱：同一 planId 二次进入是否命中缓存、筛选变更是否必然触发新请求）。
4. **i18n 深检**：AST/程序化比对 zh/en（全文件重复键、死键、新增 t() 引用双向可解析、interpolation 变量一致）；StatusBadge/EnumBadge 的动态键 `plan.status.*`/`plan.priceType.*` 是否全部有键（5 价格类型 × 4 状态）。
5. **回归面**：`git diff 18309b955..HEAD --stat` 全量——除任务文件+routeTree+计划文档+（可能的 fix）外零改动；routeTree 再生成对既有路由节点零扰动（对照 18309b955 版本逐节点比对）；sidebar/命令面板/既有页面行为不变。
6. **跨提交一致性**：每个 commit 的 message 与内容相符；fix-round 提交（若有）是否引入新面。
7. **安全**：无秘密、无 eval/dynamic import、无 XSS 注入面（所有用户数据经 React 文本节点/t()）、零新增依赖（package.json/pnpm-lock 零 diff）。
8. **证据审计**：/tmp/issue4-*.log 日志时间线自洽、walkthrough 的 wire 记录与断言互相印证、台账 Ruling 与各报告结论一致、截图指标合理。
9. **仓库约定**：AGENTS.md web 侧约定（#1–#3 先例）：命名导出、类型导入 style、JSDoc 领域意图、最小 diff、prettier。

## 自行验证（必须）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-4/web
pnpm lint 2>&1 | tail -3          # 期望 0 error
node 一次性 i18n 深检脚本（/tmp）
git -C .. diff 18309b955..HEAD --stat
git -C .. log --format='%h %s' 18309b955..HEAD
```

## 输出

写入 `.superpowers/sdd/issue-4-plans-list-detail/final-review-report.md`：每角度 PASS/FAIL+证据；发现按 Critical/Important/Minor 分级；总裁定 **SAFE TO PRESENT** 或 NOT SAFE（附必改清单）。只写这个文件，不改 progress.md。
