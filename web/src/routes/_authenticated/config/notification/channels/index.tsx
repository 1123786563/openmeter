import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute(
  '/_authenticated/config/notification/channels/'
)({
  component: () => (
    <PlaceholderPage
      titleKey='config.notification.channels.title'
      descriptionKey='config.notification.channels.description'
    />
  ),
})
