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

/** POST /api/v1/plans/{planIdOrKey}/next — clone the latest published version into a new draft. */
export interface LegacyPlan {
  id: string
  name: string
  key: string
  version: number
  currency: string
  billingCadence: string
  status: string
  createdAt: string
  updatedAt: string
}

export async function clonePlanNextVersion(
  planIdOrKey: string
): Promise<LegacyPlan> {
  return apiFetch<LegacyPlan>(
    `/v1/plans/${encodeURIComponent(planIdOrKey)}/next`,
    {
      method: 'POST',
    }
  )
}

/* ------------------------------------------------------------------ */
/* Notifications (v1) — channels                                       */
/* ------------------------------------------------------------------ */

/**
 * GET /api/v1/notification/channels — spec only defines the WEBHOOK variant
 * (NotificationChannelWebhook), so the hand-written type is not a union.
 */
export interface NotificationChannel {
  id: string
  type: 'WEBHOOK'
  name: string
  url: string
  disabled: boolean
  customHeaders?: Record<string, string>
  signingSecret?: string
  metadata?: Record<string, string> | null
  createdAt: string
  updatedAt: string
  deletedAt?: string | null
}

export interface NotificationChannelPaginatedResponse {
  totalCount: number
  page: number
  pageSize: number
  items: NotificationChannel[]
}

export interface NotificationChannelListParams {
  includeDeleted?: boolean
  includeDisabled?: boolean
  page?: number
  pageSize?: number
}

export async function listNotificationChannels(
  params: NotificationChannelListParams = {}
): Promise<NotificationChannelPaginatedResponse> {
  const search = new URLSearchParams()
  if (params.includeDeleted) search.set('includeDeleted', 'true')
  // The endpoint hides disabled channels unless asked; the admin view needs them.
  if (params.includeDisabled !== undefined) {
    search.set('includeDisabled', String(params.includeDisabled))
  }
  if (params.page) search.set('page', String(params.page))
  if (params.pageSize) search.set('pageSize', String(params.pageSize))
  const qs = search.toString()
  return apiFetch<NotificationChannelPaginatedResponse>(
    `/v1/notification/channels${qs ? `?${qs}` : ''}`
  )
}

/** POST /api/v1/notification/channels — body mirrors NotificationChannelWebhookCreateRequest. */
export interface NotificationChannelCreateRequest {
  type: 'WEBHOOK'
  name: string
  url: string
  disabled?: boolean
  customHeaders?: Record<string, string>
  signingSecret?: string
  metadata?: Record<string, string>
}

export async function createNotificationChannel(
  body: NotificationChannelCreateRequest
): Promise<NotificationChannel> {
  return apiFetch<NotificationChannel>('/v1/notification/channels', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

/**
 * PUT is a full replacement (same body shape as create). Omitting
 * `signingSecret` clears it server-side, so callers must backfill the current
 * value when only flipping `disabled` or editing other fields.
 */
export async function updateNotificationChannel(
  channelId: string,
  body: NotificationChannelCreateRequest
): Promise<NotificationChannel> {
  return apiFetch<NotificationChannel>(
    `/v1/notification/channels/${encodeURIComponent(channelId)}`,
    { method: 'PUT', body: JSON.stringify(body) }
  )
}

/** Soft delete; once deleted a channel cannot be undeleted. */
export async function deleteNotificationChannel(
  channelId: string
): Promise<void> {
  return apiFetch<void>(
    `/v1/notification/channels/${encodeURIComponent(channelId)}`,
    { method: 'DELETE' }
  )
}

/** GET /api/v1/info/currencies — fiat currency list (v1-only lookup endpoint). */
export interface FiatCurrency {
  code: string
  name: string
  symbol: string
  subunits: number
}

export async function listFiatCurrencies(): Promise<FiatCurrency[]> {
  return apiFetch<FiatCurrency[]>('/v1/info/currencies')
}

/* ------------------------------------------------------------------ */
/* Portal tokens (v1)                                                  */
/* ------------------------------------------------------------------ */

/** GET/POST /api/v1/portal/tokens 响应（api/openapi.yaml PortalToken，camelCase）。 */
export interface PortalToken {
  id: string
  subject: string
  expiresAt?: string
  expired?: boolean
  createdAt?: string
  /** 仅创建响应携带的一次性明文（om_portal_ 前缀）；列表响应不含此字段。 */
  token?: string
  allowedMeterSlugs?: string[]
}

/**
 * POST /api/v1/portal/tokens — 发放 consumer portal token。
 * 可写字段仅 subject（必填）与 allowedMeterSlugs（可选，缺省=全部 meter）。
 */
export async function createPortalToken(body: {
  subject: string
  allowedMeterSlugs?: string[]
}): Promise<PortalToken> {
  return apiFetch<PortalToken>('/v1/portal/tokens', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}
