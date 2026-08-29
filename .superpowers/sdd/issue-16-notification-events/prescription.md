## 详细实施计划

> 对应总计划 Task 8（事件页）。前置：#14（规则列表/`useNotificationRules`、渠道 `useNotificationChannels`、`MultiSelect` 已合入）；#15 已定义事件实体类型（`NotificationEvent` / `NotificationEventDeliveryStatus` / `NotificationEventDeliveryAttempt` / `EventDeliveryAttemptResponse` / `NotificationEventDeliveryState` / `NotificationEventPayload` / `NotificationEventType`）——本任务直接复用，不重复定义。
>
> Spec 事实（`api/openapi.yaml`，已核对）：
> - `GET /api/v1/notification/events`：过滤 `from`/`to`（RFC 3339，含端点）、`feature[]`/`subject[]`/`rule[]`/`channel[]`（重复参数数组）、`page`/`pageSize`；响应 `{totalCount,page,pageSize,items}`。
> - `GET /api/v1/notification/events/{eventId}`：单事件（行内刷新用）。
> - `POST /api/v1/notification/events/{eventId}/resend`：body `{channels?: string[]}`（**省略 = 重发到原渠道**），响应 **202 Accepted 无内容**——成功与否只能靠事后投递状态与错误 toast 体现。
> - 事件实体：`deliveryStatus[]` 元素 `{state, reason, updatedAt, channel:{id,type}, nextAttempt?, attempts[]}`；`attempts[]` 元素 `{state, response:{statusCode?, body, durationMs}, timestamp}`；`rule` 为完整规则实体（取 `rule.name` 展示）。

### 文件（精确路径）

- Modify: `web/src/api/legacy.ts`（追加 events 段：list/get/resend）
- Modify: `web/src/api/query-keys.ts`（加 `notificationEvents`）
- Modify: `web/src/api/hooks.ts`（`useNotificationEvents` / `useResendEvent`）
- Create: `web/src/features/config/notification/events.tsx`
- Create or Modify: `web/src/routes/_authenticated/config/notification/events/index.tsx`
- Modify: `web/src/i18n/locales/zh-CN.ts`、`en.ts`

### 接口契约（Consumes/Produces）

**Consumes**：`GET /api/v1/notification/events`、`GET /api/v1/notification/events/{eventId}`、`POST /api/v1/notification/events/{eventId}/resend`。

**Produces**：

- `legacy.ts` 新增：`NotificationEventListParams`（`from?/to?/feature[]?/subject[]?/rule[]?/channel[]?/page?/pageSize?`）、`NotificationEventPaginatedResponse`、`NotificationEventResendRequest`（`{channels?: string[]}`）；函数 `listNotificationEvents(params)` / `getNotificationEvent(eventId)` / `resendNotificationEvent(eventId, body?)`
- hooks：`useNotificationEvents(params)`（params：`from?/to?/rule?/channel?/page/pageSize`，rule/channel 为单选 string，内部升为数组）、`useResendEvent()`（`{eventId, channels?}`；onSuccess 失效 `notification.events` 前缀）
- query key：`notificationEvents(params)` → `['api', ns, 'notification.events', params]`
- i18n：`config.notification.events.*`

### 步骤

#### 1. `web/src/api/legacy.ts` —— events 段追加

```ts
/* ------------------------------------------------------------------ */
/* Notifications (v1) — events                                         */
/* ------------------------------------------------------------------ */

export interface NotificationEventListParams {
  /** RFC 3339 date-time, inclusive. */
  from?: string
  to?: string
  feature?: string[]
  subject?: string[]
  rule?: string[]
  channel?: string[]
  page?: number
  pageSize?: number
}

export interface NotificationEventPaginatedResponse {
  totalCount: number
  page: number
  pageSize: number
  items: NotificationEvent[]
}

export async function listNotificationEvents(
  params: NotificationEventListParams = {}
): Promise<NotificationEventPaginatedResponse> {
  const search = new URLSearchParams()
  if (params.from) search.set('from', params.from)
  if (params.to) search.set('to', params.to)
  // Repeated array params per spec: ?rule=a&rule=b
  params.feature?.forEach((value) => search.append('feature', value))
  params.subject?.forEach((value) => search.append('subject', value))
  params.rule?.forEach((value) => search.append('rule', value))
  params.channel?.forEach((value) => search.append('channel', value))
  if (params.page) search.set('page', String(params.page))
  if (params.pageSize) search.set('pageSize', String(params.pageSize))
  const qs = search.toString()
  return apiFetch<NotificationEventPaginatedResponse>(
    `/v1/notification/events${qs ? `?${qs}` : ''}`
  )
}

export async function getNotificationEvent(
  eventId: string
): Promise<NotificationEvent> {
  return apiFetch<NotificationEvent>(
    `/v1/notification/events/${encodeURIComponent(eventId)}`
  )
}

/** POST resend — 202 Accepted with no content; omitting `channels` resends to the original channels. */
export interface NotificationEventResendRequest {
  channels?: string[]
}

export async function resendNotificationEvent(
  eventId: string,
  body: NotificationEventResendRequest = {}
): Promise<void> {
  return apiFetch<void>(
    `/v1/notification/events/${encodeURIComponent(eventId)}/resend`,
    { method: 'POST', body: JSON.stringify(body) }
  )
}
```

