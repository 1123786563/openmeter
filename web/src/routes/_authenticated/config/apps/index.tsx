import { createFileRoute } from '@tanstack/react-router'
import { AppsPage } from '@/features/config/apps'

export const Route = createFileRoute('/_authenticated/config/apps/')({
  component: AppsPage,
})
