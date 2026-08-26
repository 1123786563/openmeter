# 管理端底座：shadcn-admin 一次性模板 + v3 生成客户端

`web/` 管理端以 shadcn-admin v2.2.1（MIT，TanStack Router + Vite + Tailwind v4）为一次性模板复制自管、不追上游；API 对接以 api/spec 生成链的 `@openmeter/client`（v3，ky + zod）为主，v1/v2 独有端点（如 entitlements 独立查询）少量手写补齐。中文界面自建 i18next（模板当前版无 i18n 库）。

## Considered Options

- 跟踪上游合并：认证、菜单、页面会被大面积替换，长期合并冲突成本高于收益。
- 全手写 fetch 封装：类型安全差，且与仓库「规范生成客户端」的工具链脱节。

选定一次性模板是因为模板项目本就预期被复制改造；后续需要时手动 cherry-pick 上游组件修复。
