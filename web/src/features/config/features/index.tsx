import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { Feature } from '@openmeter/client'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useDeleteFeature, useFeatures } from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { ServerTable } from '@/components/data-table/server-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { FeatureFormDialog } from './feature-form-dialog'

export function FeaturesPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')

  const [formOpen, setFormOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Feature | null>(null)

  const { data, isLoading, isFetching } = useFeatures({
    page,
    pageSize,
    search: search || undefined,
  })
  const deleteMutation = useDeleteFeature()

  const columns: ColumnDef<Feature, unknown>[] = [
    {
      accessorKey: 'key',
      header: t('config.features.fields.key'),
      cell: ({ row }) => (
        <Link
          to='/config/features/$featureId'
          params={{ featureId: row.original.id }}
          className='font-mono text-xs hover:underline'
        >
          {row.original.key}
        </Link>
      ),
    },
    {
      accessorKey: 'name',
      header: t('config.features.fields.name'),
      cell: ({ row }) => (
        <span className='font-medium'>{row.original.name}</span>
      ),
    },
    {
      accessorKey: 'description',
      header: t('config.features.fields.description'),
      cell: ({ row }) => (
        <span className='text-muted-foreground'>
          {row.original.description || '—'}
        </span>
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('config.features.fields.createdAt'),
      cell: ({ row }) => (
        <span className='text-muted-foreground tabular-nums'>
          {formatDateTime(row.original.createdAt)}
        </span>
      ),
    },
    {
      id: 'actions',
      header: () => (
        <span className='sr-only'>{t('config.features.actions')}</span>
      ),
      cell: ({ row }) => (
        <div className='flex justify-end'>
          <Button
            variant='ghost'
            size='sm'
            className='h-7 px-2 text-destructive hover:text-destructive'
            onClick={() => setDeleteTarget(row.original)}
          >
            <Trash2 className='size-4' />
            {t('common.delete')}
          </Button>
        </div>
      ),
    },
  ]

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.features.title')}
          description={t('config.features.description')}
          actions={
            <Button onClick={() => setFormOpen(true)}>
              <Plus className='size-4' />
              {t('config.features.create')}
            </Button>
          }
        />
        <div className='mt-6'>
          <ServerTable
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
                  placeholder={t('config.features.searchPlaceholder')}
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
            emptyMessage={t('config.features.empty')}
          />
        </div>
      </Main>

      <FeatureFormDialog open={formOpen} onOpenChange={setFormOpen} />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('config.features.deleteConfirm.title')}
        desc={t('config.features.deleteConfirm.description', {
          name: deleteTarget?.name ?? '',
          key: deleteTarget?.key ?? '',
        })}
        confirmText={t('common.delete')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (!deleteTarget) return
          deleteMutation.mutate(
            { featureId: deleteTarget.id },
            {
              onSuccess: () => {
                toast.success(t('config.features.toast.deleted'))
                setDeleteTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </>
  )
}
