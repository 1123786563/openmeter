import { useNamespaceStore } from '@/stores/namespace-store'

/**
 * Query key factory. Every key embeds the active namespace so switching
 * namespaces gives each namespace its own cache entry; the switcher
 * additionally invalidates everything for a hard refetch.
 */
export function ns<T extends readonly unknown[]>(
  ...key: T
): [string, string, ...T] {
  const { currentNamespace } = useNamespaceStore.getState()
  return ['api', currentNamespace ?? '', ...key]
}

export const queryKeys = {
  namespaces: () => ns('namespaces'),
  customers: (params: object = {}) => ns('customers', params),
  customer: (id: string) => ns('customer', id),
  customerEntitlements: (id: string) => ns('customer-entitlements', id),
  customerEntitlementValue: (id: string, entitlementId: string) =>
    ns('customer-entitlement-value', id, entitlementId),
  subscriptions: (params: object = {}) => ns('subscriptions', params),
  subscription: (id: string) => ns('subscription', id),
  plans: () => ns('plans'),
  plansPage: (params: object = {}) => ns('plans-page', params),
  plan: (id: string) => ns('plan', id),
  invoices: (params: object = {}) => ns('invoices', params),
  invoice: (id: string) => ns('invoice', id),
  meters: () => ns('meters'),
  meter: (id: string) => ns('meter', id),
  meterQuery: (id: string, params: object) => ns('meter-query', id, params),
  events: (params: object) => ns('events', params),
  subjects: () => ns('subjects'),
  creditBalance: (customerId: string) => ns('credit-balance', customerId),
  creditGrants: (customerId: string, params: object = {}) =>
    ns('credit-grants', customerId, params),
  creditTransactions: (customerId: string) =>
    ns('credit-transactions', customerId),
  customerWallet: (customerId: string) => ns('customer-wallet', customerId),
  rechargeProducts: (params: object = {}) => ns('recharge-products', params),
  order: (id: string) => ns('order', id),
  orders: (params: object = {}) => ns('orders', params),
  refund: (id: string) => ns('refund', id),
  refunds: (params: object = {}) => ns('refunds', params),
  notificationChannels: (params: object = {}) =>
    ns('notification.channels', params),
  features: (params: object = {}) => ns('features', params),
  feature: (id: string) => ns('feature', id),
  featureCostQuery: (id: string, params: object) =>
    ns('feature-cost-query', id, params),
  apps: () => ns('apps'),
}