#### 2. `web/src/api/query-keys.ts` —— `notificationRules` 之后追加

```ts
  notificationEvents: (params: object = {}) => ns('notification.events', params),
```

#### 3. `web/src/api/hooks.ts` —— rules 段后追加（import 区补 `listNotificationEvents, resendNotificationEvent, type NotificationEventListParams`）

```ts
/* ------------------------------------------------------------------ */
/* Notification events (v1) — read-only stream + resend                */
/* ------------------------------------------------------------------ */

export interface NotificationEventsParams {
  from?: string
  to?: string
  rule?: string
  channel?: string
  page: number
  pageSize: number
}

export function useNotificationEvents(params: NotificationEventsParams) {
  return useQuery({
    queryKey: queryKeys.notificationEvents(params),
    queryFn: () =>
      listNotificationEvents({
        ...(params.from ? { from: params.from } : {}),
        ...(params.to ? { to: params.to } : {}),
        // Single-select UI values lifted to the spec's repeated-array params.
        ...(params.rule ? { rule: [params.rule] } : {}),
        ...(params.channel ? { channel: [params.channel] } : {}),
        page: params.page,
        pageSize: params.pageSize,
      } satisfies NotificationEventListParams),
  })
}

export function useResendEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      eventId,
      channels,
    }: {
      eventId: string
      channels?: string[]
    }) =>
      // Empty selection means "resend to the original channels" — omit the field.
      resendNotificationEvent(
        eventId,
        channels?.length ? { channels } : {}
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('notification.events'),
      })
    },
  })
}
```

#### 4. Create `web/src/features/config/notification/events.tsx`（过滤表单 + 分页 + 行展开投递状态 + resend 确认）

