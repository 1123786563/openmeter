import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from '@tanstack/react-query'
import { api } from '@/api/client'
import {
  clonePlanNextVersion,
  createNotificationChannel,
  getCustomerEntitlementValueV2,
  listCustomerEntitlementsV2,
  listNotificationChannels,
  listSubjects,
  type EntitlementValueV2,
  type EntitlementV2,
  type NotificationChannelCreateRequest,
  type Subject,
} from '@/api/legacy'
import { queryKeys } from '@/api/query-keys'
import { useNamespaceStore } from '@/stores/namespace-store'

/* ------------------------------------------------------------------ */
/* Namespaces                                                          */
/* ------------------------------------------------------------------ */

export function useNamespaces() {
  return useQuery({
    queryKey: queryKeys.namespaces(),
    queryFn: ({ signal }) => api.namespaces.list({}, { signal }),
  })
}

/* ------------------------------------------------------------------ */
/* Customers                                                           */
/* ------------------------------------------------------------------ */

export interface CustomerListParams {
  page: number
  pageSize: number
  search?: string
}

export function useCustomers(params: CustomerListParams) {
  return useQuery({
    queryKey: queryKeys.customers(params),
    queryFn: ({ signal }) =>
      api.customers.list(
        {
          page: { number: params.page, size: params.pageSize },
          // Field-level filters AND together (no cross-field OR in the v3
          // filter language), so search matches on the display name only.
          filter: params.search
            ? { name: { contains: params.search } }
            : undefined,
        },
        { signal }
      ),
  })
}

export function useCustomer(customerId: string) {
  return useQuery({
    queryKey: queryKeys.customer(customerId),
    queryFn: ({ signal }) => api.customers.get({ customerId }, { signal }),
    enabled: Boolean(customerId),
  })
}

export function useCreateCustomer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.customers.create>[0]) =>
      api.customers.create(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('customers') })
    },
  })
}

export function useUpdateCustomer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.customers.upsert>[0]) =>
      api.customers.upsert(input),
    onSuccess: (customer) => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('customers') })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.customer(customer.id),
      })
    },
  })
}

/* ------------------------------------------------------------------ */
/* Subscriptions & plans                                               */
/* ------------------------------------------------------------------ */

export interface SubscriptionListParams {
  page: number
  pageSize: number
  customerId?: string
  status?: string
}

export function useSubscriptions(params: SubscriptionListParams) {
  return useQuery({
    queryKey: queryKeys.subscriptions(params),
    queryFn: ({ signal }) =>
      api.subscriptions.list(
        {
          page: { number: params.page, size: params.pageSize },
          // Supported sort attributes: id, active_from, active_to.
          sort: { by: 'active_from', order: 'desc' },
          filter: {
            ...(params.customerId ? { customerId: params.customerId } : {}),
            ...(params.status ? { status: params.status } : {}),
          },
        },
        { signal }
      ),
  })
}

export function useSubscription(subscriptionId: string) {
  return useQuery({
    queryKey: queryKeys.subscription(subscriptionId),
    queryFn: ({ signal }) =>
      api.subscriptions.get({ subscriptionId }, { signal }),
    enabled: Boolean(subscriptionId),
  })
}

export function useCancelSubscription() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.subscriptions.cancel>[0]) =>
      api.subscriptions.cancel(input),
    onSuccess: (subscription) => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('subscriptions'),
      })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.subscription(subscription.id),
      })
    },
  })
}

export function useCreateSubscription() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.subscriptions.create>[0]) =>
      api.subscriptions.create(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('subscriptions'),
      })
    },
  })
}

/** All plans for the create-subscription wizard (plans are a small set). */
export function usePlans() {
  return useQuery({
    queryKey: queryKeys.plans(),
    queryFn: async ({ signal }) => {
      const plans = []
      for await (const plan of api.plans.listAll({}, { signal })) {
        plans.push(plan)
      }
      return plans
    },
  })
}

export interface PlanListParams {
  page: number
  pageSize: number
  status?: 'draft' | 'active' | 'archived' | 'scheduled'
}

