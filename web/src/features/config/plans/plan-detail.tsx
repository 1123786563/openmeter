import { Link, useParams } from '@tanstack/react-router'
import type { RateCard } from '@openmeter/client'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePlan } from '@/api/hooks'
import { formatAmount, formatDateTime } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
import { EnumBadge, StatusBadge } from '@/components/status-badge'

function InfoRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className='flex items-baseline justify-between gap-4 py-1.5'>
      <span className='text-sm text-muted-foreground'>{label}</span>
      <span className='text-end text-sm font-medium break-all'>{children}</span>
    </div>
  )
}

/** Human-readable price summary per rate card; tiers collapse to a count. */
function RateCardPriceSummary({
  card,
  planCurrency,
}: {
  card: RateCard
  planCurrency: string
}) {
  const { t } = useTranslation()
  const currency = card.currency ?? planCurrency
  switch (card.price.type) {
    case 'free':
      return (
        <span className='text-muted-foreground'>{t('plan.price.free')}</span>
      )
    case 'flat':
      return (
        <span className='tabular-nums'>
          {formatAmount(card.price.amount, currency)}
        </span>
      )
    case 'unit':
      return (
        <span className='tabular-nums'>
          {formatAmount(card.price.amount, currency)} /{' '}
          {t('config.plans.detail.unit')}
        </span>
      )
    case 'graduated':
    case 'volume':
      return (
        <span className='text-muted-foreground'>
          {t('config.plans.detail.tierSummary', {
            count: card.price.tiers.length,
          })}
        </span>
      )
  }
}

export function PlanDetail() {
  const { t } = useTranslation()
  const { planId } = useParams({ from: '/_authenticated/config/plans/$planId' })
  const { data: plan, isLoading } = usePlan(planId)

  if (isLoading) {
    return (
      <>
        <Header />
        <Main fixed>
          <Skeleton className='h-8 w-64' />
          <Skeleton className='mt-6 h-64 w-full' />
        </Main>
      </>
    )
  }
  if (!plan) {
    return (
      <>
        <Header />
        <Main fixed>
          <p className='text-sm text-muted-foreground'>
            {t('config.plans.detail.notFound')}
          </p>
        </Main>
      </>
    )
  }

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2'>
          <Button variant='ghost' size='sm' asChild>
            <Link to='/config/plans'>
              <ArrowLeft className='size-4' />
              {t('config.plans.detail.back')}
            </Link>
          </Button>
        </div>
        <div className='mt-4 flex items-center gap-3'>
          <h1 className='text-2xl font-bold tracking-tight'>{plan.name}</h1>
          <StatusBadge domain='plan' value={plan.status} />
          <span className='text-sm text-muted-foreground'>v{plan.version}</span>
        </div>

        <Card className='mt-6 py-0'>
          <CardContent className='divide-y'>
            <InfoRow label={t('config.plans.fields.key')}>
              <span className='font-mono text-xs'>{plan.key}</span>
            </InfoRow>
            <InfoRow label={t('config.plans.fields.currency')}>
              {plan.currency}
            </InfoRow>
            <InfoRow label={t('config.plans.fields.billingCadence')}>
              {plan.billingCadence}
            </InfoRow>
            <InfoRow label={t('config.plans.fields.createdAt')}>
              {formatDateTime(plan.createdAt)}
            </InfoRow>
            <InfoRow label={t('config.plans.fields.updatedAt')}>
              {formatDateTime(plan.updatedAt)}
            </InfoRow>
            {plan.description ? (
              <InfoRow label={t('config.plans.fields.description')}>
                {plan.description}
              </InfoRow>
            ) : null}
          </CardContent>
        </Card>

        <h2 className='mt-8 text-lg font-semibold'>
          {t('config.plans.detail.phases')}
        </h2>
        {plan.phases.map((phase, phaseIndex) => (
          <Card key={phase.key ?? phaseIndex} className='mt-4 py-0'>
            <CardHeader>
              <CardTitle className='flex items-baseline gap-2 text-base'>
                <span>
                  {t('config.plans.detail.phaseIndex', {
                    index: phaseIndex + 1,
                  })}{' '}
                  · {phase.name}
                </span>
                <span className='text-xs font-normal text-muted-foreground'>
                  {phase.duration
                    ? t('config.plans.detail.duration', {
                        duration: phase.duration,
                      })
                    : t('config.plans.detail.noDuration')}
                </span>
              </CardTitle>
            </CardHeader>
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('config.plans.detail.rateCardName')}
                  </TableHead>
                  <TableHead>{t('config.plans.detail.priceType')}</TableHead>
                  <TableHead>{t('config.plans.detail.feature')}</TableHead>
                  <TableHead>{t('config.plans.detail.price')}</TableHead>
                  <TableHead>{t('config.plans.detail.cadence')}</TableHead>
                  <TableHead className='pr-6'>
                    {t('config.plans.detail.key')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {phase.rateCards.map((card) => (
                  <TableRow key={card.key}>
                    <TableCell className='pl-6 font-medium'>
                      {card.name}
                    </TableCell>
                    <TableCell>
                      <EnumBadge
                        domain='plan'
                        kind='priceType'
                        value={card.price.type}
                      />
                    </TableCell>
                    <TableCell className='font-mono text-xs text-muted-foreground'>
                      {card.feature?.id ?? '—'}
                    </TableCell>
                    <TableCell>
                      <RateCardPriceSummary
                        card={card}
                        planCurrency={plan.currency}
                      />
                    </TableCell>
                    <TableCell className='text-muted-foreground'>
                      {card.billingCadence ?? plan.billingCadence}
                    </TableCell>
                    <TableCell className='pr-6 font-mono text-xs text-muted-foreground'>
                      {card.key}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        ))}
      </Main>
    </>
  )
}
