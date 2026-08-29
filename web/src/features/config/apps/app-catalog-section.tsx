import { useState } from 'react'
import type { AppCatalogItem } from '@openmeter/client'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAppCatalog } from '@/api/hooks'
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
import { InstallAppDialog } from './install-app-dialog'

/**
 * Installable app catalog below the installed list. Entries offering only
 * OAuth install are shown but not installable (the API has no OAuth flow),
 * so their install button stays disabled with an explanatory note next to
 * it.
 */
export function AppCatalogSection() {
  const { t } = useTranslation()
  const { data, isLoading } = useAppCatalog()
  const items = data?.data ?? []

  const [installTarget, setInstallTarget] = useState<AppCatalogItem | null>(
    null
  )

  return (
    <div className='mt-6'>
      <h2 className='mb-3 text-sm font-medium text-muted-foreground'>
        {t('config.apps.catalog.title')}
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
              <TableHead>{t('config.apps.fields.capabilities')}</TableHead>
              <TableHead>
                {t('config.apps.catalog.installMethodsLabel')}
              </TableHead>
              <TableHead className='pr-6 text-right'>
                {t('config.apps.fields.actions')}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((item) => {
              const installable =
                item.installMethods.includes('with_api_key') ||
                item.installMethods.includes('no_credentials_required')
              return (
                <TableRow key={item.type}>
                  <TableCell className='pl-6'>
                    <div className='font-medium'>{item.name}</div>
                    <div className='text-xs text-muted-foreground'>
                      {item.description}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant='outline'>
                      {t(`config.apps.type.${item.type}`)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className='flex flex-wrap gap-1'>
                      {item.capabilities.map((capability) => (
                        <Badge
                          key={capability.key}
                          variant='secondary'
                          className='font-normal'
                        >
                          {t(`config.apps.capability.${capability.type}`)}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className='flex flex-wrap gap-1'>
                      {item.installMethods.map((method) => (
                        <Badge
                          key={method}
                          variant='outline'
                          className='font-normal'
                        >
                          {t(`config.apps.catalog.installMethod.${method}`)}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className='pr-6'>
                    <div className='flex items-center justify-end gap-2'>
                      {!installable && (
                        <span className='text-xs text-muted-foreground'>
                          {t('config.apps.catalog.oauthUnsupported')}
                        </span>
                      )}
                      <Button
                        variant='ghost'
                        size='sm'
                        className='h-7 px-2'
                        disabled={!installable}
                        onClick={() => setInstallTarget(item)}
                      >
                        <Plus className='size-4' />
                        {t('config.apps.catalog.installAction')}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
            {items.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={5}
                  className='h-24 text-center text-muted-foreground'
                >
                  {t('config.apps.catalog.empty')}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      )}
      {/* The per-type key remounts the dialog so its zod resolver (API key
          required only for Stripe) matches the selected catalog item. */}
      <InstallAppDialog
        key={installTarget?.type ?? 'none'}
        open={Boolean(installTarget)}
        onOpenChange={(open) => !open && setInstallTarget(null)}
        item={installTarget}
      />
    </div>
  )
}
