import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute(
  '/_authenticated/config/billing-profiles/'
)({
  component: () => (
    <PlaceholderPage
      titleKey='config.billingProfiles.title'
      descriptionKey='config.billingProfiles.description'
    />
  ),
})
