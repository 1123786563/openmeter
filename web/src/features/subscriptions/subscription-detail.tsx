import { useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import type { Plan, RateCard } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useCancelSubscription,
  useCustomer,
  usePlans,
  useSubscription,
} from '@/api/hooks'
import { formatAmount, formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
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
import { StatusBadge } from '@/components/status-badge'

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

/** Price cell for the plan rate-card table. */
function RateCardPrice({ card }: { card: RateCard }) {
  const price = card.price
  switch (price.type) {
    case 'flat':
      return formatAmount(price.amount, card.currency ?? '')
    case 'unit':
      return `${formatAmount(price.amount, card.currency ?? '')} / unit`
    case 'free':
      return '—'
    default:
      // Graduated/volume tiers are summarized; the full ladder is plan config.
      return '—'
  }
}

function PlanRateCards({ plan }: { plan: Plan }) {
  const { t } = useTranslation()
  const phases = plan.phases ?? []

  return (
    <Card className='py-0'>
      <CardHeader>
        <CardTitle className='text-base'>
          {t('subscriptions.detail.rateCards')}
        </CardTitle>
      </CardHeader>
      {phases.map((phase, phaseIndex) => (
        <div key={phase.key ?? phaseIndex}>
          <div className='border-b px-6 py-2 text-sm font-medium'>
            {phase.name}
            {phase.duration ? (
              <span className='ms-2 text-xs text-muted-foreground'>
                {phase.duration}
              </span>
            ) : null}
          </div>
          <Table>
            <TableHeader>
              <TableRow className='bg-hover/50'>
                <TableHead className='pl-6'>
                  {t('subscriptions.detail.rateCardName')}
                </TableHead>
                <TableHead>{t('subscriptions.detail.feature')}</TableHead>
                <TableHead>{t('subscriptions.detail.price')}</TableHead>
                <TableHead>{t('subscriptions.detail.cadence')}</TableHead>
                <TableHead className='pr-6'>
                  {t('subscriptions.detail.paymentTerm.label')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {phase.rateCards.map((card) => (
                <TableRow key={card.key}>
                  <TableCell className='pl-6 font-medium'>
                    {card.name}
                  </TableCell>
                  <TableCell>{card.feature?.id ?? '—'}</TableCell>
                  <TableCell className='tabular-nums'>
                    <RateCardPrice card={card} />
                  </TableCell>
                  <TableCell className='text-muted-foreground'>
                    {card.billingCadence ?? plan.billingCadence}
                  </TableCell>
                  <TableCell className='pr-6 text-muted-foreground'>
                    {card.paymentTerm
                      ? t(
                          `subscriptions.detail.paymentTerm.${card.paymentTerm}`,
                          {
                            defaultValue: card.paymentTerm,
                          }
                        )
                      : '—'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ))}
    </Card>
  )
}

export function SubscriptionDetail() {
  const { t } = useTranslation()
  const { subscriptionId } = useParams({
    from: '/_authenticated/subscriptions/$subscriptionId',
  })
  const { data: subscription, isLoading } = useSubscription(subscriptionId)
  const { data: customer } = useCustomer(subscription?.customerId ?? '')
  const { data: plans } = usePlans()
  const plan = plans?.find((candidate) => candidate.id === subscription?.planId)
  const [cancelOpen, setCancelOpen] = useState(false)
  const cancelMutation = useCancelSubscription()

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
  if (!subscription) return null

  const cancellable =
    subscription.status === 'active' || subscription.status === 'inactive'

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Link to='/subscriptions' className='hover:text-foreground'>
            {t('sidebar.subscriptions')}
          </Link>
          <span>/</span>
          <span className='font-mono text-xs text-foreground'>
            {subscription.id}
          </span>
        </div>
        <div className='mt-2 flex items-center justify-between gap-4'>
          <h1 className='flex items-center gap-3 text-2xl font-bold tracking-tight md:text-3xl'>
            {customer?.name ?? subscription.customerId}
            <StatusBadge domain='subscription' value={subscription.status} />
          </h1>
          {cancellable && (
            <Button variant='destructive' onClick={() => setCancelOpen(true)}>
              {t('subscriptions.cancel')}
            </Button>
          )}
        </div>

        <div className='mt-6 space-y-6'>
          <Card>
            <CardHeader>
              <CardTitle className='text-base'>
                {t('subscriptions.detail.info')}
              </CardTitle>
            </CardHeader>
            <Separator />
            <CardContent className='grid gap-x-10 py-4 sm:grid-cols-2 lg:grid-cols-3'>
              <InfoRow label={t('subscriptions.fields.id')}>
                <span className='font-mono text-xs'>{subscription.id}</span>
              </InfoRow>
              <InfoRow label={t('subscriptions.fields.customer')}>
                {customer ? (
                  <Link
                    to='/customers/$customerId'
                    params={{ customerId: customer.id }}
                    className='hover:underline'
                  >
                    {customer.name}
                  </Link>
                ) : (
                  subscription.customerId
                )}
              </InfoRow>
              <InfoRow label={t('subscriptions.fields.plan')}>
                {plan ? `${plan.name} · v${plan.version}` : '—'}
              </InfoRow>
              <InfoRow label={t('subscriptions.fields.billingAnchor')}>
                {formatDateTime(subscription.billingAnchor)}
              </InfoRow>
              <InfoRow label={t('subscriptions.fields.settlementMode')}>
                {subscription.settlementMode
                  ? t(
                      `subscriptions.settlementMode.${subscription.settlementMode}`,
                      { defaultValue: subscription.settlementMode }
                    )
                  : '—'}
              </InfoRow>
              <InfoRow label={t('subscriptions.fields.createdAt')}>
                {formatDateTime(subscription.createdAt)}
              </InfoRow>
              <InfoRow label={t('subscriptions.fields.updatedAt')}>
                {formatDateTime(subscription.updatedAt)}
              </InfoRow>
              <InfoRow label={t('subscriptions.fields.deletedAt')}>
                {formatDateTime(subscription.deletedAt)}
              </InfoRow>
            </CardContent>
          </Card>

          {/* Status timeline: v3 subscriptions expose lifecycle timestamps only. */}
          <Card className='py-0'>
            <CardHeader>
              <CardTitle className='text-base'>
                {t('subscriptions.detail.timeline.label')}
              </CardTitle>
            </CardHeader>
            <CardContent className='pt-0'>
              <ol className='relative ms-3 border-s border-border'>
                {(
                  [
                    ['createdAt', subscription.createdAt],
                    ['updatedAt', subscription.updatedAt],
                    ['deletedAt', subscription.deletedAt],
                  ] as const
                ).map(([key, value]) =>
                  value ? (
                    <li key={key} className='ms-6 pb-4 last:pb-0'>
                      <span className='absolute -start-[5px] mt-1.5 size-2.5 rounded-full bg-primary' />
                      <p className='text-sm font-medium'>
                        {t(`subscriptions.detail.timeline.${key}`)}
                      </p>
                      <p className='text-xs text-muted-foreground'>
                        {formatDateTime(value)}
                      </p>
                    </li>
                  ) : null
                )}
              </ol>
            </CardContent>
          </Card>

          {plan ? <PlanRateCards plan={plan} /> : null}
        </div>
      </Main>

      <ConfirmDialog
        open={cancelOpen}
        onOpenChange={setCancelOpen}
        title={t('subscriptions.cancelConfirm.title')}
        desc={t('subscriptions.cancelConfirm.description')}
        confirmText={t('subscriptions.cancelConfirm.confirm')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={cancelMutation.isPending}
        handleConfirm={() =>
          cancelMutation.mutate(
            { subscriptionId: subscription.id, body: { timing: 'immediate' } },
            {
              onSuccess: () => {
                toast.success(t('subscriptions.toast.canceled'))
                setCancelOpen(false)
              },
              onError: handleServerError,
            }
          )
        }
      />
    </>
  )
}
