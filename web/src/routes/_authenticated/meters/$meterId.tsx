import { createFileRoute } from '@tanstack/react-router'
import { MeterDetail } from '@/features/meters/meter-detail'

export const Route = createFileRoute('/_authenticated/meters/$meterId')({
  component: MeterDetail,
})
