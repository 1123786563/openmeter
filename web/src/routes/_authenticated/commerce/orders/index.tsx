import { createFileRoute } from '@tanstack/react-router'
import { OrdersPage } from '@/features/commerce/orders'

export const Route = createFileRoute('/_authenticated/commerce/orders/')({
  component: OrdersPage,
})
