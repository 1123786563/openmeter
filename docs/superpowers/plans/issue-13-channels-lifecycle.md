# Issue #13 — 通知渠道：编辑/禁用/删除

- **Issue**: https://github.com/1123786563/openmeter/issues/13
- **分支**: `codex/admin-config-13`（worktree `/Users/wuyongjun/trea/openmeter-issue-13`，base `5a4666ec7` = main）
- **处方来源**: Issue comment 1（821 行精确到文件的代码处方）；总计划 Task 7（渠道部分收尾）
- **前置**: #12 已合入（channels legacy 层/列表/创建 dialog/query key `notification.channels` 在位，已核对）

## 范围

1. `web/src/api/legacy.ts` — notification channels 段末追加：
   - `updateNotificationChannel(channelId, body)` → `PUT /v1/notification/channels/{id}`（全量替换 body，复用 `NotificationChannelCreateRequest`）
   - `deleteNotificationChannel(channelId)` → `DELETE /v1/notification/channels/{id}`（204）
2. `web/src/api/hooks.ts` — channels 段追加 `useUpdateChannel()` / `useDeleteChannel()`，onSuccess 失效 `nsPrefix('notification.channels')`
3. `web/src/features/config/notification/channel-form-dialog.tsx` — 创建/编辑双模式：
   - 新 prop `channel?: NotificationChannel`（传入即编辑模式）
   - `useEffect` open 时回填（含 `signingSecret ?? ''`——PUT 是全量替换，省略会被服务端清空，必须回填原样回传）
   - 标题/描述 create|edit 双分支；提交按 isCreate 分流 create/update mutation
4. `web/src/features/config/notification/channels.tsx` — 操作列（编辑/禁用|启用/删除）：
   - 新增 `toChannelBody(channel)` 行级全量 PUT body 重建助手
   - toggle：ConfirmDialog → `updateMutation.mutate({channelId, body:{...toChannelBody, disabled: !was}})`
   - delete：ConfirmDialog（destructive）→ `deleteMutation.mutate(id)`
   - 表头加「操作」列，空态 colSpan 4→5
5. `web/src/i18n/locales/zh-CN.ts` + `en.ts` — `config.notification.channels` 下追加 `actions/enable/disable/delete/toggleConfirm.*/deleteConfirm.*/form.{editTitle,editDescription}/toast.{updated,enabled,disabled,deleted}`

## 非目标

- 不做通知规则/事件页（#14/#15/#16）
- 不改列表过滤/分页逻辑（#12 已定型）
- 不新增「查看 signingSecret 明文」之外的密钥管理功能

## 规格与现状偏差裁定（实施前已发现，必须遵守）

Issue comment 的 channel-form-dialog「完整更新后文件」采用 **FormMessage children 三元式**错误渲染——该模式正是 #12 修复轮（5ec73ca05）修掉的缺陷（FormMessage 有错误时忽略 children，只渲染原始 error.message）。**实施保留现状文件的 i18n-key zod message + FieldError 模式与 `V` 前缀常量，仅叠加编辑模式相关改动**；处方中 FormMessage 用法视为过时基线，其语义（错误文案走 i18n）以现状模式实现。此裁定记入台账。

## 已核对的契约事实

- `api/openapi.yaml` L6218：`PUT /api/v1/notification/channels/{channelId}`（updateNotificationChannel）；同 path 下 `delete`（软删除、不可恢复）。body 复用 `NotificationChannelWebhookCreateRequest`。
- 现状 `NotificationChannel` 接口含 `signingSecret?: string`（GET/列表返回），编辑回填可行。
- `useNotificationChannels` 已带 `includeDisabled: true`——禁用渠道在列表可见，toggle 后状态即时反映。

## 任务拆分（SDD）

- T1 API 层：legacy.ts 追加两个函数——implementer 提交
- T2 hooks：useUpdateChannel/useDeleteChannel——implementer 提交
- T3 表单双模式：channel-form-dialog.tsx——implementer 提交
- T4 列表操作列：channels.tsx + ConfirmDialog ×2——implementer 提交
- T5 i18n：两份 locale 追加——implementer 提交
- 每任务后：规格符合性审查 + 代码质量审查 + 修复轮 ≤5
- 全部任务后：全分支审查 + 验证三连

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

- 既有 2 条冒烟不得回归；locale zh/en key 集合零漂移。
- 浏览器走查（真实后端）：编辑渠道改名称保存 → 列表更新；重开编辑弹窗核对 signingSecret 回填未丢；禁用/启用确认 → 徽章切换 + toast；删除确认 → 行消失。删除受引用错误由后端 toast 原文透出。

## 全局约束

- 文案全部 i18n（zh-CN + en 同步）；zod message 一律 i18n key + FieldError。
- 写操作一律 ConfirmDialog 或明确后果提示（禁用/删除均确认弹窗）。
- PUT 全量替换语义：所有编辑路径必须回填并回传 signingSecret/customHeaders，禁止部分更新。
- 每任务验证三连；e2e 端口与并行轨道共享，验收 e2e 串行执行。
- 无用户批准不 push/merge/close。
