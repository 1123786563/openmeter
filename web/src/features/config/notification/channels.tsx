import { useState } from 'react'
import { ChevronLeft, ChevronRight, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNotificationChannels } from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
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
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { ChannelFormDialog } from './channel-form-dialog'

const PAGE_SIZE = 20

/** Webhook notification channel management (list + create). */
export function NotificationChannelsPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const { data, isLoading, isFetching } = useNotificationChannels({
    page,
    pageSize: PAGE_SIZE,
  })
  const [formOpen, setFormOpen] = useState(false)

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
            <Button onClick={() => setFormOpen(true)}>
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
                  </TableRow>
                ))}
                {channels.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={4}
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

      <ChannelFormDialog open={formOpen} onOpenChange={setFormOpen} />
    </>
  )
}
