import { useState } from 'react'
import type { Addon } from '@openmeter/client'
import { Archive, Pencil, Plus, Rocket, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useAddons,
  useArchiveAddon,
  useDeleteAddon,
  usePublishAddon,
} from '@/api/hooks'
import { formatShortDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { AddonFormDialog } from './addon-form-dialog'

/**
 * Standalone add-on administration: list with lifecycle actions
 * (publish draft → active, archive, delete) and a create/edit dialog whose
 * rate cards reuse the plans pricing contract. `key` and `currency` are
 * immutable after creation, so editing locks both fields.
 */
export function AddonsPage() {
  const { t } = useTranslation()
  const { data, isLoading } = useAddons()
  const addons = data?.data ?? []

  const [editTarget, setEditTarget] = useState<Addon | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [publishTarget, setPublishTarget] = useState<Addon | null>(null)
  const [archiveTarget, setArchiveTarget] = useState<Addon | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Addon | null>(null)

  const publishMutation = usePublishAddon()
  const archiveMutation = useArchiveAddon()
  const deleteMutation = useDeleteAddon()

  const openCreate = () => {
    setEditTarget(null)
    setDialogOpen(true)
  }

  const openEdit = (addon: Addon) => {
    setEditTarget(addon)
    setDialogOpen(true)
  }

  const statusVariant = (status: Addon['status']) =>
    status === 'active'
      ? 'default'
      : status === 'draft'
        ? 'secondary'
        : 'outline'

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.addons.title')}
          description={t('config.addons.description')}
          actions={
            <Button onClick={openCreate}>
              <Plus className='size-4' />
              {t('config.addons.create')}
            </Button>
          }
        />
        <div className='mt-6'>
          {isLoading ? (
            <div className='space-y-2'>
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className='h-10 w-full' />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('config.addons.name')}
                  </TableHead>
                  <TableHead>{t('config.addons.key')}</TableHead>
                  <TableHead>{t('config.addons.version')}</TableHead>
                  <TableHead>{t('config.addons.instanceType')}</TableHead>
                  <TableHead>{t('config.addons.currency')}</TableHead>
                  <TableHead>{t('config.addons.status')}</TableHead>
                  <TableHead>{t('config.addons.rateCards')}</TableHead>
                  <TableHead>{t('config.addons.createdAt')}</TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('common.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {addons.map((addon) => (
                  <TableRow key={addon.id}>
                    <TableCell className='pl-6 font-medium'>
                      {addon.name}
                    </TableCell>
                    <TableCell className='font-mono text-xs'>
                      {addon.key}
                    </TableCell>
                    <TableCell className='tabular-nums'>
                      v{addon.version}
                    </TableCell>
                    <TableCell>
                      {t(`config.addons.instanceTypes.${addon.instanceType}`)}
                    </TableCell>
                    <TableCell className='font-mono text-xs'>
                      {addon.currency}
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusVariant(addon.status)}>
                        {t(`config.addons.statuses.${addon.status}`)}
                      </Badge>
                    </TableCell>
                    <TableCell className='tabular-nums'>
                      {addon.rateCards.length}
                    </TableCell>
                    <TableCell className='text-muted-foreground'>
                      {formatShortDateTime(addon.createdAt)}
                    </TableCell>
                    <TableCell className='pr-6'>
                      <div className='flex justify-end gap-1'>
                        <Button
                          variant='ghost'
                          size='icon'
                          className='size-8'
                          title={t('config.addons.edit')}
                          onClick={() => openEdit(addon)}
                        >
                          <Pencil className='size-4' />
                        </Button>
                        {addon.status === 'draft' && (
                          <Button
                            variant='ghost'
                            size='icon'
                            className='size-8'
                            title={t('config.addons.publish')}
                            onClick={() => setPublishTarget(addon)}
                          >
                            <Rocket className='size-4' />
                          </Button>
                        )}
                        {addon.status !== 'archived' && (
                          <Button
                            variant='ghost'
                            size='icon'
                            className='size-8'
                            title={t('config.addons.archive')}
                            onClick={() => setArchiveTarget(addon)}
                          >
                            <Archive className='size-4' />
                          </Button>
                        )}
                        <Button
                          variant='ghost'
                          size='icon'
                          className='size-8 text-destructive'
                          title={t('config.addons.delete')}
                          onClick={() => setDeleteTarget(addon)}
                        >
                          <Trash2 className='size-4' />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {addons.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={9}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.addons.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </Main>

      <AddonFormDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        addon={editTarget}
      />

      <ConfirmDialog
        open={Boolean(publishTarget)}
        onOpenChange={(open) => !open && setPublishTarget(null)}
        title={t('config.addons.publishConfirm.title')}
        desc={t('config.addons.publishConfirm.description', {
          name: publishTarget?.name ?? '',
        })}
        confirmText={t('config.addons.publish')}
        cancelBtnText={t('common.cancel')}
        isLoading={publishMutation.isPending}
        handleConfirm={() =>
          publishMutation.mutate(
            { addonId: publishTarget?.id ?? '' },
            {
              onSuccess: () => {
                toast.success(t('config.addons.toast.published'))
                setPublishTarget(null)
              },
              onError: handleServerError,
            }
          )
        }
      />
      <ConfirmDialog
        open={Boolean(archiveTarget)}
        onOpenChange={(open) => !open && setArchiveTarget(null)}
        title={t('config.addons.archiveConfirm.title')}
        desc={t('config.addons.archiveConfirm.description', {
          name: archiveTarget?.name ?? '',
        })}
        confirmText={t('config.addons.archive')}
        cancelBtnText={t('common.cancel')}
        isLoading={archiveMutation.isPending}
        handleConfirm={() =>
          archiveMutation.mutate(
            { addonId: archiveTarget?.id ?? '' },
            {
              onSuccess: () => {
                toast.success(t('config.addons.toast.archived'))
                setArchiveTarget(null)
              },
              onError: handleServerError,
            }
          )
        }
      />
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('config.addons.deleteConfirm.title')}
        desc={t('config.addons.deleteConfirm.description', {
          name: deleteTarget?.name ?? '',
        })}
        confirmText={t('config.addons.delete')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() =>
          deleteMutation.mutate(
            { addonId: deleteTarget?.id ?? '' },
            {
              onSuccess: () => {
                toast.success(t('config.addons.toast.deleted'))
                setDeleteTarget(null)
              },
              onError: handleServerError,
            }
          )
        }
      />
    </>
  )
}