```tsx
import { Fragment, useMemo, useState } from 'react'
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Play,
  RefreshCw,
  RotateCw,
  Send,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  getNotificationEvent,
  type NotificationEvent,
  type NotificationEventDeliveryState,
} from '@/api/legacy'
import {
  useNotificationChannels,
  useNotificationEvents,
  useNotificationRules,
  useResendEvent,
} from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { MultiSelect } from '@/components/multi-select'

const PAGE_SIZE = 20
const ALL = '__all__'

const DELIVERY_STATE_CLASS: Record<NotificationEventDeliveryState, string> = {
  SUCCESS:
    'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300',
  FAILED:
    'border-red-200 bg-red-50 text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300',
  SENDING:
    'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900 dark:bg-sky-950 dark:text-sky-300',
  PENDING:
    'border-zinc-200 bg-zinc-50 text-zinc-700 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300',
  RESENDING:
    'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300',
}

function toLocalInputValue(date: Date): string {
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

/**
 * Read-only notification event stream. Filters (from/to/rule/channel) are
 * drafted locally and applied on submit; rows expand into per-channel
 * delivery status with attempts, and any event can be resent (optionally to
 * a subset of channels).
 */
export function NotificationEventsPage() {
  const { t } = useTranslation()
  const { data: rulesData } = useNotificationRules({ page: 1, pageSize: 100 })
  const { data: channelsData } = useNotificationChannels({
    page: 1,
    pageSize: 100,
  })

  const channelNameById = useMemo(
    () =>
      new Map(
        (channelsData?.items ?? []).map((channel) => [
          channel.id,
          channel.name,
        ])
      ),
    [channelsData]
  )

  const [fromInput, setFromInput] = useState(
    toLocalInputValue(new Date(Date.now() - 7 * 24 * 60 * 60 * 1000))
  )
  const [toInput, setToInput] = useState('')
  const [ruleFilter, setRuleFilter] = useState(ALL)
  const [channelFilter, setChannelFilter] = useState(ALL)
  const [filters, setFilters] = useState<{
    from?: string
    to?: string
    rule?: string
    channel?: string
  }>({})
  const [page, setPage] = useState(1)

  const { data, isLoading, isFetching, refetch } = useNotificationEvents({
    ...filters,
    page,
    pageSize: PAGE_SIZE,
  })

  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [resendTarget, setResendTarget] = useState<NotificationEvent | null>(
    null
  )
  const [resendChannels, setResendChannels] = useState<string[]>([])
  const [refreshingId, setRefreshingId] = useState<string | null>(null)
  // Freshly refetched events shown instead of the stale list row.
  const [eventOverrides, setEventOverrides] = useState<
    Record<string, NotificationEvent>
  >({})

  const resendMutation = useResendEvent()

  const events = useMemo(
    () =>
      (data?.items ?? []).map((event) => eventOverrides[event.id] ?? event),
    [data, eventOverrides]
  )
  const totalPages = data
    ? Math.max(1, Math.ceil(data.totalCount / data.pageSize))
    : 1

  const applyFilters = () => {
    setPage(1)
    setFilters({
      from: fromInput ? new Date(fromInput).toISOString() : undefined,
      to: toInput ? new Date(toInput).toISOString() : undefined,
      rule: ruleFilter === ALL ? undefined : ruleFilter,
      channel: channelFilter === ALL ? undefined : channelFilter,
    })
  }

  const openResend = (event: NotificationEvent) => {
    setResendChannels([])
    setResendTarget(event)
  }

  const refreshEvent = async (eventId: string) => {
    setRefreshingId(eventId)
    try {
      const fresh = await getNotificationEvent(eventId)
      setEventOverrides((prev) => ({ ...prev, [eventId]: fresh }))
    } catch (error) {
      handleServerError(error)
    } finally {
      setRefreshingId(null)
    }
  }

  const resendOptions = useMemo(() => {
    const target = eventOverrides[resendTarget?.id ?? ''] ?? resendTarget
    return (target?.deliveryStatus ?? []).map((status) => ({
      value: status.channel.id,
      label:
        channelNameById.get(status.channel.id) ?? status.channel.id,
    }))
  }, [resendTarget, eventOverrides, channelNameById])

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.notification.events.title')}
          description={t('config.notification.events.description')}
          actions={
            <Button
              variant='outline'
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RotateCw className='size-4' />
              {t('config.notification.events.refresh')}
            </Button>
          }
        />
        <Card className='mt-6 py-0'>
          <CardHeader className='flex flex-wrap items-end gap-3 space-y-0'>
            <div className='space-y-1'>
              <CardTitle className='text-xs font-normal text-muted-foreground'>
                {t('config.notification.events.filter.from')}
              </CardTitle>
              <Input
                type='datetime-local'
                className='h-8 w-56'
                value={fromInput}
                onChange={(event) => setFromInput(event.target.value)}
              />
            </div>
            <div className='space-y-1'>
              <CardTitle className='text-xs font-normal text-muted-foreground'>
                {t('config.notification.events.filter.to')}
              </CardTitle>
              <Input
                type='datetime-local'
                className='h-8 w-56'
                value={toInput}
                onChange={(event) => setToInput(event.target.value)}
              />
            </div>
            <div className='space-y-1'>
              <CardTitle className='text-xs font-normal text-muted-foreground'>
                {t('config.notification.events.filter.rule')}
              </CardTitle>
              <Select value={ruleFilter} onValueChange={setRuleFilter}>
                <SelectTrigger className='h-8 w-52'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL}>
                    {t('config.notification.events.filter.allRules')}
                  </SelectItem>
                  {(rulesData?.items ?? []).map((rule) => (
                    <SelectItem key={rule.id} value={rule.id}>
                      {rule.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className='space-y-1'>
              <CardTitle className='text-xs font-normal text-muted-foreground'>
                {t('config.notification.events.filter.channel')}
              </CardTitle>
              <Select value={channelFilter} onValueChange={setChannelFilter}>
                <SelectTrigger className='h-8 w-52'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL}>
                    {t('config.notification.events.filter.allChannels')}
                  </SelectItem>
                  {(channelsData?.items ?? []).map((channel) => (
                    <SelectItem key={channel.id} value={channel.id}>
                      {channel.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button size='sm' className='h-8' onClick={applyFilters}>
              <Play className='size-3.5' />
              {t('config.notification.events.filter.apply')}
            </Button>
          </CardHeader>
          <CardContent className='pt-0'>
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='w-8 pl-6' />
                  <TableHead>
                    {t('config.notification.events.fields.createdAt')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.events.fields.type')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.events.fields.rule')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.events.fields.deliveryStatus')}
                  </TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('config.notification.events.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  Array.from({ length: 6 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell colSpan={6}>
                        <Skeleton className='h-5 w-full' />
                      </TableCell>
                    </TableRow>
                  ))
                ) : events.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.notification.events.empty')}
                    </TableCell>
                  </TableRow>
                ) : (
                  events.map((event) => {
                    const expanded = expandedId === event.id
                    return (
                      <Fragment key={event.id}>
                        <TableRow>
                          <TableCell className='pl-6'>
                            <Button
                              variant='ghost'
                              size='icon'
                              className='size-6'
                              onClick={() =>
                                setExpandedId(expanded ? null : event.id)
                              }
                            >
                              {expanded ? (
                                <ChevronDown className='size-4' />
                              ) : (
                                <ChevronRight className='size-4' />
                              )}
                              <span className='sr-only'>
                                {t('config.notification.events.toggleDetails')}
                              </span>
                            </Button>
                          </TableCell>
                          <TableCell className='text-muted-foreground'>
                            {formatDateTime(event.createdAt)}
                          </TableCell>
                          <TableCell>
                            <Badge variant='outline'>
                              {t(
                                `config.notification.rules.types.${event.type}`
                              )}
                            </Badge>
                          </TableCell>
                          <TableCell>{event.rule.name}</TableCell>
                          <TableCell>
                            <div className='flex flex-wrap gap-1'>
                              {event.deliveryStatus.map((status) => (
                                <Badge
                                  key={status.channel.id}
                                  variant='outline'
                                  className={
                                    DELIVERY_STATE_CLASS[status.state]
                                  }
                                >
                                  {channelNameById.get(status.channel.id) ??
                                    status.channel.id}
                                  ：{status.state}
                                </Badge>
                              ))}
                            </div>
                          </TableCell>
                          <TableCell className='pr-6'>
                            <div className='flex justify-end'>
                              <Button
                                variant='ghost'
                                size='sm'
                                className='h-7 px-2'
                                onClick={() => openResend(event)}
                              >
                                <Send className='size-4' />
                                {t('config.notification.events.resend')}
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                        {expanded && (
                          <TableRow>
                            <TableCell colSpan={6} className='bg-hover/30 p-4'>
                              <div className='flex items-center gap-2 pb-2'>
                                <span className='text-sm font-medium'>
                                  {t(
                                    'config.notification.events.delivery.title'
                                  )}
                                </span>
                                <Button
                                  variant='ghost'
                                  size='sm'
                                  className='h-6 px-2'
                                  disabled={refreshingId === event.id}
                                  onClick={() => void refreshEvent(event.id)}
                                >
                                  <RefreshCw className='size-3.5' />
                                  {t('config.notification.events.delivery.refresh')}
                                </Button>
                              </div>
                              {event.deliveryStatus.map((status) => (
                                <div
                                  key={status.channel.id}
                                  className='mb-3 space-y-1 rounded-lg border p-3 last:mb-0'
                                >
                                  <div className='flex flex-wrap items-center gap-2'>
                                    <span className='text-sm font-medium'>
                                      {channelNameById.get(status.channel.id) ??
                                        status.channel.id}
                                    </span>
                                    <Badge
                                      variant='outline'
                                      className={DELIVERY_STATE_CLASS[status.state]}
                                    >
                                      {status.state}
                                    </Badge>
                                    <span className='text-xs text-muted-foreground'>
                                      {formatDateTime(status.updatedAt)}
                                    </span>
                                  </div>
                                  {status.reason && (
                                    <p className='text-xs text-muted-foreground'>
                                      {status.reason}
                                    </p>
                                  )}
                                  {status.nextAttempt && (
                                    <p className='text-xs text-muted-foreground'>
                                      {t(
                                        'config.notification.events.delivery.nextAttempt',
                                        {
                                          time: formatDateTime(
                                            status.nextAttempt
                                          ),
                                        }
                                      )}
                                    </p>
                                  )}
                                  {status.attempts.length > 0 && (
                                    <div className='space-y-1'>
                                      {status.attempts.map(
                                        (attempt, index) => (
                                          <div
                                            key={`${attempt.timestamp}-${index}`}
                                            className='rounded border bg-background p-2 text-xs'
                                          >
                                            <div className='flex flex-wrap gap-2 text-muted-foreground'>
                                              <span>
                                                {formatDateTime(
                                                  attempt.timestamp
                                                )}
                                              </span>
                                              <span>{attempt.state}</span>
                                              {attempt.response.statusCode !==
                                                undefined && (
                                                <span>
                                                  HTTP{' '}
                                                  {attempt.response.statusCode}
                                                </span>
                                              )}
                                              <span>
                                                {attempt.response.durationMs}ms
                                              </span>
                                            </div>
                                            {attempt.response.body && (
                                              <pre className='mt-1 max-h-32 overflow-auto whitespace-pre-wrap break-all text-muted-foreground'>
                                                {attempt.response.body}
                                              </pre>
                                            )}
                                          </div>
                                        )
                                      )}
                                    </div>
                                  )}
                                </div>
                              ))}
                              <div className='rounded-lg border p-3'>
                                <p className='pb-1 text-sm font-medium'>
                                  {t('config.notification.events.payload')}
                                </p>
                                <pre className='max-h-48 overflow-auto whitespace-pre-wrap break-all text-xs text-muted-foreground'>
                                  {JSON.stringify(event.payload, null, 2)}
                                </pre>
                              </div>
                            </TableCell>
                          </TableRow>
                        )}
                      </Fragment>
                    )
                  })
                )}
              </TableBody>
            </Table>
            {data && (
              <div className='flex items-center justify-end gap-2 pt-4 pb-2'>
                <span className='mr-auto text-sm text-muted-foreground'>
                  {t('config.notification.events.pagination.total', {
                    total: data.totalCount,
                  })}
                </span>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page <= 1 || isFetching}
                  onClick={() => setPage((p) => p - 1)}
                >
                  <ChevronLeft className='size-4' />
                </Button>
                <span className='text-sm tabular-nums'>
                  {data.page} / {totalPages}
                </span>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page >= totalPages || isFetching}
                  onClick={() => setPage((p) => p + 1)}
                >
                  <ChevronRight className='size-4' />
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      </Main>

      <ConfirmDialog
        open={Boolean(resendTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setResendTarget(null)
            setResendChannels([])
          }
        }}
        title={t('config.notification.events.resendConfirm.title')}
        desc={
          <div className='space-y-3'>
            <p>
              {t('config.notification.events.resendConfirm.description', {
                type: t(
                  `config.notification.rules.types.${resendTarget?.type ?? 'invoice.created'}`
                ),
              })}
            </p>
            <div>
              <p className='pb-1 text-sm font-medium'>
                {t('config.notification.events.resendConfirm.channels')}
              </p>
              <MultiSelect
                options={resendOptions}
                value={resendChannels}
                onChange={setResendChannels}
                placeholder={t(
                  'config.notification.events.resendConfirm.channelsPlaceholder'
                )}
                searchPlaceholder={t('common.search')}
                emptyText={t(
                  'config.notification.events.resendConfirm.noChannels'
                )}
              />
              <p className='pt-1 text-xs text-muted-foreground'>
                {t('config.notification.events.resendConfirm.channelsHint')}
              </p>
            </div>
          </div>
        }
        confirmText={t('config.notification.events.resend')}
        cancelBtnText={t('common.cancel')}
        isLoading={resendMutation.isPending}
        handleConfirm={() => {
          if (!resendTarget) return
          resendMutation.mutate(
            {
              eventId: resendTarget.id,
              channels: resendChannels.length ? resendChannels : undefined,
            },
            {
              onSuccess: () => {
                toast.success(
                  t('config.notification.events.toast.resent')
                )
                setResendTarget(null)
                setResendChannels([])
                // 202 Accepted is async; pull fresh delivery status shortly.
                setTimeout(() => void refetch(), 1500)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </>
  )
}
```