/** Admin paginated plan list (the existing `usePlans` stays for the subscription wizard's listAll). */
export function usePlansPage(params: PlanListParams) {
  return useQuery({
    queryKey: queryKeys.plansPage(params),
    queryFn: ({ signal }) =>
      api.plans.list(
        {
          page: { number: params.page, size: params.pageSize },
          sort: { by: 'created_at', order: 'desc' },
          filter: params.status ? { status: params.status } : undefined,
        },
        { signal }
      ),
  })
}

export function usePlan(planId: string) {
  return useQuery({
    queryKey: queryKeys.plan(planId),
    queryFn: ({ signal }) => api.plans.get({ planId }, { signal }),
    enabled: Boolean(planId),
  })
}

export function useCreatePlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.plans.create>[0]) =>
      api.plans.create(input),
    onSuccess: (plan) => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans') })
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans-page') })
      void queryClient.invalidateQueries({ queryKey: queryKeys.plan(plan.id) })
    },
  })
}

export function usePublishPlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.plans.publish>[0]) =>
      api.plans.publish(input),
    onSuccess: (plan) => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans') })
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans-page') })
      void queryClient.invalidateQueries({ queryKey: queryKeys.plan(plan.id) })
    },
  })
}

export function useArchivePlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.plans.archive>[0]) =>
      api.plans.archive(input),
    onSuccess: (plan) => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans') })
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans-page') })
      void queryClient.invalidateQueries({ queryKey: queryKeys.plan(plan.id) })
    },
  })
}

export function useClonePlanNext() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ planIdOrKey }: { planIdOrKey: string }) =>
      clonePlanNextVersion(planIdOrKey),
    onSuccess: (plan) => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans') })
      void queryClient.invalidateQueries({ queryKey: nsPrefix('plans-page') })
      // v1 id 是 v3 同一资源 id；新 draft 详情可直接用它进入。
      void queryClient.invalidateQueries({ queryKey: queryKeys.plan(plan.id) })
    },
  })
}

/* ------------------------------------------------------------------ */
/* Invoices                                                            */
/* ------------------------------------------------------------------ */

export interface InvoiceListParams {
  page: number
  pageSize: number
  customerId?: string
  status?: string
}

export function useInvoices(params: InvoiceListParams) {
  return useQuery({
    queryKey: queryKeys.invoices(params),
    queryFn: ({ signal }) =>
      api.internal.invoices.list(
        {
          page: { number: params.page, size: params.pageSize },
          sort: { by: 'created_at', order: 'desc' },
          filter: {
            ...(params.customerId ? { customerId: params.customerId } : {}),
            ...(params.status ? { status: params.status } : {}),
          },
        },
        { signal }
      ),
  })
}

export function useInvoice(invoiceId: string) {
  return useQuery({
    queryKey: queryKeys.invoice(invoiceId),
    queryFn: ({ signal }) =>
      api.internal.invoices.get({ invoiceId }, { signal }),
    enabled: Boolean(invoiceId),
  })
}

export type InvoiceAction =
  | 'advance'
  | 'approve'
  | 'retry'
  | 'snapshotQuantities'

export function useInvoiceAction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      invoiceId,
      action,
    }: {
      invoiceId: string
      action: InvoiceAction
    }) => {
      const invoices = api.internal.invoices
      switch (action) {
        case 'advance':
          return invoices.advance({ invoiceId })
        case 'approve':
          return invoices.approve({ invoiceId })
        case 'retry':
          return invoices.retry({ invoiceId })
        case 'snapshotQuantities':
          return invoices.snapshotQuantities({ invoiceId })
      }
    },
    onSuccess: (invoice) => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('invoices') })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.invoice(invoice.id),
      })
    },
  })
}

/* ------------------------------------------------------------------ */
/* Meters & events                                                     */
/* ------------------------------------------------------------------ */

export function useMeters() {
  return useQuery({
    queryKey: queryKeys.meters(),
    queryFn: ({ signal }) =>
      api.meters.list({ page: { number: 1, size: 100 } }, { signal }),
  })
}

