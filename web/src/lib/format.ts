import { formatDate } from 'date-fns'

/**
 * Presentation helpers shared across the admin pages. Decimal amounts arrive
 * as strings (API spec) and fen amounts as bigint (smallest currency unit).
 */

export function formatAmount(amount: string | number, currency?: string) {
  const value = typeof amount === 'number' ? amount : Number.parseFloat(amount)
  if (Number.isNaN(value)) return String(amount)
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: currency ?? 'USD',
      currencyDisplay: 'code',
    }).format(value)
  } catch {
    return `${amount} ${currency ?? ''}`.trim()
  }
}

/** Format a fen (smallest unit) bigint/number amount as a major currency unit. */
export function formatFen(amountFen: bigint | number, currency: string) {
  const fen = typeof amountFen === 'number' ? amountFen : Number(amountFen)
  return formatAmount(fen / 100, currency)
}

export function formatNumber(value: string | bigint | number | undefined) {
  if (value === undefined) return '—'
  if (typeof value === 'bigint') return value.toLocaleString()
  const num = typeof value === 'number' ? value : Number.parseFloat(value)
  return Number.isNaN(num) ? String(value) : num.toLocaleString()
}

export function formatDateTime(date: Date | string | undefined | null) {
  if (!date) return '—'
  const d = typeof date === 'string' ? new Date(date) : date
  if (Number.isNaN(d.getTime())) return '—'
  return formatDate(d, 'PPpp')
}

export function formatShortDateTime(date: Date | string | undefined | null) {
  if (!date) return '—'
  const d = typeof date === 'string' ? new Date(date) : date
  if (Number.isNaN(d.getTime())) return '—'
  return formatDate(d, 'PP p')
}
