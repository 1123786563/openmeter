import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute(
  '/_authenticated/config/notification/events/'
)({
  component: () => (
    <PlaceholderPage
      titleKey='config.notification.events.title'
      descriptionKey='config.notification.events.description'
    />
  ),
})
