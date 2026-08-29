import { createFileRoute } from '@tanstack/react-router'
import { TaxCodesPage } from '@/features/config/tax-codes'

export const Route = createFileRoute('/_authenticated/config/tax-codes/')({
  component: TaxCodesPage,
})
