import { useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { Invoice, Subscription } from '@openmeter/client'
import { Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  useCreditBalance,
  useCreditTransactions,
  useCustomer,
  useCustomerWallet,
  useInvoices,
  useSubscriptions,
} from '@/api/hooks'
import { formatAmount, formatDateTime, formatNumber } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ServerTable } from '@/components/data-table/server-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { EnumBadge, StatusBadge } from '@/components/status-badge'
import { EntitlementsTable } from '@/features/credits/entitlements-table'
import { CustomerFormDialog } from './customer-form-dialog'

function InfoRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className='flex items-baseline justify-between gap-4 py-1.5'>
      <span className='text-sm text-muted-foreground'>{label}</span>
      <span className='text-sm font-medium break-all'>{children}</span>
    </div>
  )
}

function CustomerOverview({ customerId }: { customerId: string }) {
  const { t } = useTranslation()
  const { data: customer, isLoading } = useCustomer(customerId)
  const [editOpen, setEditOpen] = useState(false)

  if (isLoading) {
    return <Skeleton className='h-48 w-full' />
  }
  if (!customer) return null

  return (
    <Card>
      <CardHeader className='flex flex-row items-center justify-between'>
        <CardTitle className='text-base'>
          {t('customers.detail.info')}
        </CardTitle>
        <Button variant='outline' size='sm' onClick={() => setEditOpen(true)}>
          <Pencil className='size-3.5' />
          {t('common.edit')}
        </Button>
      </CardHeader>
      <Separator />
      <CardContent className='grid gap-x-10 py-4 sm:grid-cols-2 lg:grid-cols-3'>
        <InfoRow label={t('customers.fields.key')}>{customer.key}</InfoRow>
        <InfoRow label={t('customers.fields.name')}>{customer.name}</InfoRow>
        <InfoRow label={t('customers.fields.primaryEmail')}>
          {customer.primaryEmail ?? '—'}
        </InfoRow>
        <InfoRow label={t('customers.fields.currency')}>
          {customer.currency ?? '—'}
        </InfoRow>
        <InfoRow label={t('customers.fields.createdAt')}>
          {formatDateTime(customer.createdAt)}
        </InfoRow>
        <InfoRow label={t('customers.fields.description')}>
          {customer.description ?? '—'}
        </InfoRow>
      </CardContent>
      <CustomerFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        customer={customer}
      />
    </Card>
  )
}

function SubscriptionsTab({ customerId }: { customerId: string }) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const pageSize = 10
  const { data, isLoading } = useSubscriptions({ page, pageSize, customerId })

  const columns: ColumnDef<Subscription, unknown>[] = [
    {
      accessorKey: 'id',
      header: t('subscriptions.fields.id'),
      cell: ({ row }) => (
        <Link
          to='/subscriptions/$subscriptionId'
          params={{ subscriptionId: row.original.id }}
          className='font-mono text-xs hover:underline'
        >
          {row.original.id}
        </Link>
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
      accessorKey: 'settlementMode',
      header: t('subscriptions.fields.settlementMode'),
      cell: ({ row }) =>
        row.original.settlementMode ? (
          <span className='text-muted-foreground'>
            {t(`subscriptions.settlementMode.${row.original.settlementMode}`, {
              defaultValue: row.original.settlementMode.replace(/_/g, ' '),
            })}
          </span>
        ) : (
          '—'
        ),
    },
  ]

  return (
    <ServerTable
      columns={columns}
      data={data?.data ?? []}
      page={page}
      pageSize={pageSize}
      total={data?.meta.page.total}
      isLoading={isLoading}
      onPageChange={(next) => setPage(next.pageIndex + 1)}
      emptyMessage={t('subscriptions.empty')}
    />
  )
}

function InvoicesTab({ customerId }: { customerId: string }) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const pageSize = 10
  const { data, isLoading } = useInvoices({ page, pageSize, customerId })

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
    <ServerTable
      columns={columns}
      data={data?.data ?? []}
      page={page}
      pageSize={pageSize}
      total={data?.meta.page.total}
      isLoading={isLoading}
      onPageChange={(next) => setPage(next.pageIndex + 1)}
      emptyMessage={t('invoices.empty')}
    />
  )
}

