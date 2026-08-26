import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute('/_authenticated/config/addons/')({
  component: () => (
    <PlaceholderPage
      titleKey='config.addons.title'
      descriptionKey='config.addons.description'
    />
  ),
})
