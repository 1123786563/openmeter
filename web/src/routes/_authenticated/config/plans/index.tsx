import { createFileRoute } from '@tanstack/react-router'
import { PlansPage } from '@/features/config/plans'

export const Route = createFileRoute('/_authenticated/config/plans/')({
  component: PlansPage,
})
