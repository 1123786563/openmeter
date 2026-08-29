import { createFileRoute } from '@tanstack/react-router'
import { NotificationEventsPage } from '@/features/config/notification/events'

export const Route = createFileRoute(
  '/_authenticated/config/notification/events/'
)({
  component: NotificationEventsPage,
})
