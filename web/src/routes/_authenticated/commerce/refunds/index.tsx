import { createFileRoute } from '@tanstack/react-router'
import { RefundsPage } from '@/features/commerce/refunds'

export const Route = createFileRoute('/_authenticated/commerce/refunds/')({
  component: RefundsPage,
})