export function useMeter(meterId: string) {
  return useQuery({
    queryKey: queryKeys.meter(meterId),
    queryFn: ({ signal }) => api.meters.get({ meterId }, { signal }),
    enabled: Boolean(meterId),
  })
}

export interface MeterQueryParams {
  meterId: string
  from?: Date
  to?: Date
  subject?: string
}

export function useMeterQuery(params: MeterQueryParams) {
  return useQuery({
    queryKey: queryKeys.meterQuery(params.meterId, {
      from: params.from?.toISOString(),
      to: params.to?.toISOString(),
      subject: params.subject,
    }),
    queryFn: ({ signal }) =>
      api.meters.query(
        {
          meterId: params.meterId,
          body: {
            from: params.from,
            to: params.to,
            timeZone: 'UTC',
            filters: params.subject
              ? { dimensions: { subject: { eq: params.subject } } }
              : undefined,
          },
        },
        { signal }
      ),
    enabled: Boolean(params.meterId),
  })
}

export interface EventListParams {
  subject?: string
  from?: Date
  to?: Date
  after?: string
}

export function useEvents(params: EventListParams) {
  return useQuery({
    queryKey: queryKeys.events({
      subject: params.subject,
      from: params.from?.toISOString(),
      to: params.to?.toISOString(),
      after: params.after,
    }),
    queryFn: ({ signal }) =>
      api.events.list(
        {
          page: { after: params.after, size: 50 },
          sort: { by: 'time', order: 'desc' },
          filter: {
            ...(params.subject ? { subject: { eq: params.subject } } : {}),
            ...(params.from || params.to
              ? {
                  time: {
                    ...(params.from ? { gte: params.from } : {}),
                    ...(params.to ? { lte: params.to } : {}),
                  },
                }
              : {}),
          },
        },
        { signal }
      ),
  })
}

export function useSubjects() {
  return useQuery({
    queryKey: queryKeys.subjects(),
    queryFn: () => listSubjects(),
    select: (subjects: Subject[]) =>
      [...subjects].sort((a, b) => a.key.localeCompare(b.key)),
  })
}

/* ------------------------------------------------------------------ */
/* Credits                                                             */
/* ------------------------------------------------------------------ */

export function useCreditBalance(customerId: string) {
  return useQuery({
    queryKey: queryKeys.creditBalance(customerId),
    queryFn: ({ signal }) =>
      api.customers.credits.balance.get({ customerId }, { signal }),
    enabled: Boolean(customerId),
  })
}

export interface CreditGrantListParams {
  customerId: string
  page: number
  pageSize: number
}

export function useCreditGrants(params: CreditGrantListParams) {
  return useQuery({
    queryKey: queryKeys.creditGrants(params.customerId, {
      page: params.page,
      pageSize: params.pageSize,
    }),
    queryFn: ({ signal }) =>
      api.customers.credits.grants.list(
        {
          customerId: params.customerId,
          page: { number: params.page, size: params.pageSize },
        },
        { signal }
      ),
    enabled: Boolean(params.customerId),
  })
}

export function useCreateCreditGrant() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      customerId,
      body,
    }: {
      customerId: string
      body: Parameters<typeof api.customers.credits.grants.create>[0]['body']
    }) => api.customers.credits.grants.create({ customerId, body }),
    onSuccess: (_grant, { customerId }) => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.creditGrants(customerId),
      })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.creditBalance(customerId),
      })
    },
  })
}

export function useCreditTransactions(customerId: string) {
  return useQuery({
    queryKey: queryKeys.creditTransactions(customerId),
    queryFn: ({ signal }) =>
      api.customers.credits.transactions.list(
        { customerId, page: { size: 50 } },
        { signal }
      ),
    enabled: Boolean(customerId),
  })
}

export function useCustomerEntitlements(
  customerId: string
): UseQueryResult<EntitlementV2[]> {
  return useQuery({
    queryKey: queryKeys.customerEntitlements(customerId),
    queryFn: () => listCustomerEntitlementsV2(customerId),
    select: (res) => res.items,
    enabled: Boolean(customerId),
  })
}