（行内 `Fragment key={event.id}` 包每行事件的两个 `TableRow`（主行 + 展开行）；import 已含 `Fragment` 与 `Input`。）

#### 5. `web/src/routes/_authenticated/config/notification/events/index.tsx`

```tsx
import { createFileRoute } from '@tanstack/react-router'
import { NotificationEventsPage } from '@/features/config/notification/events'

export const Route = createFileRoute(
  '/_authenticated/config/notification/events/'
)({
  component: NotificationEventsPage,
})
```

#### 6. i18n（`config.notification.events`，两份同步）

`zh-CN.ts`：

```ts
      events: {
        title: '通知事件',
        description: '规则触发的通知事件流（只读），支持按时间/规则/渠道过滤与重发。',
        refresh: '刷新',
        resend: '重发',
        empty: '所选过滤条件下暂无通知事件',
        actions: '操作',
        pagination: { total: '共 {{total}} 条' },
        fields: {
          createdAt: '触发时间',
          type: '事件类型',
          rule: '来源规则',
          deliveryStatus: '投递状态',
        },
        filter: {
          from: '开始时间',
          to: '结束时间',
          rule: '规则',
          channel: '渠道',
          allRules: '全部规则',
          allChannels: '全部渠道',
          apply: '应用过滤',
        },
        delivery: {
          title: '投递明细',
          refresh: '刷新该事件',
          nextAttempt: '下次重试：{{time}}',
        },
        payload: '事件载荷（JSON）',
        toggleDetails: '展开/收起投递明细',
        resendConfirm: {
          title: '重发通知事件',
          description:
            '该{{type}}事件将按原始载荷重新投递。重发为异步受理，结果以投递状态与错误提示为准。',
          channels: '重发渠道（可选）',
          channelsPlaceholder: '选择渠道（可多选，留空=原渠道）',
          channelsHint: '不选则重发到该事件原本投递的全部渠道。',
          noChannels: '该事件没有可用的投递渠道记录',
        },
        toast: {
          resent: '重发请求已受理，投递状态稍后刷新可见。',
        },
      },
```

