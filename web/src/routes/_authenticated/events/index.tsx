import { createFileRoute } from '@tanstack/react-router'
import { EventsPage } from '@/features/events'

export const Route = createFileRoute('/_authenticated/events/')({
  component: EventsPage,
})
