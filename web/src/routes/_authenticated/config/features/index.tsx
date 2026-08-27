import { createFileRoute } from '@tanstack/react-router'
import { FeaturesPage } from '@/features/config/features'

export const Route = createFileRoute('/_authenticated/config/features/')({
  component: FeaturesPage,
})
