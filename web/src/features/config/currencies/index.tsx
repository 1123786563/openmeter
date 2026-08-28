import { useState } from 'react'
import type { CurrencyCustom } from '@openmeter/client'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCurrencies, useFiatCurrencies } from '@/api/hooks'
import { formatShortDateTime } from '@/lib/format'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { CustomCurrencyDialog } from './custom-currency-dialog'

/**
 * Currency administration: read-only fiat list (v1 info endpoint) and the
 * custom currency catalog (v3 filter[type]=custom + expand=cost_basis).
 * Custom currencies have no update/delete endpoint, so the page states that
 * creation is permanent.
 */
export function CurrenciesPage() {
  const { t } = useTranslation()
  const { data: fiat, isLoading: fiatLoading } = useFiatCurrencies()
  const { data, isLoading: customLoading } = useCurrencies({
    type: 'custom',
    expandCostBasis: true,
  })
  const customCurrencies = (data?.data ?? []).filter(
    (currency): currency is CurrencyCustom => currency.type === 'custom'
  )

  const [createOpen, setCreateOpen] = useState(false)

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.currencies.title')}
          description={t('config.currencies.description')}
        />
        <div className='mt-6'>
          <Tabs defaultValue='fiat'>
            <TabsList>
              <TabsTrigger value='fiat'>
                {t('config.currencies.tabs.fiat')}
              </TabsTrigger>
              <TabsTrigger value='custom'>
                {t('config.currencies.tabs.custom')}
              </TabsTrigger>
            </TabsList>

            <TabsContent value='fiat' className='mt-4 space-y-4'>
              {fiatLoading ? (
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
                        {t('config.currencies.fiat.code')}
                      </TableHead>
                      <TableHead>{t('config.currencies.fiat.name')}</TableHead>
                      <TableHead>{t('config.currencies.fiat.symbol')}</TableHead>
                      <TableHead className='pr-6'>
                        {t('config.currencies.fiat.subunits')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(fiat ?? []).map((currency) => (
                      <TableRow key={currency.code}>
                        <TableCell className='pl-6 font-mono'>
                          {currency.code}
                        </TableCell>
                        <TableCell>{currency.name}</TableCell>
                        <TableCell>{currency.symbol}</TableCell>
                        <TableCell className='tabular-nums'>
                          {currency.subunits}
                        </TableCell>
                      </TableRow>
                    ))}
                    {(fiat ?? []).length === 0 && (
                      <TableRow>
                        <TableCell
                          colSpan={4}
                          className='h-24 text-center text-muted-foreground'
                        >
                          {t('config.currencies.fiat.empty')}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              )}
            </TabsContent>

            <TabsContent value='custom' className='mt-4 space-y-4'>
              <Alert>
                <AlertTitle>
                  {t('config.currencies.custom.immutableTitle')}
                </AlertTitle>
                <AlertDescription>
                  {t('config.currencies.custom.immutableDescription')}
                </AlertDescription>
              </Alert>
              <div className='flex justify-end'>
                <Button onClick={() => setCreateOpen(true)}>
                  <Plus className='size-4' />
                  {t('config.currencies.custom.create')}
                </Button>
              </div>
              {customLoading ? (
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
                        {t('config.currencies.custom.code')}
                      </TableHead>
                      <TableHead>{t('config.currencies.custom.name')}</TableHead>
                      <TableHead>{t('config.currencies.custom.symbol')}</TableHead>
                      <TableHead>{t('config.currencies.custom.precision')}</TableHead>
                      <TableHead>
                        {t('config.currencies.custom.decimalMark')} /{' '}
                        {t('config.currencies.custom.thousandSeparator')}
                      </TableHead>
                      <TableHead>
                        {t('config.currencies.custom.costBasisCount')}
                      </TableHead>
                      <TableHead className='pr-6'>
                        {t('config.currencies.custom.createdAt')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {customCurrencies.map((currency) => (
                      <TableRow key={currency.id}>
                        <TableCell className='pl-6 font-mono'>
                          <Badge variant='outline'>{currency.code}</Badge>
                        </TableCell>
                        <TableCell className='font-medium'>{currency.name}</TableCell>
                        <TableCell>{currency.symbol ?? '—'}</TableCell>
                        <TableCell className='tabular-nums'>
                          {currency.precision}
                        </TableCell>
                        <TableCell className='text-muted-foreground'>
                          {currency.decimalMark} / {currency.thousandSeparator}
                        </TableCell>
                        <TableCell className='tabular-nums'>
                          {currency.costBasis?.length ?? 0}
                        </TableCell>
                        <TableCell className='pr-6 text-muted-foreground'>
                          {formatShortDateTime(currency.createdAt)}
                        </TableCell>
                      </TableRow>
                    ))}
                    {customCurrencies.length === 0 && (
                      <TableRow>
                        <TableCell
                          colSpan={7}
                          className='h-24 text-center text-muted-foreground'
                        >
                          {t('config.currencies.custom.empty')}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              )}
            </TabsContent>
          </Tabs>
        </div>
      </Main>

      <CustomCurrencyDialog open={createOpen} onOpenChange={setCreateOpen} />
    </>
  )
}
