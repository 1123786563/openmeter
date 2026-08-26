import { createFileRoute } from '@tanstack/react-router'
import { MetersPage } from '@/features/meters'

export const Route = createFileRoute('/_authenticated/meters/')({
  component: MetersPage,
})
