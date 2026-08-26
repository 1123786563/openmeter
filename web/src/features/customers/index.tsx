import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { Customer } from '@openmeter/client'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCustomers } from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ServerTable } from '@/components/data-table/server-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { CustomerFormDialog } from './customer-form-dialog'

export function CustomersPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [formOpen, setFormOpen] = useState(false)

  const { data, isLoading, isFetching } = useCustomers({
    page,
    pageSize,
    search: search || undefined,
  })

  const columns: ColumnDef<Customer, unknown>[] = [
    {
      accessorKey: 'key',
      header: t('customers.fields.key'),
      cell: ({ row }) => (
        <Link
          to='/customers/$customerId'
          params={{ customerId: row.original.id }}
          className='font-medium hover:underline'
        >
          {row.original.key}
        </Link>
      ),
    },
    {
      accessorKey: 'name',
      header: t('customers.fields.name'),
    },
    {
      accessorKey: 'primaryEmail',
      header: t('customers.fields.primaryEmail'),
      cell: ({ row }) => row.original.primaryEmail ?? '—',
    },
    {
      accessorKey: 'createdAt',
      header: t('customers.fields.createdAt'),
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
          title={t('customers.title')}
          description={t('customers.description')}
          actions={
            <Button onClick={() => setFormOpen(true)}>
              <Plus className='size-4' />
              {t('customers.create')}
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
            <div className='flex items-center gap-2'>
              <Input
                placeholder={t('customers.searchPlaceholder')}
                className='h-8 w-62.5'
                value={searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    setPage(1)
                    setSearch(searchInput.trim())
                  }
                }}
              />
              <Button
                variant='outline'
                size='sm'
                className='h-8'
                onClick={() => {
                  setPage(1)
                  setSearch(searchInput.trim())
                }}
              >
                {t('common.search')}
              </Button>
            </div>
          }
          emptyMessage={t('customers.empty')}
        />
      </Main>
      <CustomerFormDialog open={formOpen} onOpenChange={setFormOpen} />
    </>
  )
}
