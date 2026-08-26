import { createFileRoute } from '@tanstack/react-router'
import { PlaceholderPage } from '@/components/placeholder-page'

export const Route = createFileRoute('/_authenticated/config/portal-tokens/')({
  component: () => (
    <PlaceholderPage
      titleKey='config.portalTokens.title'
      descriptionKey='config.portalTokens.description'
    />
  ),
})
