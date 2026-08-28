# Issue #12 实施计划 — 通知渠道：列表与创建（Webhook）

- Issue: https://github.com/1123786563/openmeter/issues/12 `[admin-config 12/29] 通知渠道：列表与创建`
- 总计划: `docs/superpowers/plans/2026-08-27-admin-config-domains.md` Task 7（渠道部分）
- Worktree: `/Users/wuyongjun/trea/openmeter-issue-12`，分支 `codex/admin-config-12`，base `f6e767dc3`（= main）

## 范围

v1 通知渠道（仅 WEBHOOK 类型）的管理页：分页列表（名称/URL/禁用徽章/创建时间）+
新建 dialog（name/url https 校验/customHeaders 键值动态行/signingSecret 可选格式
校验/disabled 开关）。数据层走 `web/src/api/legacy.ts` 手写 v1 层（spec
`api/openapi.yaml` 已核对：`GET /api/v1/notification/channels` 的
`includeDisabled` 默认 false，**管理端必须显式传 true**；POST body=
`NotificationChannelWebhookCreateRequest`，响应 201 渠道实体；字段全 camelCase）。

## 非目标

- 不做编辑/禁用切换/删除（#13 的范围）；本任务只交付创建模式 dialog。
- 不做通知规则与事件流（#14–#16）。
- 不改后端、不改 v3 SDK；不动其他域文件与 routeTree.gen.ts（路由文件 #1 已建，仅替换占位实现内容）。

## 任务拆分（单任务）

1. `web/src/api/legacy.ts` 文件末尾追加 v1 notification channels 段：类型
   `NotificationChannel`/`NotificationChannelPaginatedResponse`/`NotificationChannelListParams`/`NotificationChannelCreateRequest`
   + `listNotificationChannels(params)`/`createNotificationChannel(body)`（经既有 `apiFetch`，query 组装含 includeDeleted/includeDisabled/page/pageSize）。
2. `web/src/api/query-keys.ts`：`refunds` 之后追加 `notificationChannels: (params) => ns('notification.channels', params)`。
3. `web/src/api/hooks.ts` Helpers 段之前追加：`useNotificationChannels(params)`（queryFn 显式 `includeDisabled: true`）、`useCreateChannel()`（onSuccess 用 `nsPrefix('notification.channels')` 失效缓存）；import 并入既有 `@/api/legacy` import。
4. 新建 `web/src/features/config/notification/channel-form-dialog.tsx`：zod schema（name 1-256、url 必须 https、signingSecret 空 or `^(whsec_)?[a-zA-Z0-9+/=]{32,100}$`、customHeaders 行 key≤256/value≤1024、空键行提交时丢弃）；提交映射 type:'WEBHOOK'；toast 成功/`handleServerError` 失败。
5. 新建 `web/src/features/config/notification/channels.tsx`：ServerTable 风格列表（Header/Main/PageHeader 布局、Skeleton 加载、空态、启用/禁用 Badge、formatDateTime、上一页/下一页分页 + 总数）。
6. 替换 `web/src/routes/_authenticated/config/notification/channels/index.tsx` 占位为挂载 `NotificationChannelsPage`（createFileRoute 路径字符串保持既有）。
7. i18n：两份 locale 合并 `config.notification.channels.*`（title/description/create/empty/enabled/disabled/pagination/fields/form{createTitle,createDescription,urlHint,signingSecretHint,customHeadersHint,disabledHint,addHeader,removeHeader,validation{required,url,https,signingSecret,headerKey}}/toast.created），zh 与 en 逐键对齐。

## 测试与验收命令

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

浏览器手测（验收标准）：新建渠道（含 2 个自定义头 + signingSecret 留空/合法值各测一次）→ 列表出现新行且启用徽章正确；URL 输 `http://` 与签名密钥输短串时表单报 https/格式错误；自定义头动态行可增删。

## 全局约束

- 遵循仓库 AGENTS.md 与 web 既有代码风格；不引入 any/as/@ts-ignore。
- 文案全部 i18n，zh-CN 与 en 同步；不得硬编码用户可见文案。
- 端点能力以 `api/openapi.yaml` 为准，禁止臆造字段。
- e2e 既有 2 条冒烟不得回归。
- Commit: `feat(admin): 通知渠道列表与创建（Webhook）`
