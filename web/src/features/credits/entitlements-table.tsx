import { useTranslation } from 'react-i18next'
import {
  useCustomerEntitlements,
  useCustomerEntitlementValue,
} from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { EnumBadge } from '@/components/status-badge'

function EntitlementValueCell({
  customerId,
  entitlementId,
  featureKey,
  type,
}: {
  customerId: string
  entitlementId: string
  featureKey: string
  type: string
}) {
  const { t } = useTranslation()
  const { data, isLoading } = useCustomerEntitlementValue(
    customerId,
    entitlementId,
    featureKey
  )

  if (isLoading) return <Skeleton className='h-5 w-16' />
  if (!data) return '—'
  if (!data.hasAccess) {
    return (
      <span className='text-muted-foreground'>
        {t('credits.entitlements.noAccess')}
      </span>
    )
  }
  if (type === 'metered') {
    return (
      <span className='tabular-nums'>
        {t('credits.entitlements.balanceUsage', {
          balance: data.balance ?? 0,
          usage: data.usage ?? 0,
        })}
      </span>
    )
  }
  if (type === 'static' && data.config) {
    return <code className='text-xs'>{data.config}</code>
  }
  return t('credits.entitlements.hasAccess')
}

/** v2 entitlements with live values (balance/usage for metered ones). */
export function EntitlementsTable({ customerId }: { customerId: string }) {
  const { t } = useTranslation()
  const { data, isLoading } = useCustomerEntitlements(customerId)

  if (isLoading) {
    return (
      <div className='space-y-2'>
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className='h-8 w-full' />
        ))}
      </div>
    )
  }

  if (!data || data.length === 0) {
    return (
      <p className='py-8 text-center text-sm text-muted-foreground'>
        {t('credits.entitlements.empty')}
      </p>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow className='bg-hover/50'>
          <TableHead>{t('credits.entitlements.feature')}</TableHead>
          <TableHead>{t('credits.entitlements.type')}</TableHead>
          <TableHead>{t('credits.entitlements.value')}</TableHead>
          <TableHead>{t('credits.entitlements.activeFrom')}</TableHead>
          <TableHead>{t('credits.entitlements.usagePeriod')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.map((entitlement) => (
          <TableRow key={entitlement.id}>
            <TableCell className='font-medium'>
              {entitlement.featureKey}
            </TableCell>
            <TableCell>
              <EnumBadge
                domain='credits'
                kind='entitlementType'
                value={entitlement.type}
              />
            </TableCell>
            <TableCell>
              <EntitlementValueCell
                customerId={customerId}
                entitlementId={entitlement.id}
                featureKey={entitlement.featureKey}
                type={entitlement.type}
              />
            </TableCell>
            <TableCell className='text-muted-foreground'>
              {formatDateTime(entitlement.activeFrom)}
            </TableCell>
            <TableCell className='text-muted-foreground'>
              {formatDateTime(entitlement.usagePeriod.from)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
