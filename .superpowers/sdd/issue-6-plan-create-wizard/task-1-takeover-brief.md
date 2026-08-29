# Task 1 Takeover Addendum — Issue #6 计划创建向导（free/flat 价卡）

本文件是 `task-1-brief.md` 的接管附录。**先通读 task-1-brief.md**（硬性约束、文件清单、
契约、验证、提交、报告要求全部沿用，此处不重复）。以下是接管特有的增量事实与指令。

## 接管背景（前任 implementer 已死于基础设施故障）

- 前任 implementer（a6cb08b1）已于 21:27–21:30 写完全部 7 个任务文件（**未提交**），
  在首次 `pnpm build` 验证时被 `dsh web` 服务器重启杀死（21:31–21:32）。死因是基础设施，
  不是代码被否决。
- 你不是从零开始：工作树里已有前任写好的未提交改动。你的职责是**接手而非重写**——
  先逐文件对照 Issue #6 评论 1 处方核对前任的实现，再修复、验证、提交。

## 控制器已修复的环境（不要重复处理，也不要报告为自己的干预）

1. SDK `dist/` 已注入（issue-4/5 先例的同基线注入）：`@openmeter/client` 已可在
   `pnpm build` 中正常解析。`pnpm install` 已完成，node_modules 就绪。
2. 已知基线事实：环境修复后 `pnpm build` = **11 个 TS 错误，全部位于两个新文件**
   （完整清单见 `/tmp/issue6-build2.log`，2026-08-27 22:0x 由控制器生成）：
   - `plan-form-wizard.tsx`：L101 `body` 不在 mutation 变量类型中；L238/251/264/414/429
     register 值类型不兼容 InputHTMLAttributes（value 可能为 null/对象）；L356 字段名
     union 超出 expected；L372 `.duration` 属性不存在；L387/446 value 不能赋给 string。
   - `price-editor.tsx`：L49 对 `"flat"|"free"` 的类型断言无充分重叠（TS2352）。
   这些错误模式指向处方代码与实际 SDK/zod/RHF 类型面的偏差——**SDK d.ts 真类型优先于
   评论代码**（issue #2/#4 既定裁定先例），修复时以
   `web/node_modules/@openmeter/client/dist/**/*.d.ts` 实际类型为准，逐条在报告中记录
   偏差与修复方式。

## routeTree.gen.ts 规则（重要）

- `pnpm build` 内含 `tsr generate`，在本工作树会**确定性重排** routeTree.gen.ts 的
  import（+369/−369 纯重排，排序后内容与 HEAD 逐字节一致——控制器已验证）。
- 该文件**不属于本任务的 7 个文件**。每次 build 之后、提交之前必须
  `git checkout -- web/src/routeTree.gen.ts` 恢复；commit 中不得出现该文件。

## 验证与提交（与原 brief 一致，重申关键点）

```bash
cd /Users/wuyongjun/trea/openmeter-issue-6/web
pnpm build      # exit 0；随后恢复 routeTree.gen.ts
pnpm lint       # exit 0
pnpm test:e2e   # exit 0，既有 2 条冒烟通过（端口 9999/4173 被占先查证并清理，记录 PID）
```

- Commit message（逐字）：`feat(admin): 计划创建向导（free/flat 价卡）`
- 单 commit 恰含 7 个任务文件；`git status --porcelain` 必须为空。
- 报告照旧写 `.superpowers/sdd/issue-6-plan-create-wizard/task-1-report.md`，其中
  Deviations 部分必须区分：(a) 前任遗留偏差（你核对时发现并修正的）、(b) 你因类型/
  lint 强制所做的机械修正、(c) 相对处方的任何其他偏差。
