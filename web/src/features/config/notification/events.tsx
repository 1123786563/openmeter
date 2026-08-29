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
  useNotificationChannels,
  useNotificationEvents,
  useNotificationRules,
  useResendEvent,
} from '@/api/hooks'
import {
  getNotificationEvent,
  type NotificationEvent,
  type NotificationEventDeliveryState,
} from '@/api/legacy'
import { formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
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
import { MultiSelect } from '@/components/multi-select'
import { PageHeader } from '@/components/page-header'

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
        (channelsData?.items ?? []).map((channel) => [channel.id, channel.name])
      ),
    [channelsData]
  )

  // Lazy initializer: Date.now() is impure and must run once, not during every render.
  const [fromInput, setFromInput] = useState(() =>
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
    () => (data?.items ?? []).map((event) => eventOverrides[event.id] ?? event),
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
      label: channelNameById.get(status.channel.id) ?? status.channel.id,
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
                                  className={DELIVERY_STATE_CLASS[status.state]}
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
                                  {t(
                                    'config.notification.events.delivery.refresh'
                                  )}
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
                                      className={
                                        DELIVERY_STATE_CLASS[status.state]
                                      }
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
                                      {status.attempts.map((attempt, index) => (
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
                                            <pre className='mt-1 max-h-32 overflow-auto break-all whitespace-pre-wrap text-muted-foreground'>
                                              {attempt.response.body}
                                            </pre>
                                          )}
                                        </div>
                                      ))}
                                    </div>
                                  )}
                                </div>
                              ))}
                              <div className='rounded-lg border p-3'>
                                <p className='pb-1 text-sm font-medium'>
                                  {t('config.notification.events.payload')}
                                </p>
                                <pre className='max-h-48 overflow-auto text-xs break-all whitespace-pre-wrap text-muted-foreground'>
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
                toast.success(t('config.notification.events.toast.resent'))
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
