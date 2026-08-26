import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { Subscription } from '@openmeter/client'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useCancelSubscription,
  useCustomer,
  usePlans,
  useSubscriptions,
} from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { ServerTable } from '@/components/data-table/server-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { StatusBadge } from '@/components/status-badge'
import { CreateSubscriptionDialog } from './create-subscription-dialog'

/** Resolves the customer display name for a subscription row. */
function CustomerCell({ customerId }: { customerId: string }) {
  const { data } = useCustomer(customerId)
  return data ? (
    <Link
      to='/customers/$customerId'
      params={{ customerId }}
      className='hover:underline'
    >
      {data.name}
    </Link>
  ) : (
    <span className='font-mono text-xs text-muted-foreground'>
      {customerId}
    </span>
  )
}

/** Resolves the plan name for a subscription row, linking to the
 * subscription detail. */
function PlanCell({
  planId,
  subscriptionId,
}: {
  planId: string | undefined
  subscriptionId: string
}) {
  const { data: plans } = usePlans()
  const plan = plans?.find((candidate) => candidate.id === planId)
  if (!plan) {
    return <span className='text-muted-foreground'>—</span>
  }

  return (
    <Link
      to='/subscriptions/$subscriptionId'
      params={{ subscriptionId }}
      className='hover:underline'
    >
      {plan.name}
    </Link>
  )
}

const STATUS_OPTIONS = ['active', 'inactive', 'canceled', 'scheduled']

export function SubscriptionsPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [status, setStatus] = useState<string | undefined>()
  const [createOpen, setCreateOpen] = useState(false)
  const [cancelTarget, setCancelTarget] = useState<Subscription | null>(null)

  const { data, isLoading, isFetching } = useSubscriptions({
    page,
    pageSize,
    status,
  })

  const cancelMutation = useCancelSubscription()

  const columns: ColumnDef<Subscription, unknown>[] = [
    {
      accessorKey: 'customer',
      header: t('subscriptions.fields.customer'),
      cell: ({ row }) => <CustomerCell customerId={row.original.customerId} />,
    },
    {
      accessorKey: 'plan',
      header: t('subscriptions.fields.plan'),
      cell: ({ row }) => (
        <PlanCell
          planId={row.original.planId}
          subscriptionId={row.original.id}
        />
      ),
    },
    {
      accessorKey: 'status',
      header: t('subscriptions.fields.status'),
      cell: ({ row }) => (
        <StatusBadge domain='subscription' value={row.original.status} />
      ),
    },
    {
      accessorKey: 'billingAnchor',
      header: t('subscriptions.fields.billingAnchor'),
      cell: ({ row }) => (
        <span className='text-muted-foreground'>
          {formatDateTime(row.original.billingAnchor)}
        </span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const subscription = row.original
        const cancellable =
          subscription.status === 'active' || subscription.status === 'inactive'
        return cancellable ? (
          <Button
            variant='ghost'
            size='sm'
            className='h-7 px-2 text-destructive hover:text-destructive'
            onClick={() => setCancelTarget(subscription)}
          >
            {t('subscriptions.cancel')}
          </Button>
        ) : null
      },
    },
  ]

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('subscriptions.title')}
          description={t('subscriptions.description')}
          actions={
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className='size-4' />
              {t('subscriptions.create.title')}
            </Button>
          }
        />
        <ServerTable
          className='mt-6'
          columns={columns}
          data={data?.data ?? []}
          page={page}
          pageSize={pageSize}
          total={data?.meta.page.total}
          isLoading={isLoading}
          isFetching={isFetching}
          onPageChange={(next) => {
            if (next.pageSize !== pageSize) {
              setPageSize(next.pageSize)
              setPage(1)
            } else {
              setPage(next.pageIndex + 1)
            }
          }}
          toolbar={
            <Select
              value={status ?? 'all'}
              onValueChange={(value) => {
                setPage(1)
                setStatus(value === 'all' ? undefined : value)
              }}
            >
              <SelectTrigger className='h-8 w-40'>
                <SelectValue placeholder={t('subscriptions.filter.status')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>
                  {t('subscriptions.filter.allStatuses')}
                </SelectItem>
                {STATUS_OPTIONS.map((option) => (
                  <SelectItem key={option} value={option}>
                    {t(`subscription.status.${option}`, {
                      defaultValue: option,
                    })}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          }
          emptyMessage={t('subscriptions.empty')}
        />
      </Main>

      <CreateSubscriptionDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
      />

      <ConfirmDialog
        open={Boolean(cancelTarget)}
        onOpenChange={(open) => !open && setCancelTarget(null)}
        title={t('subscriptions.cancelConfirm.title')}
        desc={t('subscriptions.cancelConfirm.description')}
        confirmText={t('subscriptions.cancelConfirm.confirm')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={cancelMutation.isPending}
        handleConfirm={() => {
          if (!cancelTarget) return
          cancelMutation.mutate(
            { subscriptionId: cancelTarget.id, body: { timing: 'immediate' } },
            {
              onSuccess: () => {
                toast.success(t('subscriptions.toast.canceled'))
                setCancelTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </>
  )
}
