import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { Invoice } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { useInvoices } from '@/api/hooks'
import { formatAmount, formatDateTime } from '@/lib/format'
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
import { StatusBadge } from '@/components/status-badge'

const STATUS_OPTIONS = [
  'draft',
  'issuing',
  'issued',
  'payment_processing',
  'overdue',
  'paid',
  'uncollectible',
  'voided',
]

export function InvoicesPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [status, setStatus] = useState<string | undefined>()

  const { data, isLoading, isFetching } = useInvoices({
    page,
    pageSize,
    status,
  })

  const columns: ColumnDef<Invoice, unknown>[] = [
    {
      accessorKey: 'number',
      header: t('invoices.fields.number'),
      cell: ({ row }) => (
        <Link
          to='/invoices/$invoiceId'
          params={{ invoiceId: row.original.id }}
          className='font-medium hover:underline'
        >
          {row.original.number}
        </Link>
      ),
    },
    {
      accessorKey: 'customer',
      header: t('invoices.fields.customer'),
      cell: ({ row }) => row.original.customer.name,
    },
    {
      accessorKey: 'totals.total',
      header: t('invoices.fields.total'),
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatAmount(row.original.totals.total, row.original.currency)}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: t('invoices.fields.status'),
      cell: ({ row }) => (
        <StatusBadge domain='invoice' value={row.original.status} />
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('invoices.fields.createdAt'),
      cell: ({ row }) => (
        <span className='text-muted-foreground'>
          {formatDateTime(row.original.createdAt)}
        </span>
      ),
    },
  ]

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('invoices.title')}
          description={t('invoices.description')}
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
              <SelectTrigger className='h-8 w-44'>
                <SelectValue placeholder={t('invoices.filter.status')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>
                  {t('invoices.filter.allStatuses')}
                </SelectItem>
                {STATUS_OPTIONS.map((option) => (
                  <SelectItem key={option} value={option}>
                    {t(`invoice.status.${option}`, { defaultValue: option })}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          }
          emptyMessage={t('invoices.empty')}
        />
      </Main>
    </>
  )
}
