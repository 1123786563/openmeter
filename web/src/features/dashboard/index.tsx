import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowRight, FileText, ReceiptText, Users, Wallet } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/client'
import { queryKeys } from '@/api/query-keys'
import { formatAmount, formatDateTime } from '@/lib/format'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
import { StatusBadge } from '@/components/status-badge'

/**
 * Namespace overview. Every metric comes from a real list endpoint's
 * pagination `total`; the month total iterates the month's invoices (the API
 * has no aggregate endpoint). Namespace-wide credit granting totals are not
 * available (credit grants are only listable per customer) and are omitted.
 */
function monthStart(): Date {
  const now = new Date()
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1))
}

function useDashboardStats() {
  const customers = useQuery({
    queryKey: queryKeys.customers({ page: 1, pageSize: 1 }),
    queryFn: ({ signal }) =>
      api.customers.list({ page: { number: 1, size: 1 } }, { signal }),
  })

  const activeSubscriptions = useQuery({
    queryKey: queryKeys.subscriptions({
      page: 1,
      pageSize: 1,
      status: 'active',
    }),
    queryFn: ({ signal }) =>
      api.subscriptions.list(
        {
          page: { number: 1, size: 1 },
          filter: { status: 'active' },
        },
        { signal }
      ),
  })

  const monthInvoices = useQuery({
    queryKey: queryKeys.invoices({ dashboard: 'month' }),
    queryFn: async ({ signal }) => {
      const invoices = []
      for await (const invoice of api.internal.invoices.listAll(
        { filter: { createdAt: { gte: monthStart() } } },
        { signal }
      )) {
        invoices.push(invoice)
      }
      return invoices
    },
  })

  const recentInvoices = useQuery({
    queryKey: queryKeys.invoices({ dashboard: 'recent' }),
    queryFn: ({ signal }) =>
      api.internal.invoices.list(
        {
          page: { number: 1, size: 10 },
          sort: { by: 'created_at', order: 'desc' },
        },
        { signal }
      ),
  })

  const monthTotalByCurrency = monthInvoices.data
    ? monthInvoices.data.reduce<Map<string, number>>((acc, invoice) => {
        const current = acc.get(invoice.currency) ?? 0
        acc.set(invoice.currency, current + Number(invoice.totals.total))
        return acc
      }, new Map())
    : undefined

  return {
    isLoading:
      customers.isLoading ||
      activeSubscriptions.isLoading ||
      monthInvoices.isLoading,
    customerCount: customers.data?.meta.page.total,
    activeSubscriptionCount: activeSubscriptions.data?.meta.page.total,
    monthInvoiceCount: monthInvoices.data?.length,
    monthTotalByCurrency,
    recentInvoices: recentInvoices.data,
    recentLoading: recentInvoices.isLoading,
  }
}

function StatCard({
  title,
  value,
  icon,
  isLoading,
}: {
  title: string
  value: React.ReactNode
  icon: React.ReactNode
  isLoading?: boolean
}) {
  return (
    <Card>
      <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
        <CardTitle className='text-sm font-medium text-muted-foreground'>
          {title}
        </CardTitle>
        {icon}
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className='h-8 w-24' />
        ) : (
          <div className='text-2xl font-semibold tabular-nums'>{value}</div>
        )}
      </CardContent>
    </Card>
  )
}

export function Dashboard() {
  const { t } = useTranslation()
  const stats = useDashboardStats()

  const monthTotalLabel = stats.monthTotalByCurrency
    ? [...stats.monthTotalByCurrency.entries()]
        .map(([currency, amount]) => formatAmount(amount, currency))
        .join(' · ')
    : undefined

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('dashboard.title')}
          description={t('dashboard.description')}
        />
        <div className='mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4'>
          <StatCard
            title={t('dashboard.stats.customers')}
            value={stats.customerCount?.toLocaleString()}
            icon={<Users className='size-4 text-muted-foreground' />}
            isLoading={stats.isLoading}
          />
          <StatCard
            title={t('dashboard.stats.activeSubscriptions')}
            value={stats.activeSubscriptionCount?.toLocaleString()}
            icon={<ReceiptText className='size-4 text-muted-foreground' />}
            isLoading={stats.isLoading}
          />
          <StatCard
            title={t('dashboard.stats.monthInvoices')}
            value={stats.monthInvoiceCount?.toLocaleString()}
            icon={<FileText className='size-4 text-muted-foreground' />}
            isLoading={stats.isLoading}
          />
          <StatCard
            title={t('dashboard.stats.monthTotal')}
            value={monthTotalLabel ?? '—'}
            icon={<Wallet className='size-4 text-muted-foreground' />}
            isLoading={stats.isLoading}
          />
        </div>

        <Card className='mt-6 py-0'>
          <CardHeader className='flex flex-row items-center justify-between'>
            <CardTitle className='text-base'>
              {t('dashboard.recentInvoices')}
            </CardTitle>
            <Link
              to='/invoices'
              className='flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground'
            >
              {t('common.viewAll')}
              <ArrowRight className='size-3.5' />
            </Link>
          </CardHeader>
          <CardContent className='px-0 pb-0'>
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('invoices.fields.number')}
                  </TableHead>
                  <TableHead>{t('invoices.fields.customer')}</TableHead>
                  <TableHead>{t('invoices.fields.total')}</TableHead>
                  <TableHead>{t('invoices.fields.status')}</TableHead>
                  <TableHead className='pr-6'>
                    {t('invoices.fields.createdAt')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stats.recentLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell colSpan={5} className='pl-6'>
                        <Skeleton className='h-5 w-full' />
                      </TableCell>
                    </TableRow>
                  ))
                ) : (stats.recentInvoices?.data.length ?? 0) === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('common.table.empty')}
                    </TableCell>
                  </TableRow>
                ) : (
                  stats.recentInvoices?.data.map((invoice) => (
                    <TableRow key={invoice.id}>
                      <TableCell className='pl-6 font-medium'>
                        <Link
                          to='/invoices/$invoiceId'
                          params={{ invoiceId: invoice.id }}
                          className='hover:underline'
                        >
                          {invoice.number}
                        </Link>
                      </TableCell>
                      <TableCell>{invoice.customer.name}</TableCell>
                      <TableCell className='tabular-nums'>
                        {formatAmount(invoice.totals.total, invoice.currency)}
                      </TableCell>
                      <TableCell>
                        <StatusBadge domain='invoice' value={invoice.status} />
                      </TableCell>
                      <TableCell className='pr-6 text-muted-foreground'>
                        {formatDateTime(invoice.createdAt)}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </Main>
    </>
  )
}