function WalletTab({ customerId }: { customerId: string }) {
  const { t } = useTranslation()
  const { data: wallet, isLoading } = useCustomerWallet(customerId)

  if (isLoading) return <Skeleton className='h-48 w-full' />
  if (!wallet) return null

  return (
    <div className='space-y-4'>
      <div className='grid gap-4 sm:grid-cols-2'>
        <Card className='py-0'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium text-muted-foreground'>
              {t('commerce.wallet.totalAvailable')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-semibold tabular-nums'>
              {formatNumber(wallet.totalAvailableCredits)}
            </div>
          </CardContent>
        </Card>
        <Card className='py-0'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium text-muted-foreground'>
              {t('commerce.wallet.buckets')}
            </CardTitle>
          </CardHeader>
          <CardContent className='space-y-2 pt-0'>
            {wallet.buckets.map((bucket, index) => (
              <div
                key={index}
                className='flex items-center justify-between gap-2 text-sm'
              >
                <EnumBadge
                  domain='commerce'
                  kind='bucketSource'
                  value={bucket.source}
                />
                <span className='tabular-nums'>
                  {formatNumber(bucket.availableCredits)}
                  {bucket.expiresAt
                    ? ` · ${t('commerce.wallet.expiresAt')} ${formatDateTime(bucket.expiresAt)}`
                    : ''}
                </span>
              </div>
            ))}
            {wallet.buckets.length === 0 && (
              <p className='text-sm text-muted-foreground'>
                {t('commerce.wallet.empty')}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <Card className='py-0'>
        <CardHeader>
          <CardTitle className='text-base'>
            {t('commerce.wallet.transactions')}
          </CardTitle>
        </CardHeader>
        <Table>
          <TableHeader>
            <TableRow className='bg-hover/50'>
              <TableHead>{t('commerce.wallet.txKind')}</TableHead>
              <TableHead>{t('commerce.wallet.txAmount')}</TableHead>
              <TableHead>{t('commerce.wallet.txSource')}</TableHead>
              <TableHead>{t('commerce.wallet.txOccurredAt')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(wallet.transactions ?? []).map((tx) => (
              <TableRow key={tx.id}>
                <TableCell>
                  {t(`commerce.wallet.kind.${tx.kind}`, {
                    defaultValue: tx.kind,
                  })}
                </TableCell>
                <TableCell
                  className={
                    tx.amount >= 0n
                      ? 'tabular-nums'
                      : 'text-red-600 tabular-nums'
                  }
                >
                  {formatNumber(tx.amount)}
                </TableCell>
                <TableCell>
                  <EnumBadge
                    domain='commerce'
                    kind='bucketSource'
                    value={tx.provenance.source}
                  />
                </TableCell>
                <TableCell className='text-muted-foreground'>
                  {formatDateTime(tx.occurredAt)}
                </TableCell>
              </TableRow>
            ))}
            {(wallet.transactions ?? []).length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={4}
                  className='h-20 text-center text-muted-foreground'
                >
                  {t('commerce.wallet.empty')}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </Card>
    </div>
  )
}

function CreditsTab({ customerId }: { customerId: string }) {
  const { t } = useTranslation()
  const { data: balances, isLoading: balancesLoading } =
    useCreditBalance(customerId)
  const { data: transactions, isLoading: txLoading } =
    useCreditTransactions(customerId)

  return (
    <div className='space-y-4'>
      <div className='grid gap-4 sm:grid-cols-3'>
        {(balances?.balances ?? []).map((balance) => (
          <Card key={balance.currency} className='py-0'>
            <CardHeader className='pb-2'>
              <CardTitle className='text-sm font-medium text-muted-foreground'>
                {balance.currency}
              </CardTitle>
            </CardHeader>
            <CardContent className='space-y-1 text-sm'>
              <div className='flex justify-between'>
                <span className='text-muted-foreground'>
                  {t('credits.balance.live')}
                </span>
                <span className='tabular-nums'>
                  {formatNumber(balance.live)}
                </span>
              </div>
              <div className='flex justify-between'>
                <span className='text-muted-foreground'>
                  {t('credits.balance.settled')}
                </span>
                <span className='tabular-nums'>
                  {formatNumber(balance.settled)}
                </span>
              </div>
              <div className='flex justify-between'>
                <span className='text-muted-foreground'>
                  {t('credits.balance.pending')}
                </span>
                <span className='tabular-nums'>
                  {formatNumber(balance.pending)}
                </span>
              </div>
            </CardContent>
          </Card>
        ))}
        {balancesLoading &&
          Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className='h-32' />
          ))}
        {!balancesLoading && (balances?.balances.length ?? 0) === 0 && (
          <p className='col-span-3 py-6 text-center text-sm text-muted-foreground'>
            {t('credits.balance.empty')}
          </p>
        )}
      </div>

      <Card className='py-0'>
        <CardHeader>
          <CardTitle className='text-base'>
            {t('credits.transactions.title')}
          </CardTitle>
        </CardHeader>
        {txLoading ? (
          <CardContent className='space-y-2'>
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className='h-8 w-full' />
            ))}
          </CardContent>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className='bg-hover/50'>
                <TableHead>{t('credits.transactions.name')}</TableHead>
                <TableHead>{t('credits.transactions.type')}</TableHead>
                <TableHead>{t('credits.transactions.amount')}</TableHead>
                <TableHead>{t('credits.transactions.balanceAfter')}</TableHead>
                <TableHead>{t('credits.transactions.bookedAt')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(transactions?.data ?? []).map((tx) => (
                <TableRow key={tx.id}>
                  <TableCell>{tx.name}</TableCell>
                  <TableCell>
                    {t(`credits.transactions.kind.${tx.type}`, {
                      defaultValue: tx.type,
                    })}
                  </TableCell>
                  <TableCell className='tabular-nums'>
                    {formatAmount(tx.amount, tx.currency)}
                  </TableCell>
                  <TableCell className='tabular-nums'>
                    {formatNumber(tx.availableBalance.after)}
                  </TableCell>
                  <TableCell className='text-muted-foreground'>
                    {formatDateTime(tx.bookedAt)}
                  </TableCell>
                </TableRow>
              ))}
              {(transactions?.data ?? []).length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={5}
                    className='h-20 text-center text-muted-foreground'
                  >
                    {t('credits.transactions.empty')}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  )
}

export function CustomerDetail() {
  const { t } = useTranslation()
  const { customerId } = useParams({
    from: '/_authenticated/customers/$customerId',
  })
  const { data: customer } = useCustomer(customerId)

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Link to='/customers' className='hover:text-foreground'>
            {t('sidebar.customers')}
          </Link>
          <span>/</span>
          <span className='text-foreground'>{customer?.key ?? customerId}</span>
        </div>
        <h1 className='mt-2 text-2xl font-bold tracking-tight md:text-3xl'>
          {customer?.name ?? customerId}
        </h1>

        <div className='mt-6 space-y-6'>
          <CustomerOverview customerId={customerId} />
          <Tabs defaultValue='subscriptions'>
            <TabsList>
              <TabsTrigger value='subscriptions'>
                {t('customers.detail.tabs.subscriptions')}
              </TabsTrigger>
              <TabsTrigger value='invoices'>
                {t('customers.detail.tabs.invoices')}
              </TabsTrigger>
              <TabsTrigger value='wallet'>
                {t('customers.detail.tabs.wallet')}
              </TabsTrigger>
              <TabsTrigger value='credits'>
                {t('customers.detail.tabs.credits')}
              </TabsTrigger>
              <TabsTrigger value='entitlements'>
                {t('customers.detail.tabs.entitlements')}
              </TabsTrigger>
            </TabsList>
            <TabsContent value='subscriptions' className='mt-4'>
              <SubscriptionsTab customerId={customerId} />
            </TabsContent>
            <TabsContent value='invoices' className='mt-4'>
              <InvoicesTab customerId={customerId} />
            </TabsContent>
            <TabsContent value='wallet' className='mt-4'>
              <WalletTab customerId={customerId} />
            </TabsContent>
            <TabsContent value='credits' className='mt-4'>
              <CreditsTab customerId={customerId} />
            </TabsContent>
            <TabsContent value='entitlements' className='mt-4'>
              <Card className='py-0'>
                <EntitlementsTable customerId={customerId} />
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </Main>
    </>
  )
}
