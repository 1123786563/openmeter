import { useState } from 'react'
import type { TaxCode } from '@openmeter/client'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useDeleteTaxCode, useTaxCodes } from '@/api/hooks'
import { handleServerError } from '@/lib/handle-server-error'
import { formatShortDateTime } from '@/lib/format'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
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
import { Label } from '@/components/ui/label'
import { TaxCodeFormDialog } from './tax-code-form-dialog'

/**
 * Tax code administration: list (optionally including deleted codes),
 * create/edit (key immutable after creation) and delete. App mappings bind
 * the internal tax code to per-app provider codes.
 */
export function TaxCodesPage() {
  const { t } = useTranslation()
  const [includeDeleted, setIncludeDeleted] = useState(false)
  const { data, isLoading } = useTaxCodes(includeDeleted)
  const taxCodes = data?.data ?? []

  const deleteMutation = useDeleteTaxCode()

  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<TaxCode | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<TaxCode | null>(null)

  const openCreate = () => {
    setEditTarget(null)
    setFormOpen(true)
  }

  const openEdit = (taxCode: TaxCode) => {
    setEditTarget(taxCode)
    setFormOpen(true)
  }

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.taxCodes.title')}
          description={t('config.taxCodes.description')}
          actions={
            <div className='flex items-center gap-4'>
              <div className='flex items-center gap-2'>
                <Switch
                  id='tax-codes-include-deleted'
                  checked={includeDeleted}
                  onCheckedChange={setIncludeDeleted}
                />
                <Label htmlFor='tax-codes-include-deleted'>
                  {t('config.taxCodes.includeDeleted')}
                </Label>
              </div>
              <Button onClick={openCreate}>
                <Plus className='size-4' />
                {t('config.taxCodes.create')}
              </Button>
            </div>
          }
        />
        <div className='mt-6'>
          {isLoading ? (
            <div className='space-y-2'>
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className='h-10 w-full' />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('config.taxCodes.fields.key')}
                  </TableHead>
                  <TableHead>{t('config.taxCodes.fields.name')}</TableHead>
                  <TableHead>
                    {t('config.taxCodes.fields.appMappings')}
                  </TableHead>
                  <TableHead>{t('config.taxCodes.updatedAt')}</TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('config.taxCodes.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {taxCodes.map((taxCode) => (
                  <TableRow key={taxCode.id}>
                    <TableCell className='pl-6 font-mono'>
                      {taxCode.key}
                    </TableCell>
                    <TableCell className='font-medium'>
                      {taxCode.name}
                      {taxCode.deletedAt && (
                        <Badge variant='destructive' className='ml-2'>
                          {t('config.taxCodes.deleted')}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className='flex flex-wrap gap-1'>
                        {taxCode.appMappings.map((mapping) => (
                          <Badge key={mapping.appType} variant='outline'>
                            {t(`config.taxCodes.appType.${mapping.appType}`)}
                            : {mapping.taxCode}
                          </Badge>
                        ))}
                        {taxCode.appMappings.length === 0 && (
                          <span className='text-muted-foreground'>—</span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className='text-muted-foreground'>
                      {formatShortDateTime(taxCode.updatedAt)}
                    </TableCell>
                    <TableCell className='pr-6'>
                      <div className='flex justify-end gap-1'>
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2'
                          disabled={Boolean(taxCode.deletedAt)}
                          onClick={() => openEdit(taxCode)}
                        >
                          <Pencil className='size-4' />
                          {t('common.edit')}
                        </Button>
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2 text-destructive hover:text-destructive'
                          disabled={Boolean(taxCode.deletedAt)}
                          onClick={() => setDeleteTarget(taxCode)}
                        >
                          <Trash2 className='size-4' />
                          {t('config.taxCodes.delete')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {taxCodes.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.taxCodes.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </Main>

      <TaxCodeFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        taxCode={editTarget ?? undefined}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('config.taxCodes.deleteConfirm.title')}
        desc={t('config.taxCodes.deleteConfirm.description', {
          name: deleteTarget?.name ?? '',
        })}
        confirmText={t('config.taxCodes.delete')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (!deleteTarget) return
          deleteMutation.mutate(
            { taxCodeId: deleteTarget.id },
            {
              onSuccess: () => {
                toast.success(t('config.taxCodes.toast.deleted'))
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
