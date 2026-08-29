# Task 1 report — issue #22 T1 API 层

状态：DONE_WITH_CONCERNS（实现与验证全部通过；疑虑为环境类，见下）
提交：`76ecccf9c` feat(admin): 应用目录与安装 API hooks (issue #22 T1)（2 files changed, 20 insertions）

## 做了什么

1. `web/src/api/query-keys.ts`：在 `apps: () => ns('apps'),` 之后追加
   `appCatalog: () => ns('app-catalog'),`（逐字按 brief）。
2. `web/src/api/hooks.ts`：Apps (config) 段内、`useUninstallApp` 之后追加两个
   hook，代码逐字来自 brief：
   - `useAppCatalog`：`useQuery` + `queryKeys.appCatalog()` +
     `api.internal.apps.listCatalog(undefined, { signal })`。SDK 签名已核实
     （`api/spec/packages/aip-client-javascript/src/sdk/internal.ts:234`
     `listCatalog(request?: ListAppCatalogRequest, options?: RequestOptions)`），
     第一参数可选，传 `undefined` 通过 tsc，语义与 brief 一致。
   - `useInstallApp`：`useMutation`，`mutationFn` 参数类型
     `Parameters<typeof api.internal.apps.install>[0]`（即
     `InstallAppRequest = AcceptDateStrings<InstallAppRequestBody>`），
     `onSuccess` 失效 `nsPrefix('apps')`（已装列表刷新）。
   - 唯一与 brief 代码块的偏差是格式：`listCatalog(...)` 行按仓库 prettier
     `printWidth: 80` 换行，语义零差异。
3. 未改动段内既有 hook，未触碰其他源文件；`git status` 全程仅上述两个文件。

## 验证（真实运行，均在 /Users/wuyongjun/trea/openmeter-issue-22/web）

- `pnpm build && pnpm lint` → **exit 0**
  （build = `tsr generate && tsc -b && vite build`：3453 modules
  transformed，`✓ built in 313ms`；lint = `eslint .` 无输出通过）。
- `git status` → 仅 `web/src/api/hooks.ts`、`web/src/api/query-keys.ts`；
  **routeTree.gen.ts 零改动**。提交后工作区干净。

## 环境问题与修复过程（重要，供后续任务参考）

首次 `pnpm build` 失败：约 70 个 `TS2307: Cannot find module
'@openmeter/client'` 及连锁 implicit-any。根因是环境而非代码：

- `web/package.json` 依赖 `@openmeter/client` 为
  `file:../api/spec/packages/aip-client-javascript`；pnpm 注入该目录依赖时按
  package.json `files` 字段打包（只含 `dist` + 必含文件），而本 worktree 的
  SDK 源目录从未执行过 `tsc` 构建，`dist/` 不存在，于是注入副本只有
  LICENSE/package.json/README.md 三个文件。
- 修复（均为 gitignored 构建产物 / node_modules 操作，无源文件改动）：
  1. 在 `api/spec/packages/aip-client-javascript` 下 `pnpm install && pnpm build`
     （exit 0，产出 `dist/`）；
  2. 删除 web 下 pnpm 的陈旧注入副本与状态文件
     （`node_modules/.pnpm/@openmeter+client@file+.../`、`.pnpm/lock.yaml`、
     `.modules.yaml`、`.pnpm-workspace-state-v1.json`——仅删
     `.pnpm-workspace-state-v1.json` 即可触发重装），
     `pnpm install --frozen-lockfile`（lockfile 无改动）重新注入含 `dist/`
     的副本；
  3. 重跑 `pnpm build && pnpm lint` → exit 0。

## 疑虑

- 上述 node_modules 环境问题属于本机 worktree 状态；后续任务（T2+）在同一
  worktree 跑 `pnpm build` 前需确认 `@openmeter/client` 注入副本含 `dist/`
  （现已修好；若再遇到同错，按上面步骤重建即可）。CI 不受影响（CI 侧
  pnpm 11.1.2 打包目录依赖时执行 `prepack: tsc`，注入即含 dist；本机
  pnpm 11.7.0 的注入副本不含 dist，差异原因未深究）。
- `listCatalog` 未传分页参数（SDK 默认服务端分页）；若目录列表超一页，
  T2 页面如需全量可换 `listCatalogAll`，属后续任务决策。
