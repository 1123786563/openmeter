import { useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { CheckCircle2, FastForward, RefreshCw, ScanLine } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useInvoice, useInvoiceAction, type InvoiceAction } from '@/api/hooks'
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

const ACTION_META: Record<
  Exclude<InvoiceAction, never>,
  { icon: React.ElementType; destructive?: boolean }
> = {
  advance: { icon: FastForward },
  approve: { icon: CheckCircle2 },
  retry: { icon: RefreshCw },
  snapshotQuantities: { icon: ScanLine },
}

export function InvoiceDetail() {
  const { t } = useTranslation()
  const { invoiceId } = useParams({
    from: '/_authenticated/invoices/$invoiceId',
  })
  const { data: invoice, isLoading } = useInvoice(invoiceId)
  const actionMutation = useInvoiceAction()
  const [confirmAction, setConfirmAction] = useState<InvoiceAction | null>(null)

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
  if (!invoice) return null

  const available = invoice.statusDetails.availableActions
  const actions = (Object.keys(ACTION_META) as InvoiceAction[]).filter(
    (action) => available[action] !== undefined
  )

  const runAction = (action: InvoiceAction) => {
    actionMutation.mutate(
      { invoiceId: invoice.id, action },
      {
        onSuccess: () => {
          toast.success(
            t(`invoices.actions.${action}.success`, { defaultValue: action })
          )
          setConfirmAction(null)
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Link to='/invoices' className='hover:text-foreground'>
            {t('sidebar.invoices')}
          </Link>
          <span>/</span>
          <span className='text-foreground'>{invoice.number}</span>
        </div>
        <div className='mt-2 flex flex-wrap items-center justify-between gap-4'>
          <h1 className='flex items-center gap-3 text-2xl font-bold tracking-tight md:text-3xl'>
            {invoice.number}
            <StatusBadge domain='invoice' value={invoice.status} />
          </h1>
          <div className='flex flex-wrap items-center gap-2'>
            {actions.map((action) => {
              const Icon = ACTION_META[action].icon
              return (
                <Button
                  key={action}
                  variant='outline'
                  size='sm'
                  disabled={actionMutation.isPending}
                  onClick={() => setConfirmAction(action)}
                >
                  <Icon className='size-3.5' />
                  {t(`invoices.actions.${action}.label`)}
                </Button>
              )
            })}
          </div>
        </div>

        <div className='mt-6 grid gap-6 lg:grid-cols-3'>
          <Card className='lg:col-span-2'>
            <CardHeader>
              <CardTitle className='text-base'>
                {t('invoices.detail.lines')}
              </CardTitle>
            </CardHeader>
            <Separator />
            <CardContent className='px-0 py-0'>
              <Table>
                <TableHeader>
                  <TableRow className='bg-hover/50'>
                    <TableHead className='pl-6'>
                      {t('invoices.detail.lineName')}
                    </TableHead>
                    <TableHead>{t('invoices.detail.servicePeriod')}</TableHead>
                    <TableHead>{t('invoices.detail.lineTotal')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(invoice.lines ?? []).map((line) => (
                    <TableRow key={line.id ?? line.name}>
                      <TableCell className='pl-6 font-medium'>
                        {line.name}
                      </TableCell>
                      <TableCell className='text-muted-foreground'>
                        {formatDateTime(line.servicePeriod.from)} →{' '}
                        {formatDateTime(line.servicePeriod.to)}
                      </TableCell>
                      <TableCell className='tabular-nums'>
                        {formatAmount(line.totals.total, invoice.currency)}
                      </TableCell>
                    </TableRow>
                  ))}
                  {(invoice.lines ?? []).length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={3}
                        className='h-20 text-center text-muted-foreground'
                      >
                        {t('invoices.detail.noLines')}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <div className='space-y-6'>
            <Card>
              <CardHeader>
                <CardTitle className='text-base'>
                  {t('invoices.detail.totals')}
                </CardTitle>
              </CardHeader>
              <Separator />
              <CardContent className='py-4'>
                <InfoRow label={t('invoices.detail.subtotal')}>
                  <span className='tabular-nums'>
                    {formatAmount(invoice.totals.amount, invoice.currency)}
                  </span>
                </InfoRow>
                <InfoRow label={t('invoices.detail.discounts')}>
                  <span className='tabular-nums'>
                    −
                    {formatAmount(
                      invoice.totals.discountsTotal,
                      invoice.currency
                    )}
                  </span>
                </InfoRow>
                <InfoRow label={t('invoices.detail.credits')}>
                  <span className='tabular-nums'>
                    −
                    {formatAmount(
                      invoice.totals.creditsTotal,
                      invoice.currency
                    )}
                  </span>
                </InfoRow>
                <InfoRow label={t('invoices.detail.tax')}>
                  <span className='tabular-nums'>
                    {formatAmount(invoice.totals.taxesTotal, invoice.currency)}
                  </span>
                </InfoRow>
                <Separator className='my-2' />
                <InfoRow label={t('invoices.detail.grandTotal')}>
                  <span className='text-base tabular-nums'>
                    {formatAmount(invoice.totals.total, invoice.currency)}
                  </span>
                </InfoRow>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className='text-base'>
                  {t('invoices.detail.info')}
                </CardTitle>
              </CardHeader>
              <Separator />
              <CardContent className='py-4'>
                <InfoRow label={t('invoices.fields.customer')}>
                  {invoice.customer.name}
                </InfoRow>
                <InfoRow label={t('invoices.fields.currency')}>
                  {invoice.currency}
                </InfoRow>
                <InfoRow label={t('invoices.fields.status')}>
                  {invoice.statusDetails.extendedStatus}
                </InfoRow>
                <InfoRow label={t('invoices.fields.issuedAt')}>
                  {formatDateTime(invoice.issuedAt)}
                </InfoRow>
                <InfoRow label={t('invoices.fields.dueAt')}>
                  {formatDateTime(invoice.dueAt)}
                </InfoRow>
                <InfoRow label={t('invoices.fields.createdAt')}>
                  {formatDateTime(invoice.createdAt)}
                </InfoRow>
                <InfoRow label={t('invoices.detail.servicePeriod')}>
                  {formatDateTime(invoice.servicePeriod.from)} →{' '}
                  {formatDateTime(invoice.servicePeriod.to)}
                </InfoRow>
              </CardContent>
            </Card>
          </div>
        </div>
      </Main>

      <ConfirmDialog
        open={confirmAction !== null}
        onOpenChange={(open) => !open && setConfirmAction(null)}
        title={t(`invoices.actions.${confirmAction}.confirmTitle`, {
          defaultValue: '',
        })}
        desc={t(`invoices.actions.${confirmAction}.confirmDescription`, {
          defaultValue: '',
        })}
        confirmText={t(`invoices.actions.${confirmAction}.label`, {
          defaultValue: '',
        })}
        cancelBtnText={t('common.cancel')}
        isLoading={actionMutation.isPending}
        handleConfirm={() => {
          if (confirmAction) runAction(confirmAction)
        }}
      />
    </>
  )
}
