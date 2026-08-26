import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute(
  '/_authenticated/config/notification/rules/'
)({
  component: () => (
    <PlaceholderPage
      titleKey='config.notification.rules.title'
      descriptionKey='config.notification.rules.description'
    />
  ),
})
