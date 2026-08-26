import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { CommerceOrder, Customer } from '@openmeter/client'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCustomer, useOrders, type OrderListParams } from '@/api/hooks'
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

/** Resolves the customer display name for an order row. */
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

const STATUS_OPTIONS: NonNullable<OrderListParams['status']>[] = [
  'created',
  'awaiting_payment',
  'paid',
  'fulfilled',
  'cancelled',
  'expired',
  'refund_pending',
  'partially_refunded',
  'refunded',
]

/**
 * Server-paginated order list (newest first) with optional customer and
 * lifecycle-status filters; rows link to the order detail view.
 */
export function OrdersPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [status, setStatus] = useState<OrderListParams['status']>()
  const [customer, setCustomer] = useState<Customer | null>(null)

  const { data, isLoading, isFetching } = useOrders({
    page,
    pageSize,
    status,
    customerId: customer?.id,
  })

  const columns: ColumnDef<CommerceOrder, unknown>[] = [
    {
      accessorKey: 'id',
      header: t('commerce.orders.fields.id'),
      cell: ({ row }) => (
        <Link
          to='/commerce/orders/$orderId'
          params={{ orderId: row.original.id }}
          className='font-mono text-xs hover:underline'
        >
          {row.original.id}
        </Link>
      ),
    },
    {
      accessorKey: 'customer',
      header: t('commerce.orders.fields.billingCustomerId'),
      cell: ({ row }) => (
        <CustomerCell customerId={row.original.billingCustomerId} />
      ),
    },
    {
      accessorKey: 'kind',
      header: t('commerce.orders.fields.kind'),
      cell: ({ row }) => (
        <EnumBadge
          domain='commerce'
          kind='orderKind'
          value={row.original.kind}
        />
      ),
    },
    {
      accessorKey: 'amount',
      header: t('commerce.orders.fields.amount'),
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatFen(row.original.amountFen, row.original.currency)}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: t('commerce.orders.fields.status'),
      cell: ({ row }) => (
        <StatusBadge domain='order' value={row.original.status} />
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('commerce.orders.fields.createdAt'),
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
            to='/commerce/orders/$orderId'
            params={{ orderId: row.original.id }}
          >
            {t('commerce.orders.view')}
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
          title={t('commerce.orders.title')}
          description={t('commerce.orders.description')}
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
                      : (value as OrderListParams['status'])
                  )
                }}
              >
                <SelectTrigger className='h-8 w-44'>
                  <SelectValue
                    placeholder={t('commerce.orders.filter.status')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='all'>
                    {t('commerce.orders.filter.allStatuses')}
                  </SelectItem>
                  {STATUS_OPTIONS.map((option) => (
                    <SelectItem key={option} value={option}>
                      {t(`order.status.${option}`, {
                        defaultValue: option,
                      })}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          }
          emptyMessage={t('commerce.orders.empty')}
        />
      </Main>
    </>
  )
}
