import { createFileRoute } from '@tanstack/react-router'
import { FeatureDetailPage } from '@/features/config/features/feature-detail'

export const Route = createFileRoute(
  '/_authenticated/config/features/$featureId'
)({
  component: RouteComponent,
})

// eslint-disable-next-line react-refresh/only-export-components
function RouteComponent() {
  const { featureId } = Route.useParams()
  return <FeatureDetailPage featureId={featureId} />
}
