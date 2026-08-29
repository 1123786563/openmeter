# Issue #30 — 通知渠道：编辑/禁用时保留外部设置的 metadata

- Issue: https://github.com/1123786563/openmeter/issues/30（[admin-config-followup]，ready-for-agent，无阻塞）
- 分支: codex/admin-config-30（worktree /Users/wuyongjun/trea/openmeter-issue-30，base f60cb90b0）

## 范围

修复独立抽查（2026-08-29，#13 评论）发现的低危数据丢失风险：PUT `/api/v1/notification/channels/{id}` 为全量替换语义，后端对省略字段清空（`mapping.go` `AsMetadata(nil)` → nil）。当前 UI 两条 PUT 路径均省略 `metadata`，导致经 API 直接创建且携带 metadata 的渠道在「编辑保存」或「禁用切换」时 metadata 被静默清空。本 UI 自身不设置 metadata，管理端须保留外部设置的值。

修复处方（Issue #30 原文，与 signingSecret 回填同模式）：

1. `web/src/features/config/notification/channels.tsx` `toChannelBody`：追加 `...(channel.metadata && Object.keys(channel.metadata).length > 0 ? { metadata: channel.metadata } : {})`——禁用切换路径（经 toChannelBody）随之修复；
2. `web/src/features/config/notification/channel-form-dialog.tsx` 编辑提交 body：从 `channel` prop（编辑模式原有实体）回填同一表达式；创建模式不设置该字段；
3. `web/src/api/legacy.ts`：更新请求类型如缺失 `metadata?` 则补充——**勘察结论：`NotificationChannelCreateRequest` 已含 `metadata?: Record<string, string>`（PUT 复用该类型），无需改动**；
4. 无新增 i18n 键；无 UI 展示变化（纯保留语义）。

## 非目标

- 不在 UI 展示/编辑 metadata 内容（纯保留语义）；
- 不改创建路径行为（创建不发送 metadata）；
- 不改 legacy.ts 类型与任何后端代码；
- 不动 notification 域其他文件（rules/events 属 #15/#16 轨道）。

## 任务拆分

- T1（单任务）：两处 PUT body 构造的 metadata 回填 + 注释同步（toChannelBody docstring、dialog 的全量替换注释）。

## 测试与验收命令

- `pnpm build`（tsc 类型门禁：metadata 回填表达式必须与 `NotificationChannelCreateRequest['metadata']` 类型吻合）
- `pnpm lint`（全仓）+ `prettier --check` 两个改动文件
- locale 奇偶校验：en/zh 键数不变（本轨无 i18n 改动，main 基线 824=824）
- `pnpm test:e2e`（:8888 live server 在线，签名与基线对比，不得回归）
- 验收标准（Issue 原文）：编辑保存带 metadata 的渠道后 metadata 原样保留（PUT body 含回填 metadata）；禁用切换后同样保留；创建路径不发送 metadata；build+lint 通过；locale 键数不变。

## 全局约束

- 遵循仓库 AGENTS.md 与总纲 `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Global Constraints；
- PUT 全量替换的既有回填模式（signingSecret）保持一致；
- 无 i18n 新键；禁改后端与 SDK 生成物。
