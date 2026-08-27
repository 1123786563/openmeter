import { useState } from 'react'
import { subDays } from 'date-fns'
import { Link } from '@tanstack/react-router'
import type { Customer } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import {
  useFeature,
  useFeatureCostQuery,
  type FeatureCostQueryParams,
} from '@/api/hooks'
import { formatDateTime, formatNumber } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { CustomerPicker } from '@/features/customers/customer-picker'

function toLocalInputValue(date: Date): string {
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

export function FeatureDetailPage({ featureId }: { featureId: string }) {
  const { t } = useTranslation()
  const { data: feature, isLoading } = useFeature(featureId)

  const [customer, setCustomer] = useState<Customer | null>(null)
  const [fromInput, setFromInput] = useState(
    toLocalInputValue(subDays(new Date(), 30))
  )
  const [toInput, setToInput] = useState(toLocalInputValue(new Date()))
  // Filters only hit the API after the user presses the query button.
  const [submitted, setSubmitted] = useState<FeatureCostQueryParams>({
    from: subDays(new Date(), 30),
    to: new Date(),
  })

  const costQuery = useFeatureCostQuery(featureId, submitted, {
    enabled: Boolean(submitted.from || submitted.to || submitted.customerId),
  })

  if (isLoading) {
    return (
      <>
        <Header />
        <Main fixed>
          <Skeleton className='h-64 w-full' />
        </Main>
      </>
    )
  }
  if (!feature) return null

  const rows = costQuery.data?.data ?? []
  // Reserved dimensions are implicit in the UI (customer picked above);
  // surface any extra group-by dimensions inline.
  const extraDimensions = (dimensions: Record<string, string>) =>
    Object.entries(dimensions)
      .filter(([key]) => key !== 'customer_id' && key !== 'subject')
      .map(([key, value]) => `${key}=${value}`)
      .join(', ')

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Link to='/config/features' className='hover:text-foreground'>
            {t('config.features.title')}
          </Link>
          <span>/</span>
          <span className='font-mono text-xs text-foreground'>
            {feature.key}
          </span>
        </div>
        <h1 className='mt-2 text-2xl font-bold tracking-tight md:text-3xl'>
          {feature.name}
        </h1>

        <Card className='mt-6 py-0'>
          <CardContent className='grid gap-x-8 gap-y-2 p-4 pt-4 sm:grid-cols-2'>
            <div className='flex gap-2 text-sm'>
              <span className='text-muted-foreground'>
                {t('config.features.fields.key')}:
              </span>
              <code className='font-mono text-xs'>{feature.key}</code>
            </div>
            <div className='flex gap-2 text-sm'>
              <span className='text-muted-foreground'>
                {t('config.features.fields.description')}:
              </span>
              <span>{feature.description || '—'}</span>
            </div>
            <div className='flex gap-2 text-sm'>
              <span className='text-muted-foreground'>
                {t('config.features.fields.createdAt')}:
              </span>
              <span className='tabular-nums'>
                {formatDateTime(feature.createdAt)}
              </span>
            </div>
            <div className='flex gap-2 text-sm'>
              <span className='text-muted-foreground'>
                {t('config.features.fields.updatedAt')}:
              </span>
              <span className='tabular-nums'>
                {formatDateTime(feature.updatedAt)}
              </span>
            </div>
          </CardContent>
        </Card>

        <Card className='mt-6 py-0'>
          <CardHeader>
            <CardTitle className='text-base'>
              {t('config.featureDetail.costQuery.title')}
            </CardTitle>
          </CardHeader>
          <CardContent className='space-y-4 pt-0'>
            <div className='flex flex-wrap items-end gap-3'>
              <div className='w-72 space-y-1'>
                <label className='text-xs text-muted-foreground'>
                  {t('config.featureDetail.costQuery.customer')}
                </label>
                <CustomerPicker
                  value={customer}
                  onChange={setCustomer}
                  className='h-8'
                />
              </div>
              <div className='space-y-1'>
                <label className='text-xs text-muted-foreground'>
                  {t('config.featureDetail.costQuery.from')}
                </label>
                <Input
                  type='datetime-local'
                  className='h-8 w-56'
                  value={fromInput}
                  onChange={(event) => setFromInput(event.target.value)}
                />
              </div>
              <div className='space-y-1'>
                <label className='text-xs text-muted-foreground'>
                  {t('config.featureDetail.costQuery.to')}
                </label>
                <Input
                  type='datetime-local'
                  className='h-8 w-56'
                  value={toInput}
                  onChange={(event) => setToInput(event.target.value)}
                />
              </div>
              <Button
                size='sm'
                className='h-8'
                onClick={() =>
                  setSubmitted({
                    customerId: customer?.id,
                    from: fromInput ? new Date(fromInput) : undefined,
                    to: toInput ? new Date(toInput) : undefined,
                  })
                }
              >
                {t('config.featureDetail.costQuery.run')}
              </Button>
            </div>

            {costQuery.isLoading ? (
              <div className='space-y-2'>
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className='h-8 w-full' />
                ))}
              </div>
            ) : rows.length === 0 ? (
              <p className='py-8 text-center text-sm text-muted-foreground'>
                {t('config.featureDetail.costQuery.empty')}
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className='bg-hover/50'>
                    <TableHead>
                      {t('config.featureDetail.costQuery.period')}
                    </TableHead>
                    <TableHead>
                      {t('config.featureDetail.costQuery.usage')}
                    </TableHead>
                    <TableHead>
                      {t('config.featureDetail.costQuery.cost')}
                    </TableHead>
                    <TableHead>
                      {t('config.featureDetail.costQuery.dimensions')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((row, index) => (
                    <TableRow key={index}>
                      <TableCell className='text-muted-foreground tabular-nums'>
                        {formatDateTime(row.from)} ~ {formatDateTime(row.to)}
                      </TableCell>
                      <TableCell className='tabular-nums'>
                        {formatNumber(row.usage)}
                      </TableCell>
                      <TableCell className='tabular-nums'>
                        {row.cost === null ? (
                          <span
                            className='text-muted-foreground'
                            title={row.detail}
                          >
                            {row.detail ||
                              t('config.featureDetail.costQuery.noPrice')}
                          </span>
                        ) : (
                          `${formatNumber(row.cost)} ${row.currency}`
                        )}
                      </TableCell>
                      <TableCell className='text-xs text-muted-foreground'>
                        {extraDimensions(row.dimensions) || '—'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </Main>
    </>
  )
}
