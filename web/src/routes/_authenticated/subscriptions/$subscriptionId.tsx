import { createFileRoute } from '@tanstack/react-router'
import { SubscriptionDetail } from '@/features/subscriptions/subscription-detail'

export const Route = createFileRoute(
  '/_authenticated/subscriptions/$subscriptionId'
)({
  component: SubscriptionDetail,
})
