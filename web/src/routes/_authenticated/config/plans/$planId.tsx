import { createFileRoute } from '@tanstack/react-router'
import { PlanDetail } from '@/features/config/plans/plan-detail'

export const Route = createFileRoute('/_authenticated/config/plans/$planId')({
  component: PlanDetail,
})
