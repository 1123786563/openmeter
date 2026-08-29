# Task 1 report — issue #24 T1 API 层

状态：DONE_WITH_CONCERNS ｜ 提交：`4048d1f64`（分支 codex/admin-config-24，基于 65fb00977）

## 做了什么

严格按 brief 逐字实施，仅触碰两个文件（+41 行，0 删除）：

1. `web/src/api/legacy.ts` — 在文件末尾 `createPortalToken` 之后追加 `listPortalTokens(limit = 100)`（GET `/v1/portal/tokens?limit=${limit}`，返回裸数组 `PortalToken[]`）与 `invalidatePortalTokens(body)`（POST `/v1/portal/tokens/invalidate`，id/subject 二选一，`Promise<void>`），代码与 brief 给出的片段逐字一致。
2. `web/src/api/hooks.ts` — 在 `useCreatePortalToken` 之后追加 `usePortalTokens(limit = 100)`（queryKey 用既有的 `queryKeys.portalTokens({ limit })`）与 `useInvalidatePortalToken()`（成功后按 `nsPrefix('portal-tokens')` 失效缓存）；`listPortalTokens`、`invalidatePortalTokens` 按字母序插入文件顶部既有 legacy 导入。

未改动 `query-keys.ts`、段内既有 hook 及其他任何文件；diff 已逐行核对与 brief 一致。

## 验证（真实运行记录）

命令：`cd /Users/wuyongjun/trea/openmeter-issue-24/web && pnpm build && pnpm lint`

- **首次运行**：`pnpm build` exit 2，`pnpm lint` exit 0。build 失败根因是环境问题而非本次改动：`@openmeter/client`（`file:../api/spec/packages/aip-client-javascript`）在此 worktree 从未构建，包内无 `dist/`，导致全局 TS2307 及连锁 TS18046/TS7006。**已证实与本次改动无关**：`git stash` 后在干净树上运行 `tsc -b` 产生完全相同的报错（仅行号因我新增 2 行导入而平移），`TSC_EXIT_ON_CLEAN=2`。
- **环境修复（未改任何仓库文件）**：在 `api/spec/packages/aip-client-javascript` 执行 `pnpm install && pnpm build`（tsc 成功产出 dist/）；随后 pnpm 拒绝刷新 web 的陈旧 `file:` 副本（`pnpm install`/`--force` 均报 "Already up to date"），故 `rm -rf node_modules && pnpm install` 全量重链（exit 0，362 包全部自 store 复用，无网络下载；期间一次 install 因我手工删除残留半态崩溃，重跑一次即干净完成，`.bin` 下 tsc/vite/eslint 齐全）。`pnpm-lock.yaml` 全程无改动。
- **最终运行**：`pnpm build` **exit 0**（tsc -b 通过，vite built in 310ms，3453 modules）；`pnpm lint` **exit 0**（eslint 无输出通过）。
- `git status`：`tsr generate`（build 第一步）会把 `src/routeTree.gen.ts` 重排导入顺序（369 增/369 删，纯排序无语义变化），已用 `git checkout --` 还原；最终 status 仅含两个目标文件，无 routeTree.gen.ts 修改，符合 brief 要求。

## 提交

```
4048d1f64 feat(admin): 门户令牌列表与失效 API 层 (issue #24 T1)
 2 files changed, 41 insertions(+)
```

## 疑虑

1. **worktree 环境前置**：`pnpm build`（web）依赖先构建本地 SDK（`cd api/spec/packages/aip-client-javascript && pnpm install && pnpm build`）。本次已构建并重链 node_modules；后续 T2+ 任务直接受益，但若任何人重建 node_modules，需先构建 SDK，否则 build 必失败。
2. **tsr generate 副作用**：本环境的 `tsr generate` 会重排 `routeTree.gen.ts` 导入顺序，每次 `pnpm build` 后都会弄脏该文件；提交前须 `git checkout -- src/routeTree.gen.ts`（本任务已处理）。疑似本地 tanstack/router CLI 版本与生成该文件时不同，未深究。
3. 本次为验证做的 SDK 构建 + node_modules 全量重链属于 worktree 本地环境变更（未进版本库、未动 lockfile），控制器如需完全干净的环境复现请知悉。
