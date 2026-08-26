import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute('/_authenticated/config/plans/')({
  component: () => (
    <PlaceholderPage
      titleKey='config.plans.title'
      descriptionKey='config.plans.description'
    />
  ),
})
