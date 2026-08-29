import { createFileRoute } from '@tanstack/react-router'
import { BillingProfilesPage } from '@/features/config/billing-profiles'

export const Route = createFileRoute(
  '/_authenticated/config/billing-profiles/'
)({
  component: BillingProfilesPage,
})
