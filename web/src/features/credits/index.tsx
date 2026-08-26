import { useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import type { CreditGrant, Customer } from '@openmeter/client'
import { Coins, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCreditBalance, useCreditGrants } from '@/api/hooks'
import { formatDateTime, formatNumber } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ServerTable } from '@/components/data-table/server-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { StatusBadge } from '@/components/status-badge'
import { CustomerPicker } from '@/features/customers/customer-picker'
import { EntitlementsTable } from './entitlements-table'
import { GrantFormDialog } from './grant-form-dialog'

export function CreditsPage() {
  const { t } = useTranslation()
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [grantOpen, setGrantOpen] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 10

  const { data: balances, isLoading: balancesLoading } = useCreditBalance(
    customer?.id ?? ''
  )
  const { data: grants, isLoading: grantsLoading } = useCreditGrants({
    customerId: customer?.id ?? '',
    page,
    pageSize,
  })

  const columns: ColumnDef<CreditGrant, unknown>[] = [
    {
      accessorKey: 'name',
      header: t('credits.grants.name'),
      cell: ({ row }) => (
        <div className='min-w-0'>
          <div className='truncate font-medium'>{row.original.name}</div>
          <div className='text-xs text-muted-foreground'>
            {row.original.description ?? ''}
          </div>
        </div>
      ),
    },
    {
      accessorKey: 'amount',
      header: t('credits.grants.amount'),
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatNumber(row.original.amount)} {row.original.currency}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: t('credits.grants.status'),
      cell: ({ row }) => {
        const grant = row.original
        const status = grant.voidedAt
          ? 'voided'
          : grant.expiresAt && grant.expiresAt < new Date()
            ? 'expired'
            : 'active'
        return <StatusBadge domain='grant' value={status} />
      },
    },
    {
      accessorKey: 'priority',
      header: t('credits.grants.priority'),
    },
    {
      accessorKey: 'effectiveAt',
      header: t('credits.grants.effectiveAt'),
      cell: ({ row }) => (
        <span className='text-muted-foreground'>
          {formatDateTime(row.original.effectiveAt)}
        </span>
      ),
    },
    {
      accessorKey: 'expiresAt',
      header: t('credits.grants.expiresAt'),
      cell: ({ row }) => (
        <span className='text-muted-foreground'>
          {formatDateTime(row.original.expiresAt)}
        </span>
      ),
    },
  ]

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('credits.title')}
          description={t('credits.description')}
          actions={
            customer ? (
              <Button onClick={() => setGrantOpen(true)}>
                <Plus className='size-4' />
                {t('credits.grantForm.title')}
              </Button>
            ) : undefined
          }
        />

        <div className='mt-6 max-w-md'>
          <CustomerPicker value={customer} onChange={setCustomer} />
        </div>

        {customer ? (
          <div className='mt-6 space-y-6'>
            <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
              {balancesLoading &&
                Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className='h-28' />
                ))}
              {(balances?.balances ?? []).map((balance) => (
                <Card key={balance.currency} className='py-0'>
                  <CardHeader className='flex flex-row items-center justify-between pb-2'>
                    <CardTitle className='flex items-center gap-2 text-sm font-medium text-muted-foreground'>
                      <Coins className='size-4' />
                      {balance.currency}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className='text-2xl font-semibold tabular-nums'>
                      {formatNumber(balance.live)}
                    </div>
                    <p className='mt-1 text-xs text-muted-foreground'>
                      {t('credits.balance.settled')}:{' '}
                      {formatNumber(balance.settled)} ·{' '}
                      {t('credits.balance.pending')}:{' '}
                      {formatNumber(balance.pending)}
                    </p>
                  </CardContent>
                </Card>
              ))}
              {!balancesLoading && (balances?.balances.length ?? 0) === 0 && (
                <p className='col-span-3 py-6 text-center text-sm text-muted-foreground'>
                  {t('credits.balance.empty')}
                </p>
              )}
            </div>

            <div>
              <h2 className='mb-3 text-base font-semibold'>
                {t('credits.grants.title')}
              </h2>
              <ServerTable
                columns={columns}
                data={grants?.data ?? []}
                page={page}
                pageSize={pageSize}
                total={grants?.meta.page.total}
                isLoading={grantsLoading}
                onPageChange={(next) => setPage(next.pageIndex + 1)}
                emptyMessage={t('credits.grants.empty')}
              />
            </div>

            <div>
              <h2 className='mb-3 text-base font-semibold'>
                {t('credits.entitlements.title')}
              </h2>
              <Card className='py-0'>
                <EntitlementsTable customerId={customer.id} />
              </Card>
            </div>
          </div>
        ) : (
          <p className='mt-12 text-center text-sm text-muted-foreground'>
            {t('credits.selectCustomer')}
          </p>
        )}
      </Main>

      {customer && (
        <GrantFormDialog
          open={grantOpen}
          onOpenChange={setGrantOpen}
          customerId={customer.id}
        />
      )}
    </>
  )
}
