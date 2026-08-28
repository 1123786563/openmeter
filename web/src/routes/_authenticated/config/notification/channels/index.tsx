import { createFileRoute } from '@tanstack/react-router'
import { NotificationChannelsPage } from '@/features/config/notification/channels'

export const Route = createFileRoute(
  '/_authenticated/config/notification/channels/'
)({
  component: NotificationChannelsPage,
})
