import { createFileRoute } from '@tanstack/react-router'
import { RefundDetail } from '@/features/commerce/refund-detail'

export const Route = createFileRoute(
  '/_authenticated/commerce/refunds/$refundId'
)({
  component: RefundDetail,
})
