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

/* ------------------------------------------------------------------ */
/* Notifications (v1) — rules                                          */
/* ------------------------------------------------------------------ */

/** Spec oneOf discriminator value for both rules and rule create requests. */
export type NotificationRuleType =
  | 'entitlements.balance.threshold'
  | 'entitlements.reset'
  | 'invoice.created'
  | 'invoice.updated'

/** Channel reference embedded in rules (spec: NotificationChannelMeta). */
export interface NotificationChannelMeta {
  id: string
  type: 'WEBHOOK'
}

/** Feature reference embedded in rules (spec: FeatureMeta, id + key only). */
export interface FeatureMeta {
  id: string
  key: string
}

export interface NotificationRuleCommon {
  id: string
  name: string
  disabled: boolean
  channels: NotificationChannelMeta[]
  metadata?: Record<string, string> | null
  createdAt: string
  updatedAt: string
  deletedAt?: string | null
}

export interface NotificationRuleBalanceThreshold extends NotificationRuleCommon {
  type: 'entitlements.balance.threshold'
  thresholds: NotificationRuleThreshold[]
  features?: FeatureMeta[]
}

export interface NotificationRuleEntitlementReset extends NotificationRuleCommon {
  type: 'entitlements.reset'
  features?: FeatureMeta[]
}

export interface NotificationRuleInvoiceCreated extends NotificationRuleCommon {
  type: 'invoice.created'
}

export interface NotificationRuleInvoiceUpdated extends NotificationRuleCommon {
  type: 'invoice.updated'
}

/** Rule union discriminated by `type` (spec oneOf + discriminator). */
export type NotificationRule =
  | NotificationRuleBalanceThreshold
  | NotificationRuleEntitlementReset
  | NotificationRuleInvoiceCreated
  | NotificationRuleInvoiceUpdated

/** Spec: NotificationRuleBalanceThresholdValue — value + threshold type. */
export interface NotificationRuleThreshold {
  value: number
  type:
    | 'PERCENT'
    | 'NUMBER'
    | 'balance_value'
    | 'usage_percentage'
    | 'usage_value'
}

export interface NotificationRuleListParams {
  includeDeleted?: boolean
  includeDisabled?: boolean
  feature?: string[]
  channel?: string[]
  page?: number
  pageSize?: number
}

export interface NotificationRulePaginatedResponse {
  totalCount: number
  page: number
  pageSize: number
  items: NotificationRule[]
}

export async function listNotificationRules(
  params: NotificationRuleListParams = {}
): Promise<NotificationRulePaginatedResponse> {
  const search = new URLSearchParams()
  if (params.includeDeleted) search.set('includeDeleted', 'true')
  // The endpoint hides disabled rules unless asked; the admin view needs them.
  if (params.includeDisabled !== undefined) {
    search.set('includeDisabled', String(params.includeDisabled))
  }
  // Repeated array params per spec: ?feature=a&feature=b
  params.feature?.forEach((value) => search.append('feature', value))
  params.channel?.forEach((value) => search.append('channel', value))
  if (params.page) search.set('page', String(params.page))
  if (params.pageSize) search.set('pageSize', String(params.pageSize))
  const qs = search.toString()
  return apiFetch<NotificationRulePaginatedResponse>(
    `/v1/notification/rules${qs ? `?${qs}` : ''}`
  )
}

/** Spec: NotificationRuleCreateRequest — oneOf discriminated by `type`. */
export interface NotificationRuleCreateRequestBase {
  name: string
  disabled?: boolean
  /** Channel ids (spec: minItems 1). */
  channels: string[]
  metadata?: Record<string, string>
}

export interface NotificationRuleBalanceThresholdCreateRequest extends NotificationRuleCreateRequestBase {
  type: 'entitlements.balance.threshold'
  /** 1-10 thresholds (spec minItems/maxItems). */
  thresholds: NotificationRuleThreshold[]
  /** Optional scope by feature ids or keys. */
  features?: string[]
}

export interface NotificationRuleEntitlementResetCreateRequest extends NotificationRuleCreateRequestBase {
  type: 'entitlements.reset'
  features?: string[]
}

export interface NotificationRuleInvoiceCreatedCreateRequest extends NotificationRuleCreateRequestBase {
  type: 'invoice.created'
}

export interface NotificationRuleInvoiceUpdatedCreateRequest extends NotificationRuleCreateRequestBase {
  type: 'invoice.updated'
}

