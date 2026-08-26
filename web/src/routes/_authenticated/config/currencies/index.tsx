import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute('/_authenticated/config/currencies/')({
  component: () => (
    <PlaceholderPage
      titleKey='config.currencies.title'
      descriptionKey='config.currencies.description'
    />
  ),
})
