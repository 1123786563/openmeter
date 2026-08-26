import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute('/_authenticated/config/tax-codes/')({
  component: () => (
    <PlaceholderPage
      titleKey='config.taxCodes.title'
      descriptionKey='config.taxCodes.description'
    />
  ),
})
