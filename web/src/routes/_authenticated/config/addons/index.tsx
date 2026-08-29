import { createFileRoute } from '@tanstack/react-router'
import { AddonsPage } from '@/features/config/addons'

export const Route = createFileRoute('/_authenticated/config/addons/')({
  component: AddonsPage,
})
