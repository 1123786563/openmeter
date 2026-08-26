import { Link, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useRefund } from '@/api/hooks'
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

/** Read-only refund detail; reached from the refund list or directly by URL. */
export function RefundDetail() {
  const { t } = useTranslation()
  const { refundId } = useParams({
    from: '/_authenticated/commerce/refunds/$refundId',
  })
  const { data: refund, isLoading } = useRefund(refundId)

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
  if (!refund) return null

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Link to='/commerce/refunds' className='hover:text-foreground'>
            {t('sidebar.refunds')}
          </Link>
          <span>/</span>
          <span className='font-mono text-xs text-foreground'>{refund.id}</span>
        </div>
        <h1 className='mt-2 flex flex-wrap items-center gap-3 text-2xl font-bold tracking-tight md:text-3xl'>
          {t('commerce.refunds.detailTitle')}
          <EnumBadge
            domain='commerce'
            kind='refundProvider'
            value={refund.provider}
          />
          <StatusBadge domain='refund' value={refund.status} />
        </h1>

        <Card className='mt-6'>
          <CardHeader>
            <CardTitle className='text-base'>
              {t('commerce.refunds.info')}
            </CardTitle>
          </CardHeader>
          <Separator />
          <CardContent className='grid gap-x-10 py-4 sm:grid-cols-2 lg:grid-cols-3'>
            <InfoRow label={t('commerce.refunds.fields.id')}>
              <span className='font-mono text-xs'>{refund.id}</span>
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.billingCustomerId')}>
              <span className='font-mono text-xs'>
                {refund.billingCustomerId}
              </span>
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.orderId')}>
              <Link
                to='/commerce/orders/$orderId'
                params={{ orderId: refund.orderId }}
                className='font-mono text-xs hover:underline'
              >
                {refund.orderId}
              </Link>
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.amount')}>
              <span className='tabular-nums'>
                {formatFen(refund.amountFen, 'CNY')}
              </span>
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.creditsReversed')}>
              {formatNumber(refund.creditsReversed)}
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.reason')}>
              {refund.reason}
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.providerRefundId')}>
              {refund.providerRefundId ?? '—'}
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.trackingNumber')}>
              {refund.businessTrackingNumber ?? '—'}
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.idempotencyKey')}>
              {refund.idempotencyKey}
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.createdAt')}>
              {formatDateTime(refund.createdAt)}
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.updatedAt')}>
              {formatDateTime(refund.updatedAt)}
            </InfoRow>
            <InfoRow label={t('commerce.refunds.fields.fulfilledAt')}>
              {formatDateTime(refund.fulfilledAt)}
            </InfoRow>
          </CardContent>
        </Card>
      </Main>
    </>
  )
}
