import { useMemo, useState } from 'react'
import type { Profile } from '@openmeter/client'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useApps,
  useBillingProfiles,
  useDeleteBillingProfile,
} from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
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
import { BillingProfileFormDialog } from './billing-profile-form-dialog'

/** Profile.apps carries ids only; resolve display names from the installed-apps list. */
function useAppNameMap() {
  const { data: appsData } = useApps()
  return useMemo(() => {
    const map = new Map<string, string>()
    for (const app of appsData?.data ?? []) {
      map.set(app.id, app.name)
    }
    return map
  }, [appsData])
}

export function BillingProfilesPage() {
  const { t } = useTranslation()
  const { data, isLoading } = useBillingProfiles()
  const appNameMap = useAppNameMap()
  const deleteMutation = useDeleteBillingProfile()
  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Profile | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Profile | null>(null)

  const appName = (id: string) => appNameMap.get(id) ?? id

  const renderProfile = (profile: Profile) => (
    <TableRow key={profile.id}>
      <TableCell className='pl-6 font-medium'>
        {profile.name}
        {profile.description ? (
          <span className='block text-xs text-muted-foreground'>
            {profile.description}
          </span>
        ) : null}
      </TableCell>
      <TableCell>
        {profile.supplier.name ?? '—'}
        {profile.supplier.taxId?.code ? (
          <span className='block text-xs text-muted-foreground'>
            {t('config.billingProfiles.fields.supplierTaxId')}：
            {profile.supplier.taxId.code}
          </span>
        ) : null}
      </TableCell>
      <TableCell className='text-xs'>
        <span className='block'>
          {t('config.billingProfiles.fields.appTax')}：
          {appName(profile.apps.tax.id)}
        </span>
        <span className='block'>
          {t('config.billingProfiles.fields.appInvoicing')}：
          {appName(profile.apps.invoicing.id)}
        </span>
        <span className='block'>
          {t('config.billingProfiles.fields.appPayment')}：
          {appName(profile.apps.payment.id)}
        </span>
      </TableCell>
      <TableCell>
        {profile.default ? (
          <Badge
            variant='outline'
            className='border-emerald-200 bg-emerald-50 px-1.5 font-normal text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300'
          >
            {t('config.billingProfiles.list.default')}
          </Badge>
        ) : (
          <span className='text-muted-foreground'>—</span>
        )}
      </TableCell>
      <TableCell className='text-muted-foreground'>
        {formatDateTime(profile.createdAt)}
      </TableCell>
      <TableCell className='pr-6'>
        <div className='flex justify-end gap-1'>
          <Button
            variant='ghost'
            size='sm'
            className='h-7 px-2'
            onClick={() => setEditTarget(profile)}
          >
            <Pencil className='size-4' />
            {t('config.billingProfiles.edit')}
          </Button>
          <Button
            variant='ghost'
            size='sm'
            className='h-7 px-2 text-destructive hover:text-destructive'
            onClick={() => setDeleteTarget(profile)}
          >
            <Trash2 className='size-4' />
            {t('config.billingProfiles.delete')}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.billingProfiles.title')}
          description={t('config.billingProfiles.description')}
          actions={
            <Button onClick={() => setFormOpen(true)}>
              <Plus className='size-4' />
              {t('config.billingProfiles.create')}
            </Button>
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
                    {t('config.billingProfiles.fields.name')}
                  </TableHead>
                  <TableHead>
                    {t('config.billingProfiles.fields.supplier')}
                  </TableHead>
                  <TableHead>
                    {t('config.billingProfiles.fields.apps')}
                  </TableHead>
                  <TableHead>
                    {t('config.billingProfiles.fields.default')}
                  </TableHead>
                  <TableHead>
                    {t('config.billingProfiles.list.createdAt')}
                  </TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('config.billingProfiles.list.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(data?.data ?? []).map(renderProfile)}
                {(data?.data ?? []).length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.billingProfiles.list.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </Main>
      <BillingProfileFormDialog
        open={formOpen || Boolean(editTarget)}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditTarget(null)
        }}
        profile={editTarget}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('config.billingProfiles.deleteConfirm.title')}
        desc={t('config.billingProfiles.deleteConfirm.description', {
          name: deleteTarget?.name ?? '',
        })}
        confirmText={t('config.billingProfiles.delete')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (!deleteTarget) return
          deleteMutation.mutate(
            { id: deleteTarget.id },
            {
              onSuccess: () => {
                toast.success(t('config.billingProfiles.toast.deleted'))
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
