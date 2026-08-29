import { useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, Pencil, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useNotificationChannels,
  useNotificationRules,
  useToggleRule,
} from '@/api/hooks'
import type {
  NotificationRule,
  NotificationRuleInvoiceCreated,
  NotificationRuleInvoiceUpdated,
} from '@/api/legacy'
import { handleServerError } from '@/lib/handle-server-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
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
import { RuleFormDialog } from './rule-form-dialog'

const PAGE_SIZE = 20

const TYPE_BADGE_CLASS: Record<NotificationRule['type'], string> = {
  'entitlements.balance.threshold':
    'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300',
  'entitlements.reset':
    'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900 dark:bg-sky-950 dark:text-sky-300',
  'invoice.created':
    'border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-900 dark:bg-violet-950 dark:text-violet-300',
  'invoice.updated':
    'border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-900 dark:bg-violet-950 dark:text-violet-300',
}

/** This iteration ships editors for invoice types only; other types still toggle fine. */
type EditableRule =
  | NotificationRuleInvoiceCreated
  | NotificationRuleInvoiceUpdated

function isEditableType(rule: NotificationRule): rule is EditableRule {
  return rule.type === 'invoice.created' || rule.type === 'invoice.updated'
}

/** Notification rule management: typed list, create/edit, enable/disable. */
export function NotificationRulesPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const { data, isLoading, isFetching } = useNotificationRules({
    page,
    pageSize: PAGE_SIZE,
  })

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

  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<EditableRule | null>(null)
  const [toggleTarget, setToggleTarget] = useState<NotificationRule | null>(
    null
  )

  const toggleMutation = useToggleRule()

  const openCreate = () => {
    setEditTarget(null)
    setFormOpen(true)
  }

  const openEdit = (rule: NotificationRule) => {
    if (!isEditableType(rule)) return
    setEditTarget(rule)
    setFormOpen(true)
  }

  const rules = data?.items ?? []
  const totalPages = data
    ? Math.max(1, Math.ceil(data.totalCount / data.pageSize))
    : 1

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.notification.rules.title')}
          description={t('config.notification.rules.description')}
          actions={
            <Button onClick={openCreate}>
              <Plus className='size-4' />
              {t('config.notification.rules.create')}
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
                    {t('config.notification.rules.fields.type')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.rules.fields.name')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.rules.fields.channels')}
                  </TableHead>
                  <TableHead>
                    {t('config.notification.rules.fields.status')}
                  </TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('config.notification.rules.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell className='pl-6'>
                      <Badge
                        variant='outline'
                        className={TYPE_BADGE_CLASS[rule.type]}
                      >
                        {t(`config.notification.rules.types.${rule.type}`)}
                      </Badge>
                    </TableCell>
                    <TableCell className='font-medium'>{rule.name}</TableCell>
                    <TableCell className='text-muted-foreground'>
                      {rule.channels
                        .map(
                          (channel) =>
                            channelNameById.get(channel.id) ?? channel.id
                        )
                        .join(', ') || '—'}
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={!rule.disabled}
                        onCheckedChange={() => setToggleTarget(rule)}
                      />
                    </TableCell>
                    <TableCell className='pr-6'>
                      <div className='flex justify-end gap-1'>
                        {isEditableType(rule) && (
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-7 px-2'
                            onClick={() => openEdit(rule)}
                          >
                            <Pencil className='size-4' />
                            {t('common.edit')}
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {rules.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.notification.rules.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
          {data && (
            <div className='flex items-center justify-end gap-2 pt-4'>
              <span className='mr-auto text-sm text-muted-foreground'>
                {t('config.notification.rules.pagination.total', {
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

      <RuleFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        rule={editTarget ?? undefined}
      />

      <ConfirmDialog
        open={Boolean(toggleTarget)}
        onOpenChange={(open) => !open && setToggleTarget(null)}
        title={
          toggleTarget?.disabled
            ? t('config.notification.rules.toggleConfirm.enableTitle')
            : t('config.notification.rules.toggleConfirm.disableTitle')
        }
        desc={
          toggleTarget?.disabled
            ? t('config.notification.rules.toggleConfirm.enableDescription', {
                name: toggleTarget?.name ?? '',
              })
            : t('config.notification.rules.toggleConfirm.disableDescription', {
                name: toggleTarget?.name ?? '',
              })
        }
        confirmText={
          toggleTarget?.disabled
            ? t('config.notification.rules.enable')
            : t('config.notification.rules.disable')
        }
        cancelBtnText={t('common.cancel')}
        destructive={!toggleTarget?.disabled}
        isLoading={toggleMutation.isPending}
        handleConfirm={() => {
          if (!toggleTarget) return
          const wasDisabled = toggleTarget.disabled
          toggleMutation.mutate(
            { rule: toggleTarget, disabled: !wasDisabled },
            {
              onSuccess: () => {
                toast.success(
                  wasDisabled
                    ? t('config.notification.rules.toast.enabled')
                    : t('config.notification.rules.toast.disabled')
                )
                setToggleTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </>
  )
}
