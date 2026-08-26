import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { CommerceRefund, Customer } from '@openmeter/client'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCustomer, useRefunds, type RefundListParams } from '@/api/hooks'
import { formatDateTime, formatFen } from '@/lib/format'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ServerTable } from '@/components/data-table/server-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { EnumBadge, StatusBadge } from '@/components/status-badge'
import { CustomerPicker } from '@/features/customers/customer-picker'

/** Resolves the customer display name for a refund row. */
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

const STATUS_OPTIONS: NonNullable<RefundListParams['status']>[] = [
  'pending_fence',
  'provider_processing',
  'ledger_reversing',
  'fulfilled',
  'failed',
]

/**
 * Server-paginated refund list (newest first) with optional customer and
 * status filters; rows link to the read-only refund detail view.
 */
export function RefundsPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [status, setStatus] = useState<RefundListParams['status']>()
  const [customer, setCustomer] = useState<Customer | null>(null)

  const { data, isLoading, isFetching } = useRefunds({
    page,
    pageSize,
    status,
    customerId: customer?.id,
  })

  const columns: ColumnDef<CommerceRefund, unknown>[] = [
    {
      accessorKey: 'id',
      header: t('commerce.refunds.fields.id'),
      cell: ({ row }) => (
        <Link
          to='/commerce/refunds/$refundId'
          params={{ refundId: row.original.id }}
          className='font-mono text-xs hover:underline'
        >
          {row.original.id}
        </Link>
      ),
    },
    {
      accessorKey: 'customer',
      header: t('commerce.refunds.fields.billingCustomerId'),
      cell: ({ row }) => (
        <CustomerCell customerId={row.original.billingCustomerId} />
      ),
    },
    {
      accessorKey: 'orderId',
      header: t('commerce.refunds.fields.orderId'),
      cell: ({ row }) => (
        <Link
          to='/commerce/orders/$orderId'
          params={{ orderId: row.original.orderId }}
          className='font-mono text-xs hover:underline'
        >
          {row.original.orderId}
        </Link>
      ),
    },
    {
      accessorKey: 'amount',
      header: t('commerce.refunds.fields.amount'),
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatFen(row.original.amountFen, 'CNY')}
        </span>
      ),
    },
    {
      accessorKey: 'provider',
      header: t('commerce.refunds.fields.provider'),
      cell: ({ row }) => (
        <EnumBadge
          domain='commerce'
          kind='refundProvider'
          value={row.original.provider}
        />
      ),
    },
    {
      accessorKey: 'status',
      header: t('commerce.refunds.fields.status'),
      cell: ({ row }) => (
        <StatusBadge domain='refund' value={row.original.status} />
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('commerce.refunds.fields.createdAt'),
      cell: ({ row }) => (
        <span className='text-muted-foreground'>
          {formatDateTime(row.original.createdAt)}
        </span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button asChild variant='ghost' size='sm' className='h-7 px-2'>
          <Link
            to='/commerce/refunds/$refundId'
            params={{ refundId: row.original.id }}
          >
            {t('commerce.refunds.view')}
          </Link>
        </Button>
      ),
    },
  ]

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('commerce.refunds.title')}
          description={t('commerce.refunds.description')}
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
            <div className='flex items-center gap-2'>
              <div className='flex items-center gap-1'>
                <CustomerPicker
                  value={customer}
                  onChange={(next) => {
                    setPage(1)
                    setCustomer(next)
                  }}
                  className='h-8 w-56'
                />
                {customer && (
                  <Button
                    variant='ghost'
                    size='icon'
                    className='size-8'
                    onClick={() => {
                      setPage(1)
                      setCustomer(null)
                    }}
                  >
                    <X className='size-4' />
                  </Button>
                )}
              </div>
              <Select
                value={status ?? 'all'}
                onValueChange={(value) => {
                  setPage(1)
                  setStatus(
                    value === 'all'
                      ? undefined
                      : (value as RefundListParams['status'])
                  )
                }}
              >
                <SelectTrigger className='h-8 w-44'>
                  <SelectValue
                    placeholder={t('commerce.refunds.filter.status')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='all'>
                    {t('commerce.refunds.filter.allStatuses')}
                  </SelectItem>
                  {STATUS_OPTIONS.map((option) => (
                    <SelectItem key={option} value={option}>
                      {t(`refund.status.${option}`, {
                        defaultValue: option,
                      })}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          }
          emptyMessage={t('commerce.refunds.empty')}
        />
      </Main>
    </>
  )
}
