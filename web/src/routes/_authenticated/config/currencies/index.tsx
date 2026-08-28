import { createFileRoute } from '@tanstack/react-router'
import { CurrenciesPage } from '@/features/config/currencies'

export const Route = createFileRoute('/_authenticated/config/currencies/')({
  component: CurrenciesPage,
})