export type NotificationRuleCreateRequest =
  | NotificationRuleBalanceThresholdCreateRequest
  | NotificationRuleEntitlementResetCreateRequest
  | NotificationRuleInvoiceCreatedCreateRequest
  | NotificationRuleInvoiceUpdatedCreateRequest

export async function createNotificationRule(
  body: NotificationRuleCreateRequest
): Promise<NotificationRule> {
  return apiFetch<NotificationRule>('/v1/notification/rules', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

/** PUT is a full replacement: the complete oneOf create body, every time. */
export async function updateNotificationRule(
  ruleId: string,
  body: NotificationRuleCreateRequest
): Promise<NotificationRule> {
  return apiFetch<NotificationRule>(
    `/v1/notification/rules/${encodeURIComponent(ruleId)}`,
    { method: 'PUT', body: JSON.stringify(body) }
  )
}

/**
 * Rebuilds the full PUT body from a rule as stored, optionally overriding
 * `disabled`. Required because rule updates replace the whole resource —
 * toggling enabled/disabled must still send type-specific fields back.
 */
export function ruleToUpdateBody(
  rule: NotificationRule,
  disabled?: boolean
): NotificationRuleCreateRequest {
  const base = {
    name: rule.name,
    disabled: disabled ?? rule.disabled,
    channels: rule.channels.map((channel) => channel.id),
    ...(rule.metadata ? { metadata: rule.metadata } : {}),
  }
  switch (rule.type) {
    case 'entitlements.balance.threshold':
      return {
        ...base,
        type: rule.type,
        thresholds: rule.thresholds,
        ...(rule.features
          ? { features: rule.features.map((feature) => feature.id) }
          : {}),
      }
    case 'entitlements.reset':
      return {
        ...base,
        type: rule.type,
        ...(rule.features
          ? { features: rule.features.map((feature) => feature.id) }
          : {}),
      }
    case 'invoice.created':
    case 'invoice.updated':
      return { ...base, type: rule.type }
  }
}

/** Notification event type — same discriminator values as rule types. */
export type NotificationEventType =
  | 'entitlements.balance.threshold'
  | 'entitlements.reset'
  | 'invoice.created'
  | 'invoice.updated'

export type NotificationEventDeliveryState =
  | 'SUCCESS'
  | 'FAILED'
  | 'SENDING'
  | 'PENDING'
  | 'RESENDING'

/** Spec: EventDeliveryAttemptResponse — what the recipient answered. */
export interface EventDeliveryAttemptResponse {
  statusCode?: number
  body: string
  durationMs: number
}

export interface NotificationEventDeliveryAttempt {
  state: NotificationEventDeliveryState
  response: EventDeliveryAttemptResponse
  timestamp: string
}

export interface NotificationEventDeliveryStatus {
  state: NotificationEventDeliveryState
  reason: string
  updatedAt: string
  channel: NotificationChannelMeta
  nextAttempt?: string | null
  attempts: NotificationEventDeliveryAttempt[]
}

/**
 * Payload envelope shared by all four event types (spec oneOf discriminated by
 * `type`); `data` varies per type and stays opaque for UI purposes.
 */
export interface NotificationEventPayload {
  id: string
  type: NotificationEventType
  timestamp: string
  data: unknown
}

export interface NotificationEvent {
  id: string
  type: NotificationEventType
  createdAt: string
  rule: NotificationRule
  deliveryStatus: NotificationEventDeliveryStatus[]
  payload: NotificationEventPayload
}

/** POST /api/v1/notification/rules/{ruleId}/test — sends a test event with random data; 201 returns the generated event. */
export async function testNotificationRule(
  ruleId: string
): Promise<NotificationEvent> {
  return apiFetch<NotificationEvent>(
    `/v1/notification/rules/${encodeURIComponent(ruleId)}/test`,
    { method: 'POST' }
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

/** GET /api/v1/portal/tokens — 列表（spec 为裸数组，无分页包装）。limit 1-100。 */
export async function listPortalTokens(limit = 100): Promise<PortalToken[]> {
  return apiFetch<PortalToken[]>(`/v1/portal/tokens?limit=${limit}`)
}

/**
 * POST /api/v1/portal/tokens/invalidate — 按 id 或 subject 失效（二选一），
 * 204 无内容。管理端按行 id 失效。
 */
export async function invalidatePortalTokens(body: {
  id?: string
  subject?: string
}): Promise<void> {
  return apiFetch<void>('/v1/portal/tokens/invalidate', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}
