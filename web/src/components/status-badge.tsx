import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

/**
 * Maps an API enum value to a tone + i18n label. Keys resolve to
 * `<domain>.status.<value>` / `<domain>.<kind>.<value>` namespaces defined in
 * the locale files.
 */

type Tone = 'default' | 'secondary' | 'destructive' | 'outline' | 'success'

const toneClass: Record<Tone, string> = {
  default: '',
  secondary: '',
  destructive: '',
  outline: '',
  success:
    'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300',
}

const warnClass =
  'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300'

const infoClass =
  'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900 dark:bg-sky-950 dark:text-sky-300'

const purpleClass =
  'border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-900 dark:bg-violet-950 dark:text-violet-300'

/** Per-domain value → tone overrides; anything unlisted falls back per tone set. */
const tones: Record<
  string,
  Record<string, Tone | 'warn' | 'info' | 'purple'>
> = {
  invoice: {
    draft: 'secondary',
    issuing: 'info',
    issued: 'info',
    payment_processing: 'info',
    overdue: 'warn',
    paid: 'success',
    uncollectible: 'destructive',
    voided: 'outline',
  },
  subscription: {
    active: 'success',
    inactive: 'secondary',
    canceled: 'outline',
    scheduled: 'info',
  },
  plan: {
    draft: 'secondary',
    active: 'success',
    archived: 'outline',
    scheduled: 'info',
  },
  order: {
    created: 'secondary',
    awaiting_payment: 'warn',
    paid: 'info',
    fulfilled: 'success',
    cancelled: 'outline',
    expired: 'outline',
    refund_pending: 'warn',
    partially_refunded: 'warn',
    refunded: 'outline',
  },
  refund: {
    pending_fence: 'warn',
    provider_processing: 'info',
    ledger_reversing: 'info',
    fulfilled: 'success',
    failed: 'destructive',
  },
  grant: {
    active: 'success',
    voided: 'outline',
    expired: 'secondary',
    settled: 'info',
    inactive: 'secondary',
  },
}

function toneToClass(tone: Tone | 'warn' | 'info' | 'purple') {
  if (tone === 'warn') return warnClass
  if (tone === 'info') return infoClass
  if (tone === 'purple') return purpleClass
  return toneClass[tone]
}

export function StatusBadge({
  domain,
  value,
  className,
}: {
  domain: keyof typeof tones
  value: string
  className?: string
}) {
  const { t } = useTranslation()
  const tone = tones[domain]?.[value] ?? 'secondary'
  const label = t(`${domain}.status.${value}`, {
    defaultValue: value.replace(/_/g, ' '),
  })
  return (
    <Badge
      variant='outline'
      className={cn(
        'px-1.5 font-normal capitalize',
        toneToClass(tone),
        className
      )}
    >
      {label}
    </Badge>
  )
}

/** Non-status enum badge (entitlement type, wallet source, order kind...). */
export function EnumBadge({
  domain,
  kind,
  value,
  className,
}: {
  domain: string
  kind: string
  value: string
  className?: string
}) {
  const { t } = useTranslation()
  const label = t(`${domain}.${kind}.${value}`, {
    defaultValue: value.replace(/_/g, ' '),
  })
  return (
    <Badge variant='secondary' className={cn('px-1.5 font-normal', className)}>
      {label}
    </Badge>
  )
}