export function useCustomerEntitlementValue(
  customerId: string,
  entitlementId: string,
  featureKey: string
): UseQueryResult<EntitlementValueV2> {
  return useQuery({
    queryKey: [
      ...queryKeys.customerEntitlementValue(customerId, entitlementId),
      featureKey,
    ],
    queryFn: () =>
      getCustomerEntitlementValueV2(customerId, featureKey || entitlementId),
    enabled: Boolean(customerId) && Boolean(entitlementId),
  })
}

/* ------------------------------------------------------------------ */
/* Commerce (fork-specific)                                            */
/* ------------------------------------------------------------------ */

export function useRechargeProducts() {
  return useQuery({
    queryKey: queryKeys.rechargeProducts({ includeInactive: true }),
    // The admin catalog view requests delisted products too so they can be
    // relisted; the default (flag omitted) keeps the customer-facing
    // active-only behavior.
    queryFn: ({ signal }) =>
      api.commerce.listRechargeProducts(
        { includeInactive: true },
        { signal }
      ),
  })
}

export function useCreateRechargeProduct() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (
      input: Parameters<typeof api.commerce.createRechargeProduct>[0]
    ) => api.commerce.createRechargeProduct(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('recharge-products'),
      })
    },
  })
}

export function useUpdateRechargeProduct() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (
      input: Parameters<typeof api.commerce.updateRechargeProduct>[0]
    ) => api.commerce.updateRechargeProduct(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('recharge-products'),
      })
    },
  })
}

export interface OrderListParams {
  page: number
  pageSize: number
  customerId?: string
  status?:
    | 'created'
    | 'awaiting_payment'
    | 'paid'
    | 'fulfilled'
    | 'cancelled'
    | 'expired'
    | 'refund_pending'
    | 'partially_refunded'
    | 'refunded'
}

export function useOrders(params: OrderListParams) {
  return useQuery({
    queryKey: queryKeys.orders(params),
    queryFn: ({ signal }) =>
      api.commerce.listOrders(
        {
          page: { number: params.page, size: params.pageSize },
          ...(params.customerId ? { customerId: params.customerId } : {}),
          ...(params.status ? { status: params.status } : {}),
        },
        { signal }
      ),
  })
}

export interface RefundListParams {
  page: number
  pageSize: number
  customerId?: string
  status?:
    | 'pending_fence'
    | 'provider_processing'
    | 'ledger_reversing'
    | 'fulfilled'
    | 'failed'
}

export function useRefunds(params: RefundListParams) {
  return useQuery({
    queryKey: queryKeys.refunds(params),
    queryFn: ({ signal }) =>
      api.commerce.listRefunds(
        {
          page: { number: params.page, size: params.pageSize },
          ...(params.customerId ? { customerId: params.customerId } : {}),
          ...(params.status ? { status: params.status } : {}),
        },
        { signal }
      ),
  })
}

export function useOrder(orderId: string) {
  return useQuery({
    queryKey: queryKeys.order(orderId),
    queryFn: ({ signal }) => api.commerce.getOrder({ orderId }, { signal }),
    enabled: Boolean(orderId),
  })
}

export function useRefund(refundId: string) {
  return useQuery({
    queryKey: queryKeys.refund(refundId),
    queryFn: ({ signal }) => api.commerce.getRefund({ refundId }, { signal }),
    enabled: Boolean(refundId),
  })
}

export function useCustomerWallet(customerId: string) {
  return useQuery({
    queryKey: queryKeys.customerWallet(customerId),
    queryFn: ({ signal }) =>
      api.commerce.getCustomerWallet({ customerId }, { signal }),
    enabled: Boolean(customerId),
  })
}

/* ------------------------------------------------------------------ */
/* Features                                                            */
/* ------------------------------------------------------------------ */

export interface FeatureListParams {
  page: number
  pageSize: number
  search?: string
}

export function useFeatures(params: FeatureListParams) {
  return useQuery({
    queryKey: queryKeys.features(params),
    queryFn: ({ signal }) =>
      api.features.list(
        {
          page: { number: params.page, size: params.pageSize },
          sort: { by: 'created_at', order: 'desc' },
          filter: params.search
            ? { name: { contains: params.search } }
            : undefined,
        },
        { signal }
      ),
  })
}

