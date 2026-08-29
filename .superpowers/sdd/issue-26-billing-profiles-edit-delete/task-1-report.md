# Task 1 report — issue #26 T1 API 层

状态：DONE_WITH_CONCERNS（代码与验证均达成；环境有一处需控制器知晓的前置修复，见下）

- Worktree: /Users/wuyongjun/trea/openmeter-issue-26（分支 codex/admin-config-26）
- 提交：`433993e4a` — feat(admin): 账单档案更新与删除 API hooks (issue #26 T1)
- 变更：仅 `web/src/api/hooks.ts`（+25 行）。在 Billing profiles 段 `useCreateBillingProfile` 之后逐字追加 brief 中的 `useUpdateBillingProfile` 与 `useDeleteBillingProfile`；未触碰段内既有 hook，未触碰 query-keys.ts 或其他任何受 git 跟踪的文件。

## 验证结果（真实运行，均在此 worktree 的 web/ 下）

| 命令 | 退出码 |
|---|---|
| `pnpm build`（= `tsr generate && tsc -b && vite build`） | 0 |
| `pnpm lint`（= `eslint .`） | 0 |
| `git status`（提交后） | 干净：无 routeTree.gen.ts 修改，仅本报告所在的未跟踪 `.superpowers/` 目录 |

SDK 侧前置验证：确认 `api/spec/packages/aip-client-javascript/src/sdk/billing.ts` 存在 `updateProfile(request: UpdateBillingProfileRequest)` 与 `deleteProfile(request: DeleteBillingProfileRequest)`，与 hook 透传用法一致。

## 环境前置修复（不影响提交内容，控制器需知晓）

首次 `pnpm build` 失败（exit 2），根因是本 worktree 的 `@openmeter/client`（file: 依赖）从未构建：pnpm store 副本只有 LICENSE/package.json/README.md，无 `dist/`，导致全仓 68 处 `TS2307 Cannot find module '@openmeter/client'` 及连锁 implicit-any 错误——均为既有环境问题，与本次 diff 无关（主仓 `/Users/wuyongjun/trea/openmeter` 的同款安装是有 dist 的，两仓 SDK `src/` 除测试用 `*.assert.ts` 外逐字节一致）。

处置（全部不在 git 跟踪范围内）：

1. `api/spec/packages/aip-client-javascript` 下 `pnpm install`（exit 0）、`pnpm build`（exit 0）→ 生成 `dist/`（该目录被 SDK 自带 `.gitignore` 的 `dist/` 规则忽略，不污染 git status）。
2. web/ 下重跑 `pnpm install` 显示 "Already up to date" 未刷新 file: 副本，故将构建产物 `dist/` 复制进 `web/node_modules/.pnpm/@openmeter+client@file+..+api+spec+packages+aip-client-javascript/node_modules/@openmeter/client/dist`。

此后 `pnpm build` exit 0、`pnpm lint` exit 0。

routeTree.gen.ts 说明：首次失败的 build 运行过 `tsr generate`，一度使 `routeTree.gen.ts` 出现纯 import 重排序的 369/369 行 churn；成功的 build 再次生成后该文件与 HEAD 完全一致，提交前后 `git status` 均无它的修改。

## 疑虑

- 上述 node_modules/SDK dist 修复是对一次性验证环境的处置；若控制器或其他任务在同一 worktree 重新 `pnpm install`，file: 依赖副本可能再次丢失 dist，需重建 SDK 或重新复制。
- `useDeleteBillingProfile` 的 `{ id }` 入参形状来自 brief 逐字要求；`DeleteBillingProfileRequest` 恰为 `{ id: string }` 形状，二者一致，无出入。
- brief 提到 body 类型 `UpsertBillingProfileRequestInput` 由 SDK 提供且不含 apps；T1 未构造 body，仅透传，未受影响。
