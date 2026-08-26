import { createFileRoute } from '@tanstack/react-router'
import { OrderDetail } from '@/features/commerce/order-detail'

export const Route = createFileRoute(
  '/_authenticated/commerce/orders/$orderId'
)({
  component: OrderDetail,
})
