import { useState } from 'react'
import { ChevronLeft, ChevronRight, Pencil, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import type { NotificationChannel } from '@/api/legacy'
import {
  useDeleteChannel,
  useNotificationChannels,
  useUpdateChannel,
} from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import { PageHeader } from '@/components/page-header'
import { ChannelFormDialog } from './channel-form-dialog'

const PAGE_SIZE = 20

/**
 * Rebuilds the full PUT body from a row (PUT is a full replacement).
 * `metadata` is backfilled like `signingSecret`: the UI never sets it, but
 * channels created via the API can carry it and an omitted field is cleared
 * server-side.
 */
function toChannelBody(channel: NotificationChannel) {
  const customHeaders = Object.fromEntries(
    Object.entries(channel.customHeaders ?? {}).filter(([key]) => key !== '')
  )
  const hasHeaders = Object.keys(customHeaders).length > 0
  return {
    type: 'WEBHOOK' as const,
    name: channel.name,
    url: channel.url,
    disabled: channel.disabled,
    ...(hasHeaders ? { customHeaders } : {}),
    ...(channel.signingSecret
      ? { signingSecret: channel.signingSecret }
      : {}),
    ...(channel.metadata && Object.keys(channel.metadata).length > 0
      ? { metadata: channel.metadata }
      : {}),
  }
}

/** Webhook notification channel management (list + create/edit/toggle/delete). */
export function NotificationChannelsPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const { data, isLoading, isFetching } = useNotificationChannels({
    page,
    pageSize: PAGE_SIZE,
  })

  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<NotificationChannel | null>(null)
  const [toggleTarget, setToggleTarget] = useState<NotificationChannel | null>(
    null
  )
  const [deleteTarget, setDeleteTarget] = useState<NotificationChannel | null>(
    null
  )

  const updateMutation = useUpdateChannel()
  const deleteMutation = useDeleteChannel()

  const openCreate = () => {
    setEditTarget(null)
    setFormOpen(true)
  }

  const openEdit = (channel: NotificationChannel) => {
    setEditTarget(channel)
    setFormOpen(true)
  }

  const channels = data?.items ?? []
  const totalPages = data
    ? Math.max(1, Math.ceil(data.totalCount / data.pageSize))
    : 1

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.notification.channels.title')}
          description={t('config.notification.channels.description')}
          actions={
            <Button onClick={openCreate}>
              <Plus className='size-4' />
              {t('config.notification.channels.create')}
            </Button>
          }
        />
        <div className='mt-6'>
          {isLoading ? (
            <div className='space-y-2'>
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className='h-10 w-full' />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('config.notification.channels.fields.name')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.channels.fields.url')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.channels.fields.status')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.channels.fields.createdAt')}
                  </TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('config.notification.channels.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {channels.map((channel) => (
                  <TableRow key={channel.id}>
                    <TableCell className='pl-6 font-medium'>
                      {channel.name}
                    </TableCell>
                    <TableCell className='max-w-[280px] truncate font-mono text-xs text-muted-foreground'>
                      {channel.url}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant='outline'
                        className={
                          channel.disabled
                            ? ''
                            : 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300'
                        }
                      >
                        {channel.disabled
                          ? t('config.notification.channels.disabled')
                          : t('config.notification.channels.enabled')}
                      </Badge>
                    </TableCell>
                    <TableCell className='text-muted-foreground'>
                      {formatDateTime(channel.createdAt)}
                    </TableCell>
                    <TableCell className='pr-6'>
                      <div className='flex justify-end gap-1'>
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2'
                          onClick={() => openEdit(channel)}
                        >
                          <Pencil className='size-4' />
                          {t('common.edit')}
                        </Button>
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2'
                          onClick={() => setToggleTarget(channel)}
                        >
                          {channel.disabled
                            ? t('config.notification.channels.enable')
                            : t('config.notification.channels.disable')}
                        </Button>
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2 text-destructive hover:text-destructive'
                          onClick={() => setDeleteTarget(channel)}
                        >
                          <Trash2 className='size-4' />
                          {t('config.notification.channels.delete')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {channels.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.notification.channels.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
          {data && (
            <div className='flex items-center justify-end gap-2 pt-4'>
              <span className='mr-auto text-sm text-muted-foreground'>
                {t('config.notification.channels.pagination.total', {
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
        </div>
      </Main>

      <ChannelFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        channel={editTarget ?? undefined}
      />

      <ConfirmDialog
        open={Boolean(toggleTarget)}
        onOpenChange={(open) => !open && setToggleTarget(null)}
        title={
          toggleTarget?.disabled
            ? t('config.notification.channels.toggleConfirm.enableTitle')
            : t('config.notification.channels.toggleConfirm.disableTitle')
        }
        desc={
          toggleTarget?.disabled
            ? t('config.notification.channels.toggleConfirm.enableDescription', {
                name: toggleTarget?.name ?? '',
              })
            : t('config.notification.channels.toggleConfirm.disableDescription', {
                name: toggleTarget?.name ?? '',
              })
        }
        confirmText={
          toggleTarget?.disabled
            ? t('config.notification.channels.enable')
            : t('config.notification.channels.disable')
        }
        cancelBtnText={t('common.cancel')}
        destructive={!toggleTarget?.disabled}
        isLoading={updateMutation.isPending}
        handleConfirm={() => {
          if (!toggleTarget) return
          const wasDisabled = toggleTarget.disabled
          updateMutation.mutate(
            {
              channelId: toggleTarget.id,
              body: { ...toChannelBody(toggleTarget), disabled: !wasDisabled },
            },
            {
              onSuccess: () => {
                toast.success(
                  wasDisabled
                    ? t('config.notification.channels.toast.enabled')
                    : t('config.notification.channels.toast.disabled')
                )
                setToggleTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('config.notification.channels.deleteConfirm.title')}
        desc={t('config.notification.channels.deleteConfirm.description', {
          name: deleteTarget?.name ?? '',
        })}
        confirmText={t('config.notification.channels.delete')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (!deleteTarget) return
          deleteMutation.mutate(deleteTarget.id, {
            onSuccess: () => {
              toast.success(t('config.notification.channels.toast.deleted'))
              setDeleteTarget(null)
            },
            onError: handleServerError,
          })
        }}
      />
    </>
  )
}
