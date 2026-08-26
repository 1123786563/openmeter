import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute('/_authenticated/config/features/')({
  component: () => (
    <PlaceholderPage
      titleKey='config.features.title'
      descriptionKey='config.features.description'
    />
  ),
})
