import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import type { PlanAddon } from '@openmeter/client'
import { ExternalLink, Pencil, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAddons, useDeletePlanAddon, usePlanAddons } from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { ServerTable } from '@/components/data-table/server-table'
import { PlanAddonFormDialog } from './plan-addon-form-dialog'

/**
 * Add-ons purchasable within one plan. Prices live on the addon itself
 * (no price fields on PlanAddon per spec) — editing price happens on the
 * addons page, linked from each row.
 */
export function PlanAddonsTab({ planId }: { planId: string }) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<PlanAddon | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<PlanAddon | null>(null)

  const { data, isLoading, isFetching } = usePlanAddons(planId, {
    page,
    pageSize,
  })
  const { data: addonsData } = useAddons()
  const addonNameById = new Map(
    (addonsData?.data ?? []).map((addon) => [addon.id, addon])
  )
  const deleteMutation = useDeletePlanAddon(planId)

  const columns: ColumnDef<PlanAddon, unknown>[] = [
    {
      accessorKey: 'name',
      header: t('config.planAddons.fields.name'),
      cell: ({ row }) => (
        <span className='font-medium'>{row.original.name}</span>
      ),
    },
    {
      accessorKey: 'addon',
      header: t('config.planAddons.fields.addon'),
      cell: ({ row }) => {
        const addon = addonNameById.get(row.original.addon.id)
        return addon ? (
          <Link
            to='/config/addons'
            className='hover:underline'
            title={t('config.planAddons.editPriceHint')}
          >
            {addon.name} ({addon.key})
            <ExternalLink className='ml-1 inline size-3' />
          </Link>
        ) : (
          <code className='font-mono text-xs text-muted-foreground'>
            {row.original.addon.id}
          </code>
        )
      },
    },
    {
      accessorKey: 'fromPlanPhase',
      header: t('config.planAddons.fields.fromPlanPhase'),
      cell: ({ row }) => (
        <code className='font-mono text-xs'>{row.original.fromPlanPhase}</code>
      ),
    },
    {
      accessorKey: 'maxQuantity',
      header: t('config.planAddons.fields.maxQuantity'),
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {row.original.maxQuantity ?? t('config.planAddons.unlimited')}
        </span>
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('config.planAddons.fields.createdAt'),
      cell: ({ row }) => (
        <span className='text-muted-foreground tabular-nums'>
          {formatDateTime(row.original.createdAt)}
        </span>
      ),
    },
    {
      id: 'actions',
      header: () => (
        <span className='sr-only'>{t('config.planAddons.actions')}</span>
      ),
      cell: ({ row }) => (
        <div className='flex justify-end gap-1'>
          <Button
            variant='ghost'
            size='sm'
            className='h-7 px-2'
            onClick={() => {
              setEditTarget(row.original)
              setFormOpen(true)
            }}
          >
            <Pencil className='size-4' />
            {t('common.edit')}
          </Button>
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
    <div className='space-y-4'>
      <div className='flex justify-end'>
        <Button
          size='sm'
          onClick={() => {
            setEditTarget(null)
            setFormOpen(true)
          }}
        >
          <Plus className='size-4' />
          {t('config.planAddons.create')}
        </Button>
      </div>
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
        emptyMessage={t('config.planAddons.empty')}
      />

      <PlanAddonFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        planId={planId}
        planAddon={editTarget ?? undefined}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('config.planAddons.deleteConfirm.title')}
        desc={t('config.planAddons.deleteConfirm.description', {
          name: deleteTarget?.name ?? '',
        })}
        confirmText={t('common.delete')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (!deleteTarget) return
          deleteMutation.mutate(
            { planAddonId: deleteTarget.id },
            {
              onSuccess: () => {
                toast.success(t('config.planAddons.toast.deleted'))
                setDeleteTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </div>
  )
}
