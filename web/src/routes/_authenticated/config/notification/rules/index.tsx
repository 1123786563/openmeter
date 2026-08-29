import { createFileRoute } from '@tanstack/react-router'
import { NotificationRulesPage } from '@/features/config/notification/rules'

export const Route = createFileRoute(
  '/_authenticated/config/notification/rules/'
)({
  component: NotificationRulesPage,
})
