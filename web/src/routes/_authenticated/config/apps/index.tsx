import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute('/_authenticated/config/apps/')({
  component: () => (
    <PlaceholderPage
      titleKey='config.apps.title'
      descriptionKey='config.apps.description'
    />
  ),
})