`en.ts`：

```ts
      events: {
        title: 'Notification Events',
        description: 'Read-only stream of rule-triggered notification events; filter by time/rule/channel and resend.',
        refresh: 'Refresh',
        resend: 'Resend',
        empty: 'No notification events for the selected filters',
        actions: 'Actions',
        pagination: { total: '{{total}} total' },
        fields: {
          createdAt: 'Triggered At',
          type: 'Event Type',
          rule: 'Source Rule',
          deliveryStatus: 'Delivery Status',
        },
        filter: {
          from: 'From',
          to: 'To',
          rule: 'Rule',
          channel: 'Channel',
          allRules: 'All rules',
          allChannels: 'All channels',
          apply: 'Apply Filters',
        },
        delivery: {
          title: 'Delivery Details',
          refresh: 'Refresh event',
          nextAttempt: 'Next attempt: {{time}}',
        },
        payload: 'Payload (JSON)',
        toggleDetails: 'Toggle delivery details',
        resendConfirm: {
          title: 'Resend notification event',
          description:
            'This {{type}} event will be redelivered with its original payload. Resend is accepted asynchronously; check delivery status and error toasts for the outcome.',
          channels: 'Channels (optional)',
          channelsPlaceholder: 'Select channels (empty = original)',
          channelsHint: 'Leave empty to resend to every channel the event originally delivered to.',
          noChannels: 'No delivery channels recorded for this event',
        },
        toast: {
          resent: 'Resend accepted; delivery status will update shortly.',
        },
      },
```

#### 7. 验证与提交

```bash
cd web && pnpm build && pnpm lint && pnpm test:e2e
```

预期：构建/静态检查通过；既有 2 条冒烟不回归。浏览器手测（配合 #15 的 test 按钮造事件）：默认近 7 天过滤出数；改 from/to/规则/渠道后点「应用过滤」结果收敛、页码重置到 1；行展开显示每渠道 state/reason/attempt（含 HTTP 状态码、耗时、响应体）与载荷 JSON；行内「刷新该事件」单条拉取最新投递状态；「重发」确认弹窗可留空（原渠道）或勾选子集渠道，确认后 toast 受理提示、约 1.5s 后列表自动刷新可见 RESENDING/状态变化；后端拒绝（如渠道已删除）时 toast 透出错误原文。

Commit：

```
feat(admin): 通知事件流与 resend
```

