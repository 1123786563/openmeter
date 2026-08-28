import { useState } from 'react'
import type { App, AppStripe } from '@openmeter/client'
import { KeyRound, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useApps, useUninstallApp } from '@/api/hooks'
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
import { StripeKeyDialog } from './stripe-key-dialog'

/**
 * Installed apps administration. The GET /apps response carries the catalog
 * definition (capabilities) per app but not default_for_capability_types —
 * that field only exists on the install response and is surfaced by the
 * install flow. Uninstall requires confirmation; Stripe apps expose a
 * replace-API-key dialog (the API has no OAuth flow).
 */
export function AppsPage() {
  const { t } = useTranslation()
  const { data, isLoading } = useApps()
  const apps = data?.data ?? []

  const uninstallMutation = useUninstallApp()

  const [keyTarget, setKeyTarget] = useState<AppStripe | null>(null)
  const [uninstallTarget, setUninstallTarget] = useState<App | null>(null)

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.apps.title')}
          description={t('config.apps.description')}
        />
        <div className='mt-6'>
          <h2 className='mb-3 text-sm font-medium text-muted-foreground'>
            {t('config.apps.installed.title')}
          </h2>
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
                    {t('config.apps.fields.name')}
                  </TableHead>
                  <TableHead>{t('config.apps.fields.type')}</TableHead>
                  <TableHead>{t('config.apps.fields.status')}</TableHead>
                  <TableHead>{t('config.apps.fields.capabilities')}</TableHead>
                  <TableHead>{t('config.apps.fields.stripeInfo')}</TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('config.apps.fields.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apps.map((app) => (
                  <TableRow key={app.id}>
                    <TableCell className='pl-6 font-medium'>
                      {app.name}
                    </TableCell>
                    <TableCell>
                      <Badge variant='outline'>
                        {t(`config.apps.type.${app.type}`)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant='outline'
                        className={
                          app.status === 'ready'
                            ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300'
                            : 'border-destructive/50 bg-destructive/10 text-destructive'
                        }
                      >
                        {t(`config.apps.status.${app.status}`)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className='flex flex-wrap gap-1'>
                        {app.definition.capabilities.map((capability) => (
                          <Badge
                            key={capability.key}
                            variant='secondary'
                            className='font-normal'
                          >
                            {t(
                              `config.apps.capability.${capability.type}`
                            )}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className='text-muted-foreground'>
                      {app.type === 'stripe' ? (
                        <div className='space-y-0.5 text-xs'>
                          <div className='font-mono'>{app.accountId}</div>
                          <div className='font-mono'>{app.maskedApiKey}</div>
                          <Badge variant='outline'>
                            {app.livemode
                              ? t('config.apps.stripe.livemode')
                              : t('config.apps.stripe.testmode')}
                          </Badge>
                        </div>
                      ) : (
                        '—'
                      )}
                    </TableCell>
                    <TableCell className='pr-6'>
                      <div className='flex justify-end gap-1'>
                        {app.type === 'stripe' && (
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-7 px-2'
                            onClick={() => setKeyTarget(app)}
                          >
                            <KeyRound className='size-4' />
                            {t('config.apps.stripeKey.open')}
                          </Button>
                        )}
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2 text-destructive hover:text-destructive'
                          onClick={() => setUninstallTarget(app)}
                        >
                          <Trash2 className='size-4' />
                          {t('config.apps.uninstall')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {apps.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.apps.installed.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </Main>

      <StripeKeyDialog
        open={Boolean(keyTarget)}
        onOpenChange={(open) => !open && setKeyTarget(null)}
        app={keyTarget}
      />

      <ConfirmDialog
        open={Boolean(uninstallTarget)}
        onOpenChange={(open) => !open && setUninstallTarget(null)}
        title={t('config.apps.uninstallConfirm.title')}
        desc={t('config.apps.uninstallConfirm.description', {
          name: uninstallTarget?.name ?? '',
          type: uninstallTarget
            ? t(`config.apps.type.${uninstallTarget.type}`)
            : '',
        })}
        confirmText={t('config.apps.uninstall')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={uninstallMutation.isPending}
        handleConfirm={() => {
          if (!uninstallTarget) return
          uninstallMutation.mutate(
            { appId: uninstallTarget.id },
            {
              onSuccess: () => {
                toast.success(t('config.apps.toast.uninstalled'))
                setUninstallTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </>
  )
}
