import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { Plan } from '@openmeter/client'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePlansPage, type PlanListParams } from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
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
import { StatusBadge } from '@/components/status-badge'
import { PlanFormWizard } from './plan-form-wizard'

const STATUS_OPTIONS = ['draft', 'active', 'scheduled', 'archived'] as const

export function PlansPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [status, setStatus] = useState<PlanListParams['status']>()
  const [createOpen, setCreateOpen] = useState(false)

  const { data, isLoading, isFetching } = usePlansPage({
    page,
    pageSize,
    status,
  })

  const columns: ColumnDef<Plan, unknown>[] = [
    {
      accessorKey: 'name',
      header: t('config.plans.fields.name'),
      cell: ({ row }) => (
        <Link
          to='/config/plans/$planId'
          params={{ planId: row.original.id }}
          className='font-medium hover:underline'
        >
          {row.original.name}
        </Link>
      ),
    },
    {
      accessorKey: 'key',
      header: t('config.plans.fields.key'),
      cell: ({ row }) => (
        <span className='font-mono text-xs text-muted-foreground'>
          {row.original.key}
        </span>
      ),
    },
    {
      accessorKey: 'version',
      header: t('config.plans.fields.version'),
      cell: ({ row }) => (
        <span className='tabular-nums'>v{row.original.version}</span>
      ),
    },
    {
      accessorKey: 'status',
      header: t('config.plans.fields.status'),
      cell: ({ row }) => (
        <StatusBadge domain='plan' value={row.original.status} />
      ),
    },
    {
      accessorKey: 'currency',
      header: t('config.plans.fields.currency'),
    },
    {
      accessorKey: 'billingCadence',
      header: t('config.plans.fields.billingCadence'),
      cell: ({ row }) => (
        <span className='text-muted-foreground'>
          {row.original.billingCadence}
        </span>
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('config.plans.fields.createdAt'),
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
          title={t('config.plans.title')}
          description={t('config.plans.description')}
          actions={
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className='size-4' />
              {t('config.plans.wizard.createTitle')}
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
                setStatus(
                  value === 'all'
                    ? undefined
                    : (value as PlanListParams['status'])
                )
              }}
            >
              <SelectTrigger className='h-8 w-40'>
                <SelectValue placeholder={t('config.plans.filter.status')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>
                  {t('config.plans.filter.allStatuses')}
                </SelectItem>
                {STATUS_OPTIONS.map((option) => (
                  <SelectItem key={option} value={option}>
                    {t(`plan.status.${option}`, { defaultValue: option })}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          }
          emptyMessage={t('config.plans.empty')}
        />
      </Main>
      <PlanFormWizard open={createOpen} onOpenChange={setCreateOpen} />
    </>
  )
}