export function useFeature(featureId: string) {
  return useQuery({
    queryKey: queryKeys.feature(featureId),
    queryFn: ({ signal }) => api.features.get({ featureId }, { signal }),
    enabled: Boolean(featureId),
  })
}

/** All features for pickers (wizard rate cards, addon rate cards). */
export function useAllFeatures() {
  return useQuery({
    queryKey: queryKeys.features({ all: true }),
    queryFn: async ({ signal }) => {
      const features = []
      for await (const feature of api.features.listAll({}, { signal })) {
        features.push(feature)
      }
      return features
    },
  })
}

export function useCreateFeature() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.features.create>[0]) =>
      api.features.create(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('features') })
    },
  })
}

/**
 * v3 PATCH update only accepts unit_cost (UpdateFeatureRequest); name, key,
 * and description have no update endpoint.
 */
export function useUpdateFeature() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.features.update>[0]) =>
      api.features.update(input),
    onSuccess: (feature) => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('features') })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.feature(feature.id),
      })
    },
  })
}

export function useDeleteFeature() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.features.delete>[0]) =>
      api.features.delete(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: nsPrefix('features') })
    },
  })
}

export interface FeatureCostQueryParams {
  customerId?: string
  from?: Date
  to?: Date
}

/**
 * Feature cost query (POST /features/{id}/cost/query). Body is a
 * MeterQueryRequest; customer filtering uses the reserved `customer_id`
 * dimension which only supports eq/in comparisons.
 */
export function useFeatureCostQuery(
  featureId: string,
  params: FeatureCostQueryParams,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: queryKeys.featureCostQuery(featureId, {
      customerId: params.customerId,
      from: params.from?.toISOString(),
      to: params.to?.toISOString(),
    }),
    queryFn: ({ signal }) =>
      api.features.queryCost(
        {
          featureId,
          body: {
            from: params.from,
            to: params.to,
            timeZone: 'UTC',
            filters: params.customerId
              ? { dimensions: { customer_id: { eq: params.customerId } } }
              : undefined,
          },
        },
        { signal }
      ),
    enabled: Boolean(featureId) && (options?.enabled ?? true),
  })
}

/* ------------------------------------------------------------------ */
/* Notification channels (v1)                                          */
/* ------------------------------------------------------------------ */

export interface NotificationChannelsParams {
  page: number
  pageSize: number
}

export function useNotificationChannels(params: NotificationChannelsParams) {
  return useQuery({
    queryKey: queryKeys.notificationChannels(params),
    queryFn: () =>
      listNotificationChannels({ ...params, includeDisabled: true }),
  })
}

export function useCreateChannel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: NotificationChannelCreateRequest) =>
      createNotificationChannel(body),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: nsPrefix('notification.channels'),
      })
    },
  })
}

/* ------------------------------------------------------------------ */
/* Receivable periods & external invoices (v3 commerce)                */
/* ------------------------------------------------------------------ */

export function useReceivablePeriods(customerId: string, after?: string) {
  return useQuery({
    queryKey: queryKeys.receivablePeriods(customerId, { after: after ?? null }),
    queryFn: ({ signal }) =>
      api.commerce.listReceivablePeriods(
        { customerId, page: { after, size: 20 } },
        { signal }
      ),
    enabled: Boolean(customerId),
  })
}

export function useUpdateExternalInvoice(customerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      periodId,
      body,
    }: {
      periodId: string
      body: Parameters<typeof api.commerce.updateExternalInvoice>[0]['body']
    }) =>
      api.commerce.updateExternalInvoice({ customerId, periodId, body }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.receivablePeriods(customerId),
      })
    },
  })
}

/* ------------------------------------------------------------------ */
/* Helpers                                                             */
/* ------------------------------------------------------------------ */

/** Prefix that scopes invalidation to the active namespace cache branch. */
function nsPrefix(domain: string): [string, string, string] {
  return ['api', useNamespaceStore.getState().currentNamespace ?? '', domain]
}
