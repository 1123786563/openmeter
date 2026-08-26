import { apiFetch } from '@/lib/api'

/**
 * Hand-written endpoints for API surfaces that the generated v3 SDK does not
 * cover: v1 subjects and v2 customer entitlements (richer than v3's
 * entitlement-access which lacks balance/usage values).
 *
 * Field names mirror api/openapi.yaml exactly.
 */

/** GET /api/v1/subjects (deprecated upstream; still the only subject listing). */
export interface Subject {
  id: string
  key: string
  createdAt: string
  updatedAt: string
  deletedAt?: string | null
}

export async function listSubjects(): Promise<Subject[]> {
  return apiFetch<Subject[]>('/v1/subjects')
}

/** GET /api/v2/customers/{customerIdOrKey}/entitlements */
export interface EntitlementV2Base {
  id: string
  type: 'metered' | 'static' | 'boolean'
  featureKey: string
  featureId: string
  createdAt: string
  updatedAt: string
  deletedAt?: string | null
  activeFrom: string
  activeTo?: string | null
  currentUsagePeriod: { from: string; to: string }
  usagePeriod: { from: string; to: string }
  customerId: string
}

export interface EntitlementMeteredV2 extends EntitlementV2Base {
  type: 'metered'
  isSoftLimit: boolean
  lastReset: string
}

export interface EntitlementStaticV2 extends EntitlementV2Base {
  type: 'static'
  config: string
}

export interface EntitlementBooleanV2 extends EntitlementV2Base {
  type: 'boolean'
}

export type EntitlementV2 =
  | EntitlementMeteredV2
  | EntitlementStaticV2
  | EntitlementBooleanV2

export interface EntitlementV2PaginatedResponse {
  totalCount: number
  page: number
  pageSize: number
  items: EntitlementV2[]
}

export async function listCustomerEntitlementsV2(
  customerId: string
): Promise<EntitlementV2PaginatedResponse> {
  return apiFetch<EntitlementV2PaginatedResponse>(
    `/v2/customers/${encodeURIComponent(customerId)}/entitlements?pageSize=100`
  )
}

/** GET /api/v2/customers/{customerIdOrKey}/entitlements/{idOrFeatureKey}/value */
export interface EntitlementValueV2 {
  hasAccess: boolean
  balance?: number
  usage?: number
  overage?: number
  totalAvailableGrantAmount?: number
  config?: string
}

export async function getCustomerEntitlementValueV2(
  customerId: string,
  entitlementIdOrFeatureKey: string
): Promise<EntitlementValueV2> {
  return apiFetch<EntitlementValueV2>(
    `/v2/customers/${encodeURIComponent(customerId)}/entitlements/${encodeURIComponent(entitlementIdOrFeatureKey)}/value`
  )
}
