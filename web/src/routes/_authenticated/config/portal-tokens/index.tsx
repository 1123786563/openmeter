import { createFileRoute } from '@tanstack/react-router'
import { PortalTokensPage } from '@/features/config/portal-tokens'

export const Route = createFileRoute('/_authenticated/config/portal-tokens/')({
  component: PortalTokensPage,
})
