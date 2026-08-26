import { createFileRoute } from '@tanstack/react-router'
import { RechargeProductsPage } from '@/features/commerce/recharge-products'

export const Route = createFileRoute(
  '/_authenticated/commerce/recharge-products/'
)({
  component: RechargeProductsPage,
})
