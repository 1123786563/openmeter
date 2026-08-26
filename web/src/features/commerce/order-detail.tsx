import { Link, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useOrder } from '@/api/hooks'
import { formatDateTime, formatFen, formatNumber } from '@/lib/format'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
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

/** Order detail; reached from the order list or directly by URL. */
export function OrderDetail() {
  const { t } = useTranslation()
  const { orderId } = useParams({
    from: '/_authenticated/commerce/orders/$orderId',
  })
  const { data: order, isLoading } = useOrder(orderId)

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
  if (!order) return null

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Link to='/commerce/orders' className='hover:text-foreground'>
            {t('sidebar.orders')}
          </Link>
          <span>/</span>
          <span className='font-mono text-xs text-foreground'>{order.id}</span>
        </div>
        <h1 className='mt-2 flex flex-wrap items-center gap-3 text-2xl font-bold tracking-tight md:text-3xl'>
          {t('commerce.orders.detailTitle')}
          <EnumBadge domain='commerce' kind='orderKind' value={order.kind} />
          <StatusBadge domain='order' value={order.status} />
        </h1>

        <Card className='mt-6'>
          <CardHeader>
            <CardTitle className='text-base'>
              {t('commerce.orders.info')}
            </CardTitle>
          </CardHeader>
          <Separator />
          <CardContent className='grid gap-x-10 py-4 sm:grid-cols-2 lg:grid-cols-3'>
            <InfoRow label={t('commerce.orders.fields.id')}>
              <span className='font-mono text-xs'>{order.id}</span>
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.billingCustomerId')}>
              <span className='font-mono text-xs'>
                {order.billingCustomerId}
              </span>
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.plan')}>
              {order.plan
                ? `${order.plan.planKey} · v${order.plan.planVersion}`
                : '—'}
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.rechargeProductId')}>
              {order.rechargeProductId ?? '—'}
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.amount')}>
              <span className='tabular-nums'>
                {formatFen(order.amountFen, order.currency)}
              </span>
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.credits')}>
              {order.credits !== undefined ? formatNumber(order.credits) : '—'}
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.trackingNumber')}>
              {order.businessTrackingNumber ?? '—'}
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.idempotencyKey')}>
              {order.idempotencyKey}
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.createdAt')}>
              {formatDateTime(order.createdAt)}
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.updatedAt')}>
              {formatDateTime(order.updatedAt)}
            </InfoRow>
            <InfoRow label={t('commerce.orders.fields.expiredAt')}>
              {formatDateTime(order.expiredAt)}
            </InfoRow>
          </CardContent>
        </Card>
      </Main>
    </>
  )
}
