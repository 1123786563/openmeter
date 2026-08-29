import { useState } from 'react'
import type { CommerceReceivablePeriod } from '@openmeter/client'
import { ChevronRight, FilePlus2, HandCoins } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useReceivablePeriods } from '@/api/hooks'
import { formatFen, formatNumber, formatShortDateTime } from '@/lib/format'
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
import { StatusBadge } from '@/components/status-badge'
import { ExternalInvoiceDialog } from './external-invoice-dialog'
import { OfflinePaymentDialog } from './offline-payment-dialog'

/**
 * Receivable periods for one customer, cursor paginated. Mirrors the events
 * page "load more": `pages` accumulates cursors already advanced past, `data`
 * is the current cursor's page; meta.page.next is a raw cursor (verified in
 * api/v3/response/cursorpagination.go), reusable as page[after].
 */
export function ReceivablePeriodsTab({ customerId }: { customerId: string }) {
  const { t } = useTranslation()
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [pages, setPages] = useState<
    Awaited<ReturnType<typeof useReceivablePeriods>['data']>[]
  >([])

  const { data, isLoading, isFetching } = useReceivablePeriods(
    customerId,
    cursor
  )

  const [invoiceTarget, setInvoiceTarget] =
    useState<CommerceReceivablePeriod | null>(null)

  const [offlineOpen, setOfflineOpen] = useState(false)

  const allPeriods = [
    ...pages.flatMap((page) => page?.data ?? []),
    ...(data?.data ?? []),
  ]

  const loadMore = () => {
    if (data?.meta.page.next) {
      setPages((prev) => [...prev, data])
      setCursor(data.meta.page.next ?? undefined)
    }
  }

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between'>
        <p className='text-xs text-muted-foreground'>
          {t('customers.receivablePeriods.offlinePayment.note')}
        </p>
        <Button
          variant='outline'
          size='sm'
          onClick={() => setOfflineOpen(true)}
        >
          <HandCoins />
          {t('customers.receivablePeriods.offlinePayment.action')}
        </Button>
      </div>
      <Table>
        <TableHeader>
          <TableRow className='bg-hover/50'>
            <TableHead className='pl-6'>
              {t('customers.receivablePeriods.fields.period')}
            </TableHead>
            <TableHead>
              {t('customers.receivablePeriods.fields.creditsConsumed')}
            </TableHead>
            <TableHead>
              {t('customers.receivablePeriods.fields.amountDue')}
            </TableHead>
            <TableHead>
              {t('customers.receivablePeriods.fields.amountPaid')}
            </TableHead>
            <TableHead>
              {t('customers.receivablePeriods.fields.status')}
            </TableHead>
            <TableHead>
              {t('customers.receivablePeriods.fields.closedAt')}
            </TableHead>
            <TableHead className='pr-6 text-right'>
              {t('customers.receivablePeriods.fields.actions')}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && pages.length === 0 ? (
            Array.from({ length: 5 }).map((_, i) => (
              <TableRow key={i}>
                <TableCell colSpan={7}>
                  <Skeleton className='h-5 w-full' />
                </TableCell>
              </TableRow>
            ))
          ) : allPeriods.length === 0 && !isFetching ? (
            <TableRow>
              <TableCell
                colSpan={7}
                className='h-24 text-center text-muted-foreground'
              >
                {t('customers.receivablePeriods.empty')}
              </TableCell>
            </TableRow>
          ) : (
            allPeriods.map((period) => (
              <TableRow key={period.id}>
                <TableCell className='pl-6 text-muted-foreground'>
                  {formatShortDateTime(period.periodStart)} ~{' '}
                  {formatShortDateTime(period.periodEnd)}
                </TableCell>
                <TableCell className='tabular-nums'>
                  {formatNumber(period.creditsConsumed)}
                </TableCell>
                <TableCell className='tabular-nums'>
                  {formatFen(period.amountDueFen, period.currency)}
                </TableCell>
                <TableCell className='tabular-nums'>
                  {formatFen(period.amountPaidFen, period.currency)}
                </TableCell>
                <TableCell>
                  <StatusBadge
                    domain='receivablePeriod'
                    value={period.status}
                  />
                </TableCell>
                <TableCell className='text-muted-foreground'>
                  {formatShortDateTime(period.closedAt)}
                </TableCell>
                <TableCell className='pr-6'>
                  <div className='flex justify-end'>
                    <Button
                      variant='ghost'
                      size='sm'
                      className='h-7 px-2'
                      onClick={() => setInvoiceTarget(period)}
                    >
                      <FilePlus2 className='size-4' />
                      {t('customers.receivablePeriods.externalInvoice.action')}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
      {data?.meta.page.next && (
        <div className='flex justify-center pb-2'>
          <Button
            variant='outline'
            size='sm'
            onClick={loadMore}
            disabled={isFetching}
          >
            {t('customers.receivablePeriods.loadMore')}
            <ChevronRight className='size-3.5' />
          </Button>
        </div>
      )}
      <ExternalInvoiceDialog
        open={Boolean(invoiceTarget)}
        onOpenChange={(open) => !open && setInvoiceTarget(null)}
        customerId={customerId}
        period={invoiceTarget}
      />
      <OfflinePaymentDialog
        open={offlineOpen}
        onOpenChange={setOfflineOpen}
        customerId={customerId}
        periods={allPeriods}
      />
    </div>
  )
}
