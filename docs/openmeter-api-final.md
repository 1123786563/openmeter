# OpenMeter API 对接清单（按版本与资源对象）

规则：比较时按 `method + path` 去重，按以下前缀归一化后再判断重复：`/openmeter/...`、`/api/v1/...`、`/api/v2/...` 的前缀视为同一业务路径前缀。
- `GET /api/v3/openmeter/addons` 与 `GET /api/v1/addons` 归一化为 `GET /addons`
- `GET /api/v3/openmeter/events` 与 `GET /api/v2/events` 归一化为 `GET /events`
- v3 覆盖时优先保留：如发现同方法同归一化路径存在 v3 实现，优先保留 v3，不再保留 v1/v2。

最终接口数：252（v3：119，v1：119，v2：14）

## 目录
- [版本 v3](#v3)
  - [customer（客户）](#v3-customer)（24）
  - [plan（套餐）](#v3-plan)（19）
  - [subscription（订阅）](#v3-subscription)（9）
  - [meter（计量）](#v3-meter)（6）
  - [event（事件）](#v3-event)（2）
  - [invoice（账单）](#v3-invoice)（14）
  - [app（应用）](#v3-app)（7）
  - [feature（特性）](#v3-feature)（6）
  - [system（系统）](#v3-system)（31）
- [版本 v1](#v1)
  - [customer（客户）](#v1-customer)（12）
  - [plan（套餐）](#v1-plan)（24）
  - [subscription（订阅）](#v1-subscription)（13）
  - [entitlement（权限/权益）](#v1-entitlement)（3）
  - [meter（计量）](#v1-meter)（8）
  - [invoice（账单）](#v1-invoice)（22）
  - [app（应用）](#v1-app)（15）
  - [notification（通知）](#v1-notification)（14）
  - [portal（门户）](#v1-portal)（3）
  - [system（系统）](#v1-system)（3）
- [版本 v2](#v2)
  - [entitlement（权限/权益）](#v2-entitlement)（13）

## v3

### customer（客户）

#### POST /customers/{customerId}/offline-payments
- version：v3
- operationId：create-offline-payment
- 摘要：Create offline payment
- 说明：Record an offline payment (bank transfer, enterprise remittance) for a customer.
The payment is held for reconciliation before being applied to a receivable
period.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CommerceOfflinePaymentCreate
- 响应码：201、400、401、403、409

#### GET /customers/{customerId}/receivable-periods
- version：v3
- operationId：list-receivable-periods
- 摘要：List receivable periods
- 说明：List receivable periods for a customer.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **page**（必填：否）：#/components/schemas/CursorPaginationQueryPage，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /customers/{customerId}/receivable-periods/{periodId}/external-invoice
- version：v3
- operationId：update-external-invoice
- 摘要：Update external invoice
- 说明：Attach or update an external invoice reference on a receivable period.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **periodId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CommerceExternalInvoiceUpdate
- 响应码：200、400、401、403、404

#### GET /customers/{customerId}/wallet
- version：v3
- operationId：get-customer-wallet
- 摘要：Get customer wallet
- 说明：Get a customer's Wallet view, including all credit buckets and recent
transactions.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/customers
- version：v3
- operationId：list-customers
- 摘要：List customers
- 说明：—
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort customers returned in the response. Supported sort attributes are:

- `id`
- `name` (default)
- `created_at`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **filter**（必填：否）：#/components/schemas/ListCustomersParamsFilter，Filter customers returned in the response.

To filter customers by key add the following query param: filter[key]=my-db-id
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/customers
- version：v3
- operationId：create-customer
- 摘要：Create customer
- 说明：—
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateCustomerRequest
- 响应码：201、400、401、403

#### GET /openmeter/customers/{customerId}
- version：v3
- operationId：get-customer
- 摘要：Get customer
- 说明：—
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/customers/{customerId}
- version：v3
- operationId：upsert-customer
- 摘要：Upsert customer
- 说明：—
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertCustomerRequest
- 响应码：200、400、401、403、404、410

#### DELETE /openmeter/customers/{customerId}
- version：v3
- operationId：delete-customer
- 摘要：Delete customer
- 说明：—
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### GET /openmeter/customers/{customerId}/billing
- version：v3
- operationId：get-customer-billing
- 摘要：Get customer billing data
- 说明：—
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/customers/{customerId}/billing
- version：v3
- operationId：update-customer-billing
- 摘要：Update customer billing data
- 说明：—
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertCustomerBillingDataRequest
- 响应码：200、400、401、403、404、410

#### PUT /openmeter/customers/{customerId}/billing/app-data
- version：v3
- operationId：update-customer-billing-app-data
- 摘要：Update customer billing app data
- 说明：—
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertAppCustomerDataRequest
- 响应码：200、400、401、403、404、410

#### POST /openmeter/customers/{customerId}/billing/stripe/checkout-sessions
- version：v3
- operationId：create-customer-stripe-checkout-session
- 摘要：Create Stripe Checkout Session
- 说明：Create a [Stripe Checkout Session](https://docs.stripe.com/payments/checkout)
for the customer.

Creates a Checkout Session for collecting payment method information from
customers. The session operates in "setup" mode, which collects payment details
without charging the customer immediately. The collected payment method can be
used for future subscription billing.

For hosted checkout sessions, redirect customers to the returned URL. For
embedded sessions, use the client_secret to initialize Stripe.js in your
application.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingCustomerStripeCreateCheckoutSessionRequest
- 响应码：201、400、401、403、404、410

#### POST /openmeter/customers/{customerId}/billing/stripe/portal-sessions
- version：v3
- operationId：create-customer-stripe-portal-session
- 摘要：Create Stripe customer portal session
- 说明：Create Stripe Customer Portal Session.

Useful to redirect the customer to the Stripe Customer Portal to manage their
payment methods, change their billing address and access their invoice history.
Only returns URL if the customer billing profile is linked to a stripe app and
customer.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingCustomerStripeCreateCustomerPortalSessionRequest
- 响应码：201、400、401、403、404、410

#### GET /openmeter/customers/{customerId}/charges
- version：v3
- operationId：list-customer-charges
- 摘要：List customer charges
- 说明：List customer charges.

Returns the customer's charges that are represented as either flat fee or
usage-based charges.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort charges returned in the response.

Supported sort attributes are:

- `id`
- `created_at`
- `service_period.from`
- `billing_period.from`
- query 参数 **filter**（必填：否）：#/components/schemas/ListChargesParamsFilter，Filter charges.

To filter charges by status add the following query param:
`filter[status][oeq]=created,active`
- query 参数 **expand**（必填：否）：array<#/components/schemas/BillingChargesExpand>，Expand full objects for referenced entities.

Supported values are:

- `real_time_usage`: Expand the charge's real-time usage.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/customers/{customerId}/charges
- version：v3
- operationId：create-customer-charges
- 摘要：Create customer charge
- 说明：Create customer charge.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateChargeRequest
- 响应码：201、400、401、403

#### POST /openmeter/customers/{customerId}/credits/adjustments
- version：v3
- operationId：create-credit-adjustment
- 摘要：Create a credit adjustment
- 说明：A credit adjustment can be used to make manual adjustments to a customer's
credit balance.

Supported use-cases:

- Usage correction
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateCreditAdjustmentRequest
- 响应码：201、400、401、403、404

#### GET /openmeter/customers/{customerId}/credits/balance
- version：v3
- operationId：get-customer-credit-balance
- 摘要：Get a customer's credit balance
- 说明：Get a credit balance.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **timestamp**（必填：否）：#/components/schemas/DateTime，Return the credit balance as of this timestamp.

Defaults to the current time. Historical responses return `live` as zero because
live charge impacts are only available for current balances.
- query 参数 **filter**（必填：否）：#/components/schemas/GetCreditBalanceParamsFilter，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/customers/{customerId}/credits/grants
- version：v3
- operationId：list-credit-grants
- 摘要：List credit grants
- 说明：List credit grants.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **filter**（必填：否）：#/components/schemas/ListCreditGrantsParamsFilter，Filter credit grants returned in the response.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/customers/{customerId}/credits/grants
- version：v3
- operationId：create-credit-grant
- 摘要：Create a new credit grant
- 说明：Create a new credit grant. A credit grant represents an allocation of prepaid
credits to a customer.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateCreditGrantRequest
- 响应码：201、400、401、403、404、409

#### GET /openmeter/customers/{customerId}/credits/grants/{creditGrantId}
- version：v3
- operationId：get-credit-grant
- 摘要：Get a credit grant
- 说明：Get a credit grant.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **creditGrantId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external
- version：v3
- operationId：update-credit-grant-external-settlement
- 摘要：Update credit grant external settlement status
- 说明：Update the payment settlement status of an externally funded credit grant.

Use this endpoint to synchronize the payment state of an external payment with
the system so that revenue recognition and credit availability work as expected.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **creditGrantId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpdateCreditGrantExternalSettlementRequest
- 响应码：200、400、401、403、404

#### POST /openmeter/customers/{customerId}/credits/grants/{creditGrantId}/void
- version：v3
- operationId：void-credit-grant
- 摘要：Void credit grant
- 说明：Void a credit grant, forfeiting the remaining unused balance.

Voiding is a forward-looking, irreversible operation. Credits already consumed
by usage remain unaffected — only the remaining balance is forfeited. The grant
reads as `voided` status afterwards. Payment state is not adjusted when
`payment_adjustment` is `none`, so invoice-backed or externally collected
payments may still collect the original amount. Only `active` grants can be
voided; voiding a pending, expired, or fully consumed grant returns a conflict.
Retrying a successful void is an idempotent success.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **creditGrantId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 否
- application/json：#/components/schemas/VoidCreditGrantRequest
- 响应码：200、400、401、403、404、409

#### GET /openmeter/customers/{customerId}/credits/transactions
- version：v3
- operationId：list-credit-transactions
- 摘要：List credit transactions
- 说明：List credit transactions for a customer.

Returns an immutable, chronological record of credit movements: funded credits
and consumed credits. Transactions are returned in reverse chronological order
by default.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **page**（必填：否）：#/components/schemas/CursorPaginationQueryPage，—
- query 参数 **filter**（必填：否）：#/components/schemas/ListCreditTransactionsParamsFilter，Filter credit transactions returned in the response.
- 请求体：
- 无
- 响应码：200、400、401、403、404


### plan（套餐）

#### GET /openmeter/addons
- version：v3
- operationId：list-addons
- 摘要：List add-ons
- 说明：List all add-ons.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort add-ons returned in the response. Supported sort attributes are:

- `id`
- `key`
- `name`
- `created_at` (default)
- `updated_at`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **filter**（必填：否）：#/components/schemas/ListAddonsParamsFilter，Filter add-ons returned in the response.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/addons
- version：v3
- operationId：create-addon
- 摘要：Create add-on
- 说明：Create a new add-on.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateAddonRequest
- 响应码：201、400、401、403

#### GET /openmeter/addons/{addonId}
- version：v3
- operationId：get-addon
- 摘要：Get add-on
- 说明：Get add-on by id.
- 参数：
- path 参数 **addonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、410

#### PUT /openmeter/addons/{addonId}
- version：v3
- operationId：update-addon
- 摘要：Update add-on
- 说明：Update an add-on by id.
- 参数：
- path 参数 **addonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertAddonRequest
- 响应码：200、400、401、403、404、410

#### DELETE /openmeter/addons/{addonId}
- version：v3
- operationId：delete-addon
- 摘要：Soft delete add-on
- 说明：Soft delete add-on by id.
- 参数：
- path 参数 **addonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### POST /openmeter/addons/{addonId}/archive
- version：v3
- operationId：archive-addon
- 摘要：Archive add-on version
- 说明：Archive an add-on version.
- 参数：
- path 参数 **addonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/addons/{addonId}/publish
- version：v3
- operationId：publish-addon
- 摘要：Publish add-on version
- 说明：Publish an add-on version.
- 参数：
- path 参数 **addonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/plans
- version：v3
- operationId：list-plans
- 摘要：List plans
- 说明：List all plans.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort plans returned in the response. Supported sort attributes are:

- `id`
- `key`
- `version`
- `created_at` (default)
- `updated_at`
- query 参数 **filter**（必填：否）：#/components/schemas/ListPlansParamsFilter，Filter plans returned in the response.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/plans
- version：v3
- operationId：create-plan
- 摘要：Create plan
- 说明：Create a new plan.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreatePlanRequest
- 响应码：201、400、401、403

#### GET /openmeter/plans/{planId}
- version：v3
- operationId：get-plan
- 摘要：Get plan
- 说明：Get a plan by id.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、410

#### PUT /openmeter/plans/{planId}
- version：v3
- operationId：update-plan
- 摘要：Update plan
- 说明：Update a plan by id.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertPlanRequest
- 响应码：200、400、401、403、404、410

#### DELETE /openmeter/plans/{planId}
- version：v3
- operationId：delete-plan
- 摘要：Delete plan
- 说明：Delete a plan by id.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### GET /openmeter/plans/{planId}/addons
- version：v3
- operationId：list-plan-addons
- 摘要：List add-ons for plan
- 说明：List add-ons associated with a plan.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/plans/{planId}/addons
- version：v3
- operationId：create-plan-addon
- 摘要：Add add-on to plan
- 说明：Add an add-on to a plan.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CreatePlanAddonRequest
- 响应码：201、400、401、403、404

#### GET /openmeter/plans/{planId}/addons/{planAddonId}
- version：v3
- operationId：get-plan-addon
- 摘要：Get add-on association for plan
- 说明：Get an add-on association for a plan.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **planAddonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/plans/{planId}/addons/{planAddonId}
- version：v3
- operationId：update-plan-addon
- 摘要：Update add-on association for plan
- 说明：Update an add-on association for a plan.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **planAddonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertPlanAddonRequest
- 响应码：200、400、401、403、404

#### DELETE /openmeter/plans/{planId}/addons/{planAddonId}
- version：v3
- operationId：delete-plan-addon
- 摘要：Remove add-on from plan
- 说明：Remove an add-on from a plan.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **planAddonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### POST /openmeter/plans/{planId}/archive
- version：v3
- operationId：archive-plan
- 摘要：Archive plan version
- 说明：Archive a plan version.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/plans/{planId}/publish
- version：v3
- operationId：publish-plan
- 摘要：Publish plan version
- 说明：Publish a plan version.
- 参数：
- path 参数 **planId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404


### subscription（订阅）

#### GET /openmeter/subscriptions
- version：v3
- operationId：list-subscriptions
- 摘要：List subscriptions
- 说明：—
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort subscriptions returned in the response. Supported sort attributes are:

- `id`
- `active_from` (default)
- `active_to`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **filter**（必填：否）：#/components/schemas/ListSubscriptionsParamsFilter，Filter subscriptions.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/subscriptions
- version：v3
- operationId：create-subscription
- 摘要：Create subscription
- 说明：—
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingSubscriptionCreate
- 响应码：201、400、401、403、404、409

#### GET /openmeter/subscriptions/{subscriptionId}
- version：v3
- operationId：get-subscription
- 摘要：Get subscription
- 说明：—
- 参数：
- path 参数 **subscriptionId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/subscriptions/{subscriptionId}/addons
- version：v3
- operationId：list-subscription-addons
- 摘要：List subscription addons
- 说明：List the add-ons of a subscription.
- 参数：
- path 参数 **subscriptionId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort subscription addons returned in the response. Supported sort attributes
are:

- `id`
- `created_at` (default)
- `updated_at`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/subscriptions/{subscriptionId}/addons
- version：v3
- operationId：create-subscription-addon
- 摘要：Create a new subscription add-on
- 说明：Add add-on to a subscription.
- 参数：
- path 参数 **subscriptionId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateSubscriptionAddonRequest
- 响应码：201、400、401、403、404、409

#### GET /openmeter/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}
- version：v3
- operationId：get-subscription-addon
- 摘要：Get add-on association for subscription
- 说明：Get an add-on association for a subscription.
- 参数：
- path 参数 **subscriptionId**（必填：是）：#/components/schemas/ULID，—
- path 参数 **subscriptionAddonId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/subscriptions/{subscriptionId}/cancel
- version：v3
- operationId：cancel-subscription
- 摘要：Cancel subscription
- 说明：Cancels the subscription. Will result in a scheduling conflict if there are
other subscriptions scheduled to start after the cancelation time.
- 参数：
- path 参数 **subscriptionId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingSubscriptionCancel
- 响应码：200、400、401、403、404、409

#### POST /openmeter/subscriptions/{subscriptionId}/change
- version：v3
- operationId：change-subscription
- 摘要：Change subscription
- 说明：Closes a running subscription and starts a new one according to the
specification. Can be used for upgrades, downgrades, and plan changes.
- 参数：
- path 参数 **subscriptionId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingSubscriptionChange
- 响应码：200、400、401、403、404、409

#### POST /openmeter/subscriptions/{subscriptionId}/unschedule-cancelation
- version：v3
- operationId：unschedule-cancelation
- 摘要：Unschedule subscription cancelation
- 说明：Unschedules the subscription cancelation.
- 参数：
- path 参数 **subscriptionId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、409


### meter（计量）

#### GET /openmeter/meters
- version：v3
- operationId：list-meters
- 摘要：List meters
- 说明：List meters.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort meters returned in the response. Supported sort attributes are:

- `key`
- `name`
- `aggregation`
- `created_at` (default)
- `updated_at`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **filter**（必填：否）：#/components/schemas/ListMetersParamsFilter，Filter meters returned in the response.

To filter meters by key add the following query param: filter[key]=my-meter-key
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/meters
- version：v3
- operationId：create-meter
- 摘要：Create meter
- 说明：Create a meter.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateMeterRequest
- 响应码：201、400、401、403

#### GET /openmeter/meters/{meterId}
- version：v3
- operationId：get-meter
- 摘要：Get meter
- 说明：Get a meter by ID.
- 参数：
- path 参数 **meterId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/meters/{meterId}
- version：v3
- operationId：update-meter
- 摘要：Update meter
- 说明：Update a meter.
- 参数：
- path 参数 **meterId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpdateMeterRequest
- 响应码：200、400、401、403、404

#### DELETE /openmeter/meters/{meterId}
- version：v3
- operationId：delete-meter
- 摘要：Delete meter
- 说明：Delete a meter.
- 参数：
- path 参数 **meterId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### POST /openmeter/meters/{meterId}/query
- version：v3
- operationId：query-meter
- 摘要：Query meter
- 说明：Query a meter for usage.

Set `Accept: application/json` (the default) to get a structured JSON response.
Set `Accept: text/csv` to download the same data as a CSV file suitable for
spreadsheets. The CSV columns, in order, are:

`from, to, [subject,] [customer_id, customer_key, customer_name,] <dimensions...>, value`

The `subject` column is emitted only when `subject` is in the query's
`group_by_dimensions`. The three `customer_*` columns are emitted together only
when `customer_id` is in the query's `group_by_dimensions`.
- 参数：
- path 参数 **meterId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/MeterQueryRequest
- 响应码：200、400、401、403、404


### event（事件）

#### GET /openmeter/events
- version：v3
- operationId：list-metering-events
- 摘要：List metering events
- 说明：List ingested events.
- 参数：
- query 参数 **page**（必填：否）：#/components/schemas/CursorPaginationQueryPage，—
- query 参数 **filter**（必填：否）：#/components/schemas/ListEventsParamsFilter，Filter events returned in the response.

To filter events by subject add the following query param:
filter[subject][eq]=customer-1
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort events returned in the response. Supported sort attributes are:

- `time` (default)
- `ingested_at`
- `stored_at`

When omitted, events are sorted by `time desc` (most recent first). When a sort
field is provided without a suffix, it sorts descending. Append the `asc` suffix
to sort ascending, or the `desc` suffix to sort descending.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/events
- version：v3
- operationId：ingest-metering-events
- 摘要：Ingest metering events
- 说明：Ingests an event or batch of events following the CloudEvents specification.
- 参数：
- 无
- 请求体：
- required: 是
- application/cloudevents+json：#/components/schemas/MeteringEvent
- application/cloudevents-batch+json：array<#/components/schemas/MeteringEvent>
- application/json：—
- 响应码：202、400、401、403


### invoice（账单）

#### GET /checkout-sessions/{sessionId}
- version：v3
- operationId：get-checkout-session
- 摘要：Get checkout session
- 说明：Retrieve a checkout session by its ID (for polling payment status after QR code
expiry or page reload).
- 参数：
- path 参数 **sessionId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/billing/invoices
- version：v3
- operationId：list-invoices
- 摘要：List billing invoices
- 说明：List billing invoices.

Returns a page of invoices. Gathering invoices are never included. Use `filter`
to narrow by status, customer, dates, or service period start. Use `sort` to
control ordering.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort invoices returned in the response. Supported sort attributes:

- `issued_at`
- `created_at` (default)
- `service_period_start`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **filter**（必填：否）：#/components/schemas/ListInvoicesParamsFilter，Filter invoices returned in the response.

Examples:

- `filter[status][oeq]=draft,issued`
- `filter[customer_id]=01KPDB8K...`
- `filter[issued_at][gte]=2024-01-01T00:00:00Z`
- 请求体：
- 无
- 响应码：200、400、401、403

#### GET /openmeter/billing/invoices/{invoiceId}
- version：v3
- operationId：get-invoice
- 摘要：Get a billing invoice
- 说明：Get a billing invoice by ID.

Returns the full invoice resource including line items, status details, totals,
and workflow configuration snapshot.
- 参数：
- path 参数 **invoiceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/billing/invoices/{invoiceId}
- version：v3
- operationId：update-invoice
- 摘要：Update a billing invoice
- 说明：Update a billing invoice.

Only the mutable fields of the invoice can be edited: description, labels,
supplier, customer, workflow settings, and top-level lines. Top-level lines are
matched by `id`; lines without an `id` are created, and existing lines omitted
from `lines` are deleted. Detailed (child) lines are always computed and cannot
be edited directly. Only invoices in draft status can be updated.
- 参数：
- path 参数 **invoiceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpdateInvoiceRequest
- 响应码：200、400、401、403、404

#### DELETE /openmeter/billing/invoices/{invoiceId}
- version：v3
- operationId：delete-invoice
- 摘要：Delete a billing invoice
- 说明：Delete a billing invoice.

Only standard invoices in draft status can be deleted. Deleting an invoice will
also delete all associated line items and workflow configuration.
- 参数：
- path 参数 **invoiceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### POST /openmeter/billing/invoices/{invoiceId}/advance
- version：v3
- operationId：advance-invoice
- 摘要：Advance billing invoice's next status
- 说明：Advance a billing invoice.

Advances the invoice to the next workflow state. The next state is determined by
the invoice's current status and workflow configuration. Only invoices in draft
or issued status can be advanced.
- 参数：
- path 参数 **invoiceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/billing/invoices/{invoiceId}/approve
- version：v3
- operationId：approve-invoice
- 摘要：Send the invoice to the customer
- 说明：Approve a billing invoice.

This call instantly sends the invoice to the customer using the configured
billing profile app.

This call is valid in two invoice statuses:

- draft: the invoice will be sent to the customer, the invoice state becomes
issued
- manual_approval_needed: the invoice will be sent to the customer, the invoice
state becomes issued
- 参数：
- path 参数 **invoiceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/billing/invoices/{invoiceId}/retry
- version：v3
- operationId：retry-invoice
- 摘要：Retry advancing the invoice after a failed attempt
- 说明：Retry sending a billing invoice.

Retry advancing the invoice after a failed attempt.

The action can be called when the invoice's statusDetails' actions field contain
the "retry" action.
- 参数：
- path 参数 **invoiceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/billing/invoices/{invoiceId}/snapshot-quantities
- version：v3
- operationId：snapshot-quantities-invoice
- 摘要：Snapshot quantities for usage based line items
- 说明：Snapshot quantities for usage-based line items.

This call will snapshot the quantities for all usage based line items in the
invoice.

This call is only valid in draft.waiting_for_collection status, where the
collection period can be skipped using this action.
- 参数：
- path 参数 **invoiceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/profiles
- version：v3
- operationId：list-billing-profiles
- 摘要：List billing profiles
- 说明：List billing profiles.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/profiles
- version：v3
- operationId：create-billing-profile
- 摘要：Create a new billing profile
- 说明：Create a new billing profile.

Billing profiles contain the settings for billing and controls invoice
generation. An organization can have multiple billing profiles defined. A
billing profile is linked to a specific app. This association is established
during the billing profile's creation and remains immutable.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateBillingProfileRequest
- 响应码：201、400、401、403

#### GET /openmeter/profiles/{id}
- version：v3
- operationId：get-billing-profile
- 摘要：Get a billing profile
- 说明：Get a billing profile.
- 参数：
- path 参数 **id**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/profiles/{id}
- version：v3
- operationId：update-billing-profile
- 摘要：Update a billing profile
- 说明：Update a billing profile.
- 参数：
- path 参数 **id**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertBillingProfileRequest
- 响应码：200、400、401、403、404

#### DELETE /openmeter/profiles/{id}
- version：v3
- operationId：delete-billing-profile
- 摘要：Delete a billing profile
- 说明：Delete a billing profile.

Only such billing profiles can be deleted that are:

- not the default profile
- not pinned to any customer using customer overrides
- only have finalized invoices
- 参数：
- path 参数 **id**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404


### app（应用）

#### GET /openmeter/app-catalog
- version：v3
- operationId：list-app-catalog
- 摘要：List app catalog
- 说明：List available apps.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/app-catalog/install
- version：v3
- operationId：install-app
- 摘要：Install app from the catalog
- 说明：Install an app from the catalog.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingInstallAppRequest
- 响应码：201、400、401、403

#### GET /openmeter/app-catalog/{appType}
- version：v3
- operationId：get-app-catalog-item
- 摘要：Get app catalog item by type
- 说明：Get an app catalog item by type.
- 参数：
- path 参数 **appType**（必填：是）：#/components/schemas/BillingAppType，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/apps
- version：v3
- operationId：list-apps
- 摘要：List apps
- 说明：List installed apps.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- 请求体：
- 无
- 响应码：200、400、401、403

#### GET /openmeter/apps/{appId}
- version：v3
- operationId：get-app
- 摘要：Get app
- 说明：Get an installed app.
- 参数：
- path 参数 **appId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/apps/{appId}
- version：v3
- operationId：update-app
- 摘要：Update app
- 说明：Update an installed app.
- 参数：
- path 参数 **appId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingUpdateAppRequest
- 响应码：200、400、401、403、404

#### DELETE /openmeter/apps/{appId}
- version：v3
- operationId：uninstall-app
- 摘要：Uninstall app
- 说明：Uninstall an app by ID.
- 参数：
- path 参数 **appId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404


### feature（特性）

#### GET /openmeter/features
- version：v3
- operationId：list-features
- 摘要：List features
- 说明：List all features.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort features returned in the response. Supported sort attributes are:

- `key`
- `name`
- `created_at` (default)
- `updated_at`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **filter**（必填：否）：#/components/schemas/ListFeatureParamsFilter，Filter features returned in the response.

To filter features by meter_id add the following query param:
filter[meter_id][oeq]=<id>
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/features
- version：v3
- operationId：create-feature
- 摘要：Create feature
- 说明：Create a feature.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateFeatureRequest
- 响应码：201、400、401、403

#### GET /openmeter/features/{featureId}
- version：v3
- operationId：get-feature
- 摘要：Get feature
- 说明：Get a feature by id.
- 参数：
- path 参数 **featureId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、410

#### PATCH /openmeter/features/{featureId}
- version：v3
- operationId：update-feature
- 摘要：Update feature
- 说明：Update a feature by id. Currently only the unit_cost field can be updated.
- 参数：
- path 参数 **featureId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpdateFeatureRequest
- 响应码：200、400、401、403、404

#### DELETE /openmeter/features/{featureId}
- version：v3
- operationId：delete-feature
- 摘要：Delete feature
- 说明：Delete a feature by id.
- 参数：
- path 参数 **featureId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### POST /openmeter/features/{featureId}/cost/query
- version：v3
- operationId：query-feature-cost
- 摘要：Query feature cost
- 说明：Query the cost of a feature.
- 参数：
- path 参数 **featureId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 否
- application/json：#/components/schemas/MeterQueryRequest
- 响应码：200、400、401、403、404


### system（系统）

#### POST /ai-usage-batches
- version：v3
- operationId：create-ai-usage-batch
- 摘要：Submit an AI usage batch
- 说明：Submit a Canonical AI Usage Batch for settlement.

The first submit for a given `idempotency_key` returns HTTP 201 with the settled
batch. An identical replay (same `idempotency_key` and `payload_hash`) returns
HTTP 200 with the stored result. A replay with the same `idempotency_key` but a
different `payload_hash` returns HTTP 409.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/AIUsageUsageBatchCreate
- 响应码：200、201、400、401、403、404、409

#### GET /ai-usage-batches/{batchId}
- version：v3
- operationId：get-ai-usage-batch
- 摘要：Get an AI usage batch
- 说明：Retrieve a settled AI Usage Batch by its ID.
- 参数：
- path 参数 **batchId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /customers/{customerId}/credit-balance
- version：v3
- operationId：get-ai-usage-credit-balance
- 摘要：Get AI usage credit balance
- 说明：Get a customer's credit balance for AI usage. Returns the same balance model as
the OpenMeter Credits endpoint but scoped to the AI Usage route.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **timestamp**（必填：否）：#/components/schemas/DateTime，Return the credit balance as of this timestamp. Defaults to now.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /customers/{customerId}/credit-transactions
- version：v3
- operationId：list-ai-usage-credit-transactions
- 摘要：List AI usage credit transactions
- 说明：List credit transactions for a customer's AI usage. Returns the same transaction
model as the OpenMeter Credits endpoint but scoped to the AI Usage route.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **page**（必填：否）：#/components/schemas/CursorPaginationQueryPage，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /customers/{customerId}/runtime-authorization
- version：v3
- operationId：get-customer-runtime-authorization
- 摘要：Get runtime authorization
- 说明：Check whether a customer is authorized to consume AI resources.

Returns the current integer credit balance, reservation ceiling, and the covered
tenant sequence watermark.
- 参数：
- path 参数 **customerId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **filter**（必填：否）：#/components/schemas/AIUsageRuntimeAuthorizationQuery，Filter the authorization check by subject and reservation.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/currencies
- version：v3
- operationId：list-currencies
- 摘要：List currencies
- 说明：List currencies supported by the billing system.
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort currencies returned in the response. Supported sort attributes are:

- `code` (default)
- `name`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **filter**（必填：否）：#/components/schemas/ListCurrenciesParamsFilter，Filter currencies returned in the response.

To filter currencies by type add the following query param: filter[type]=custom
- query 参数 **expand**（必填：否）：array<#/components/schemas/BillingCurrencyExpand>，Expand the currencies returned in the response.

To include the currently-active cost basis add: expand=cost_basis
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/currencies/custom
- version：v3
- operationId：create-custom-currency
- 摘要：Create custom currency
- 说明：Create a custom currency. This operation allows defining your own custom
currency for billing purposes.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateCurrencyCustomRequest
- 响应码：201、400、401、403

#### GET /openmeter/currencies/custom/{currencyId}
- version：v3
- operationId：get-custom-currency
- 摘要：Get custom currency
- 说明：Get a custom currency.
- 参数：
- path 参数 **currencyId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403

#### GET /openmeter/currencies/custom/{currencyId}/cost-bases
- version：v3
- operationId：list-cost-bases
- 摘要：List cost bases
- 说明：List cost bases for a currency. For custom currencies, there can be multiple
cost bases with different `effective_from` dates.
- 参数：
- path 参数 **currencyId**（必填：是）：#/components/schemas/ULID，—
- query 参数 **filter**（必填：否）：#/components/schemas/ListCostBasesParamsFilter，Filter cost bases returned in the response.

To filter cost bases by fiat currency code add the following query param:
filter[fiat_code]=USD
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /openmeter/currencies/custom/{currencyId}/cost-bases
- version：v3
- operationId：create-cost-basis
- 摘要：Create cost basis
- 说明：Create a cost basis for a currency.
- 参数：
- path 参数 **currencyId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateCostBasisRequest
- 响应码：201、400、401、403

#### GET /openmeter/defaults/tax-codes
- version：v3
- operationId：get-organization-default-tax-codes
- 摘要：Get organization default tax codes
- 说明：—
- 参数：
- 无
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/defaults/tax-codes
- version：v3
- operationId：update-organization-default-tax-codes
- 摘要：Update organization default tax codes
- 说明：—
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/UpdateOrganizationDefaultTaxCodesRequest
- 响应码：200、400、401、403、404

#### POST /openmeter/governance/query
- version：v3
- operationId：query-governance-access
- 摘要：Query governance access
- 说明：Query feature access for a list of customers.

The endpoint resolves each provided identifier to a customer and returns the
access status for the requested features, plus optional credit balance
availability.

_Designed to be called on a fixed refresh interval and the query response is
intended to be cached._
- 参数：
- query 参数 **page**（必填：否）：#/components/schemas/CursorPaginationQueryPage，—
- 请求体：
- required: 是
- application/json：#/components/schemas/GovernanceQueryRequest
- 响应码：200、400、401、403

#### GET /openmeter/llm-cost/overrides
- version：v3
- operationId：list-llm-cost-overrides
- 摘要：List LLM cost overrides
- 说明：List per-namespace price overrides.
- 参数：
- query 参数 **filter**（必填：否）：#/components/schemas/ListLLMCostPricesParamsFilter，—
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/llm-cost/overrides
- version：v3
- operationId：create-llm-cost-override
- 摘要：Create LLM cost override
- 说明：Create a per-namespace price override.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/LLMCostOverrideCreate
- 响应码：201、400、401、403

#### DELETE /openmeter/llm-cost/overrides/{priceId}
- version：v3
- operationId：delete-llm-cost-override
- 摘要：Delete LLM cost override
- 说明：Delete a per-namespace price override.
- 参数：
- path 参数 **priceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### GET /openmeter/llm-cost/prices
- version：v3
- operationId：list-llm-cost-prices
- 摘要：List LLM cost prices
- 说明：List global LLM cost prices. Returns prices with overrides applied if any.
- 参数：
- query 参数 **filter**（必填：否）：#/components/schemas/ListLLMCostPricesParamsFilter，Filter prices.
- query 参数 **sort**（必填：否）：#/components/schemas/SortQuery，Sort prices returned in the response. Supported sort attributes are:

- `id`
- `provider.id`
- `model.id` (default)
- `effective_from`
- `effective_to`

The `asc` suffix is optional as the default sort order is ascending. The `desc`
suffix is used to specify a descending order.
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- 请求体：
- 无
- 响应码：200、400、401、403

#### GET /openmeter/llm-cost/prices/{priceId}
- version：v3
- operationId：get-llm-cost-price
- 摘要：Get LLM cost price
- 说明：Get a specific LLM cost price by ID. Returns the price with overrides applied if
any.
- 参数：
- path 参数 **priceId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### GET /openmeter/tax-codes
- version：v3
- operationId：list-tax-codes
- 摘要：List tax codes
- 说明：—
- 参数：
- query 参数 **page**（必填：否）：object，Determines which page of the collection to retrieve.
- query 参数 **include_deleted**（必填：否）：boolean，Include deleted tax codes in the response.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /openmeter/tax-codes
- version：v3
- operationId：create-tax-code
- 摘要：Create tax code
- 说明：—
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateTaxCodeRequest
- 响应码：201、400、401、403

#### GET /openmeter/tax-codes/{taxCodeId}
- version：v3
- operationId：get-tax-code
- 摘要：Get tax code
- 说明：—
- 参数：
- path 参数 **taxCodeId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### PUT /openmeter/tax-codes/{taxCodeId}
- version：v3
- operationId：upsert-tax-code
- 摘要：Upsert tax code
- 说明：—
- 参数：
- path 参数 **taxCodeId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/UpsertTaxCodeRequest
- 响应码：200、400、401、403、404、410

#### DELETE /openmeter/tax-codes/{taxCodeId}
- version：v3
- operationId：delete-tax-code
- 摘要：Delete tax code
- 说明：—
- 参数：
- path 参数 **taxCodeId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：204、400、401、403、404

#### POST /orders
- version：v3
- operationId：create-order
- 摘要：Create order
- 说明：Create a new order (plan purchase, subscription renewal, or wallet top-up).

Returns HTTP 201 on first creation. Replaying the same idempotency key returns
the stored order with HTTP 200.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CommerceOrderCreate
- 响应码：200、201、400、401、403、409

#### GET /orders/{orderId}
- version：v3
- operationId：get-order
- 摘要：Get order
- 说明：Retrieve an order by its ID.
- 参数：
- path 参数 **orderId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404

#### POST /orders/{orderId}/checkout-sessions
- version：v3
- operationId：create-checkout-session
- 摘要：Create checkout session
- 说明：Create a checkout session for an order, initiating a payment attempt with the
specified provider.
- 参数：
- path 参数 **orderId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CommerceCheckoutSessionCreate
- 响应码：201、400、401、403、404

#### POST /payment-providers/alipay/callback
- version：v3
- operationId：alipay-payment-callback
- 摘要：Alipay callback
- 说明：Alipay payment callback. OpenMeter verifies the signature, confirms the payment
fact, and fulfills the order.
- 参数：
- 无
- 请求体：
- required: 是
- text/plain：string
- 响应码：200、400、401、403

#### POST /payment-providers/wechat/callback
- version：v3
- operationId：wechat-payment-callback
- 摘要：WeChat Pay callback
- 说明：WeChat Pay payment callback. OpenMeter verifies the signature, confirms the
payment fact, and fulfills the order.
- 参数：
- 无
- 请求体：
- required: 是
- text/plain：string
- 响应码：200、400、401、403

#### GET /recharge-products
- version：v3
- operationId：list-recharge-products
- 摘要：List recharge products
- 说明：List all active recharge products available for purchase.
- 参数：
- query 参数 **currency**（必填：否）：#/components/schemas/CurrencyCode，Filter by currency to show only products priced in the customer's currency.
- 请求体：
- 无
- 响应码：200、400、401、403

#### POST /refunds
- version：v3
- operationId：create-refund
- 摘要：Create refund
- 说明：Create a refund request for an order.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CommerceRefundCreate
- 响应码：201、400、401、403、409

#### GET /refunds/{refundId}
- version：v3
- operationId：get-refund
- 摘要：Get refund
- 说明：Retrieve a refund by its ID.
- 参数：
- path 参数 **refundId**（必填：是）：#/components/schemas/ULID，—
- 请求体：
- 无
- 响应码：200、400、401、403、404


## v1

### customer（客户）

#### GET /api/v1/customers
- version：v1
- operationId：listCustomers
- 摘要：List customers
- 说明：List customers.
- 参数：
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/CustomerOrderBy，The order by field.
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted customers.
- query 参数 **key**（必填：否）：string，Filter customers by key.
Case-insensitive partial match.
- query 参数 **name**（必填：否）：string，Filter customers by name.
Case-insensitive partial match.
- query 参数 **primaryEmail**（必填：否）：string，Filter customers by primary email.
Case-insensitive partial match.
- query 参数 **subject**（必填：否）：string，Filter customers by usage attribution subject.
Case-insensitive partial match.
- query 参数 **planKey**（必填：否）：string，Filter customers by the plan key of their susbcription.
- query 参数 **expand**（必填：否）：array<#/components/schemas/CustomerExpand>，What parts of the list output to expand in listings
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/customers
- version：v1
- operationId：createCustomer
- 摘要：Create customer
- 说明：Create a new customer.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CustomerCreate
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/customers/{customerIdOrKey}
- version：v1
- operationId：getCustomer
- 摘要：Get customer
- 说明：Get a customer by ID or key.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- query 参数 **expand**（必填：否）：array<#/components/schemas/CustomerExpand>，What parts of the customer output to expand
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/customers/{customerIdOrKey}
- version：v1
- operationId：updateCustomer
- 摘要：Update customer
- 说明：Update a customer by ID.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CustomerReplaceUpdate
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/customers/{customerIdOrKey}
- version：v1
- operationId：deleteCustomer
- 摘要：Delete customer
- 说明：Delete a customer by ID.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### GET /api/v1/customers/{customerIdOrKey}/apps
- version：v1
- operationId：listCustomerAppData
- 摘要：List customer app data
- 说明：List customers app data.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **type**（必填：否）：#/components/schemas/AppType，Filter customer data by app type.
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/customers/{customerIdOrKey}/apps
- version：v1
- operationId：upsertCustomerAppData
- 摘要：Upsert customer app data
- 说明：Upsert customer app data.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- required: 是
- application/json：array<#/components/schemas/CustomerAppDataCreateOrUpdateItem>
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/customers/{customerIdOrKey}/apps/{appId}
- version：v1
- operationId：deleteCustomerAppData
- 摘要：Delete customer app data
- 说明：Delete customer app data.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **appId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### GET /api/v1/customers/{customerIdOrKey}/stripe
- version：v1
- operationId：getCustomerStripeAppData
- 摘要：Get customer stripe app data
- 说明：Get stripe app data for a customer.
Only returns data if the customer billing profile is linked to a stripe app.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/customers/{customerIdOrKey}/stripe
- version：v1
- operationId：upsertCustomerStripeAppData
- 摘要：Upsert customer stripe app data
- 说明：Upsert stripe app data for a customer.
Only updates data if the customer billing profile is linked to a stripe app.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- required: 是
- application/json：#/components/schemas/StripeCustomerAppDataBase
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/customers/{customerIdOrKey}/stripe/portal
- version：v1
- operationId：createCustomerStripePortalSession
- 摘要：Create Stripe customer portal session
- 说明：Create Stripe customer portal session.
Only returns URL if the customer billing profile is linked to a stripe app and customer.

Useful to redirect the customer to the Stripe customer portal to manage their payment methods,
change their billing address and access their invoice history.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateStripeCustomerPortalSessionParams
- 响应码：201、400、401、403、404、412、500、503、default

#### GET /api/v1/customers/{customerIdOrKey}/subscriptions
- version：v1
- operationId：listCustomerSubscriptions
- 摘要：List customer subscriptions
- 说明：Lists all subscriptions for a customer.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- query 参数 **status**（必填：否）：array<#/components/schemas/SubscriptionStatus>，—
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/CustomerSubscriptionOrderBy，The order by field.
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default


### plan（套餐）

#### GET /api/v1/addons
- version：v1
- operationId：listAddons
- 摘要：List add-ons
- 说明：List all add-ons.
- 参数：
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted add-ons in response.

Usage: `?includeDeleted=true`
- query 参数 **id**（必填：否）：array<string>，Filter by addon.id attribute
- query 参数 **key**（必填：否）：array<string>，Filter by addon.key attribute
- query 参数 **keyVersion**（必填：否）：object，Filter by addon.key and addon.version attributes
- query 参数 **status**（必填：否）：array<#/components/schemas/AddonStatus>，Only return add-ons with the given status.

Usage:
- `?status=active`: return only the currently active add-ons
- `?status=draft`: return only the draft add-ons
- `?status=archived`: return only the archived add-ons
- query 参数 **currency**（必填：否）：array<#/components/schemas/CurrencyCode>，Filter by addon.currency attribute
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/AddonOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/addons
- version：v1
- operationId：createAddon
- 摘要：Create an add-on
- 说明：Create a new add-on.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/AddonCreate
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/addons/{addonId}
- version：v1
- operationId：getAddon
- 摘要：Get add-on
- 说明：Get add-on by id or key. The latest published version is returned if latter is used.
- 参数：
- path 参数 **addonId**（必填：是）：string，—
- query 参数 **includeLatest**（必填：否）：boolean，Include latest version of the add-on instead of the version in active state.

Usage: `?includeLatest=true`
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/addons/{addonId}
- version：v1
- operationId：updateAddon
- 摘要：Update add-on
- 说明：Update add-on by id.
- 参数：
- path 参数 **addonId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/AddonReplaceUpdate
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/addons/{addonId}
- version：v1
- operationId：deleteAddon
- 摘要：Delete add-on
- 说明：Soft delete add-on by id.

Once a add-on is deleted it cannot be undeleted.
- 参数：
- path 参数 **addonId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### POST /api/v1/addons/{addonId}/archive
- version：v1
- operationId：archiveAddon
- 摘要：Archive add-on version
- 说明：Archive a add-on version.
- 参数：
- path 参数 **addonId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/addons/{addonId}/publish
- version：v1
- operationId：publishAddon
- 摘要：Publish add-on
- 说明：Publish a add-on version.
- 参数：
- path 参数 **addonId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### GET /api/v1/features
- version：v1
- operationId：listFeatures
- 摘要：List features
- 说明：List features.
- 参数：
- query 参数 **meterSlug**（必填：否）：array<string>，Filter by meterSlug
- query 参数 **includeArchived**（必填：否）：boolean，Include archived features in response.
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **offset**（必填：否）：integer，Number of items to skip.

Default is 0.
- query 参数 **limit**（必填：否）：integer，Number of items to return.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/FeatureOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/features
- version：v1
- operationId：createFeature
- 摘要：Create feature
- 说明：Features are either metered or static. A feature is metered if meterSlug is provided at creation.
For metered features you can pass additional filters that will be applied when calculating feature usage, based on the meter's groupBy fields.
Meters with SUM, COUNT, UNIQUE_COUNT and LATEST aggregations are supported for features.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/FeatureCreateInputs
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/features/{featureId}
- version：v1
- operationId：getFeature
- 摘要：Get feature
- 说明：Get a feature by ID.
- 参数：
- path 参数 **featureId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/features/{featureId}
- version：v1
- operationId：deleteFeature
- 摘要：Delete feature
- 说明：Archive a feature by ID.

Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements cannot be created for it, but archiving the feature does not affect existing entitlements.
This means, if you want to create a new feature with the same key, and then create entitlements for it, the previous entitlements have to be deleted first on a per subject basis.
- 参数：
- path 参数 **featureId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### GET /api/v1/plans
- version：v1
- operationId：listPlans
- 摘要：List plans
- 说明：List all plans.
- 参数：
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted plans in response.

Usage: `?includeDeleted=true`
- query 参数 **id**（必填：否）：array<string>，Filter by plan.id attribute
- query 参数 **key**（必填：否）：array<string>，Filter by plan.key attribute
- query 参数 **keyVersion**（必填：否）：object，Filter by plan.key and plan.version attributes
- query 参数 **status**（必填：否）：array<#/components/schemas/PlanStatus>，Only return plans with the given status.

Usage:
- `?status=active`: return only the currently active plan
- `?status=draft`: return only the draft plan
- `?status=archived`: return only the archived plans
- query 参数 **currency**（必填：否）：array<#/components/schemas/CurrencyCode>，Filter by plan.currency attribute
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/PlanOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/plans
- version：v1
- operationId：createPlan
- 摘要：Create a plan
- 说明：Create a new plan.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/PlanCreate
- 响应码：201、400、401、403、412、500、503、default

#### POST /api/v1/plans/{planIdOrKey}/next
- version：v1
- operationId：nextPlan
- 摘要：New draft plan
- 说明：Create a new draft version from plan.
It returns error if there is already a plan in draft or planId does not reference the latest published version.
- 参数：
- path 参数 **planIdOrKey**（必填：是）：string，—
- 请求体：
- 无
- 响应码：201、400、401、403、404、412、500、503、default

#### GET /api/v1/plans/{planId}
- version：v1
- operationId：getPlan
- 摘要：Get plan
- 说明：Get a plan by id or key. The latest published version is returned if latter is used.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- query 参数 **includeLatest**（必填：否）：boolean，Include latest version of the Plan instead of the version in active state.

Usage: `?includeLatest=true`
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/plans/{planId}
- version：v1
- operationId：updatePlan
- 摘要：Update a plan
- 说明：Update plan by id.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/PlanReplaceUpdate
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/plans/{planId}
- version：v1
- operationId：deletePlan
- 摘要：Delete plan
- 说明：Soft delete plan by plan.id.

Once a plan is deleted it cannot be undeleted.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### GET /api/v1/plans/{planId}/addons
- version：v1
- operationId：listPlanAddons
- 摘要：List all available add-ons for plan
- 说明：List all available add-ons for plan.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted plan add-on assignments.

Usage: `?includeDeleted=true`
- query 参数 **id**（必填：否）：array<string>，Filter by addon.id attribute.
- query 参数 **key**（必填：否）：array<string>，Filter by addon.key attribute.
- query 参数 **keyVersion**（必填：否）：object，Filter by addon.key and addon.version attributes.
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/PlanAddonOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/plans/{planId}/addons
- version：v1
- operationId：createPlanAddon
- 摘要：Create new add-on assignment for plan
- 说明：Create new add-on assignment for plan.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/PlanAddonCreate
- 响应码：201、400、401、403、404、409、412、500、503、default

#### GET /api/v1/plans/{planId}/addons/{planAddonId}
- version：v1
- operationId：getPlanAddon
- 摘要：Get add-on assignment for plan
- 说明：Get add-on assignment for plan by id.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- path 参数 **planAddonId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/plans/{planId}/addons/{planAddonId}
- version：v1
- operationId：updatePlanAddon
- 摘要：Update add-on assignment for plan
- 说明：Update add-on assignment for plan.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- path 参数 **planAddonId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/PlanAddonReplaceUpdate
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/plans/{planId}/addons/{planAddonId}
- version：v1
- operationId：deletePlanAddon
- 摘要：Delete add-on assignment for plan
- 说明：Delete add-on assignment for plan.

Once a plan is deleted it cannot be undeleted.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- path 参数 **planAddonId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### POST /api/v1/plans/{planId}/archive
- version：v1
- operationId：archivePlan
- 摘要：Archive plan version
- 说明：Archive a plan version.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/plans/{planId}/publish
- version：v1
- operationId：publishPlan
- 摘要：Publish plan
- 说明：Publish a plan version.
- 参数：
- path 参数 **planId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default


### subscription（订阅）

#### POST /api/v1/subscriptions
- version：v1
- operationId：createSubscription
- 摘要：Create subscription
- 说明：—
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/SubscriptionCreate
- 响应码：201、400、401、403、409、412、500、503、default

#### GET /api/v1/subscriptions/{subscriptionId}
- version：v1
- operationId：getSubscription
- 摘要：Get subscription
- 说明：—
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- query 参数 **at**（必填：否）：string，The time at which the subscription should be queried. If not provided the current time is used.
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PATCH /api/v1/subscriptions/{subscriptionId}
- version：v1
- operationId：editSubscription
- 摘要：Edit subscription
- 说明：Batch processing commands for manipulating running subscriptions.
The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/SubscriptionEdit
- 响应码：200、400、401、403、404、409、412、500、503、default

#### DELETE /api/v1/subscriptions/{subscriptionId}
- version：v1
- operationId：deleteSubscription
- 摘要：Delete subscription
- 说明：Deletes a subscription. Only scheduled subscriptions can be deleted.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、409、412、500、503、default

#### GET /api/v1/subscriptions/{subscriptionId}/addons
- version：v1
- operationId：listSubscriptionAddons
- 摘要：List subscription addons
- 说明：List all addons of a subscription. In the returned list will match to a set unique by addonId.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/subscriptions/{subscriptionId}/addons
- version：v1
- operationId：createSubscriptionAddon
- 摘要：Create subscription addon
- 说明：Create a new subscription addon, either providing the key or the id of the addon.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/SubscriptionAddonCreate
- 响应码：201、400、401、403、404、409、412、500、503、default

#### GET /api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}
- version：v1
- operationId：getSubscriptionAddon
- 摘要：Get subscription addon
- 说明：Get a subscription addon by id.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- path 参数 **subscriptionAddonId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PATCH /api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}
- version：v1
- operationId：updateSubscriptionAddon
- 摘要：Update subscription addon
- 说明：Updates a subscription addon (allows changing the quantity: purchasing more instances or cancelling the current instances)
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- path 参数 **subscriptionAddonId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/SubscriptionAddonUpdate
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/subscriptions/{subscriptionId}/cancel
- version：v1
- operationId：cancelSubscription
- 摘要：Cancel subscription
- 说明：Cancels the subscription.
Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancellation time.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：object
- 响应码：200、400、401、403、404、409、412、500、503、default

#### POST /api/v1/subscriptions/{subscriptionId}/change
- version：v1
- operationId：changeSubscription
- 摘要：Change subscription
- 说明：Closes a running subscription and starts a new one according to the specification.
Can be used for upgrades, downgrades, and plan changes.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/SubscriptionChange
- 响应码：200、400、401、403、404、409、412、500、503、default

#### POST /api/v1/subscriptions/{subscriptionId}/migrate
- version：v1
- operationId：migrateSubscription
- 摘要：Migrate subscription
- 说明：Migrates the subscripiton to the provided version of the current plan.
If possible, the migration will be done immediately.
If not, the migration will be scheduled to the end of the current billing period.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：object
- 响应码：200、400、401、403、404、409、412、500、503、default

#### POST /api/v1/subscriptions/{subscriptionId}/restore
- version：v1
- operationId：restoreSubscription
- 摘要：Restore subscription
- 说明：Restores a canceled subscription.
Any subscription scheduled to start later will be deleted and this subscription will be continued indefinitely.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/subscriptions/{subscriptionId}/unschedule-cancelation
- version：v1
- operationId：unscheduleCancelation
- 摘要：Unschedule cancelation
- 说明：Cancels the scheduled cancelation.
- 参数：
- path 参数 **subscriptionId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、409、412、500、503、default


### entitlement（权限/权益）

#### GET /api/v1/customers/{customerIdOrKey}/access
- version：v1
- operationId：getCustomerAccess
- 摘要：Get customer access
- 说明：Get the overall access of a customer.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### GET /api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/value
- version：v1
- operationId：getCustomerEntitlementValue
- 摘要：Get customer entitlement value
- 说明：Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **featureKey**（必填：是）：string，—
- query 参数 **time**（必填：否）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/grants/{grantId}
- version：v1
- operationId：voidGrant
- 摘要：Void grant
- 说明：Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations. Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed to the grant will remain, but future usage cannot be burnt down from the grant.
For example, if you have a single grant for your metered entitlement with an initial amount of 100, and so far 60 usage has been metered, the grant (and the entitlement itself) would have a balance of 40. If you then void that grant, balance becomes 0, but the 60 previous usage will not be affected.
- 参数：
- path 参数 **grantId**（必填：是）：string，—
- query 参数 **at**（必填：否）：string，The time at which the grant should be voided.
Must not be in the future and must be within the current usage period of the entitlement.
Defaults to the current time if not specified.
- 请求体：
- 无
- 响应码：204、400、401、403、404、409、412、500、503、default


### meter（计量）

#### GET /api/v1/meters
- version：v1
- operationId：listMeters
- 摘要：List meters
- 说明：List meters.
- 参数：
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/MeterOrderBy，The order by field.
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted meters.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/meters
- version：v1
- operationId：createMeter
- 摘要：Create meter
- 说明：Create a meter.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/MeterCreate
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/meters/{meterIdOrSlug}
- version：v1
- operationId：getMeter
- 摘要：Get meter
- 说明：Get a meter by ID or slug.
- 参数：
- path 参数 **meterIdOrSlug**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/meters/{meterIdOrSlug}
- version：v1
- operationId：updateMeter
- 摘要：Update meter
- 说明：Update a meter.
- 参数：
- path 参数 **meterIdOrSlug**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/MeterUpdate
- 响应码：200、400、401、403、412、500、503、default

#### DELETE /api/v1/meters/{meterIdOrSlug}
- version：v1
- operationId：deleteMeter
- 摘要：Delete meter
- 说明：Delete a meter.
- 参数：
- path 参数 **meterIdOrSlug**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、412、500、503、default

#### GET /api/v1/meters/{meterIdOrSlug}/group-by/{groupByKey}/values
- version：v1
- operationId：listMeterGroupByValues
- 摘要：List meter group by values
- 说明：List meter group by values.
- 参数：
- path 参数 **meterIdOrSlug**（必填：是）：string，—
- path 参数 **groupByKey**（必填：是）：string，—
- query 参数 **from**（必填：否）：string，Start date-time in RFC 3339 format.

Inclusive. Defaults to 24 hours ago.

For example: ?from=2025-01-01T00%3A00%3A00.000Z
- query 参数 **to**（必填：否）：string，End date-time in RFC 3339 format.

Inclusive.

For example: ?to=2025-02-01T00%3A00%3A00.000Z
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/meters/{meterIdOrSlug}/query
- version：v1
- operationId：queryMeterPost
- 摘要：Query meter
- 说明：—
- 参数：
- path 参数 **meterIdOrSlug**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/MeterQueryRequest
- 响应码：200、400、401、403、404、412、500、503、default

#### GET /api/v1/meters/{meterIdOrSlug}/subjects
- version：v1
- operationId：listMeterSubjects
- 摘要：List meter subjects
- 说明：List subjects for a meter.
- 参数：
- path 参数 **meterIdOrSlug**（必填：是）：string，—
- query 参数 **from**（必填：否）：string，Start date-time in RFC 3339 format.

Inclusive. Defaults to the beginning of time.

For example: ?from=2025-01-01T00%3A00%3A00.000Z
- query 参数 **to**（必填：否）：string，End date-time in RFC 3339 format.

Inclusive.

For example: ?to=2025-02-01T00%3A00%3A00.000Z
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default


### invoice（账单）

#### GET /api/v1/billing/customers
- version：v1
- operationId：listBillingProfileCustomerOverrides
- 摘要：List customer overrides
- 说明：List customer overrides using the specified filters.

The response will include the customer override values and the merged billing profile values.

If the includeAllCustomers is set to true, the list contains all customers. This mode is
useful for getting the current effective billing workflow settings for all users regardless
if they have customer orverrides or not.
- 参数：
- query 参数 **billingProfile**（必填：否）：array<string>，Filter by billing profile.
- query 参数 **customersWithoutPinnedProfile**（必填：否）：boolean，Only return customers without pinned billing profiles. This implicitly sets includeAllCustomers to true.
- query 参数 **includeAllCustomers**（必填：否）：boolean，Include customers without customer overrides.

If set to false only the customers specifically associated with a billing profile will be returned.

If set to true, in case of the default billing profile, all customers will be returned.
- query 参数 **customerId**（必填：否）：array<string>，Filter by customer id.
- query 参数 **customerName**（必填：否）：string，Filter by customer name.
- query 参数 **customerKey**（必填：否）：string，Filter by customer key
- query 参数 **customerPrimaryEmail**（必填：否）：string，Filter by customer primary email
- query 参数 **expand**（必填：否）：array<#/components/schemas/BillingProfileCustomerOverrideExpand>，Expand the response with additional details.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/BillingProfileCustomerOverrideOrderBy，The order by field.
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/billing/customers/{customerId}
- version：v1
- operationId：getBillingProfileCustomerOverride
- 摘要：Get a customer override
- 说明：Get a customer override by customer id.

The response will include the customer override values and the merged billing profile values.

If the customer override is not found, the default billing profile's values are returned. This behavior
allows for getting a merged profile regardless of the customer override existence.
- 参数：
- path 参数 **customerId**（必填：是）：string，—
- query 参数 **expand**（必填：否）：array<#/components/schemas/BillingProfileCustomerOverrideExpand>，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/billing/customers/{customerId}
- version：v1
- operationId：upsertBillingProfileCustomerOverride
- 摘要：Create a new or update a customer override
- 说明：The customer override can be used to pin a given customer to a billing profile
different from the default one.

This can be used to test the effect of different billing profiles before making them
the default ones or have different workflow settings for example for enterprise customers.
- 参数：
- path 参数 **customerId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingProfileCustomerOverrideCreate
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/billing/customers/{customerId}
- version：v1
- operationId：deleteBillingProfileCustomerOverride
- 摘要：Delete a customer override
- 说明：Delete a customer override by customer id.

This will remove the customer override and the customer will be subject to the default
billing profile's settings again.
- 参数：
- path 参数 **customerId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### POST /api/v1/billing/customers/{customerId}/invoices/pending-lines
- version：v1
- operationId：createPendingInvoiceLine
- 摘要：Create pending line items
- 说明：Create a new pending line item (charge).

This call is used to create a new pending line item for the customer if required a new
gathering invoice will be created.

A new invoice will be created if:
- there is no invoice in gathering state
- the currency of the line item doesn't match the currency of any invoices in gathering state
- 参数：
- path 参数 **customerId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/InvoicePendingLineCreateInput
- 响应码：201、400、401、403、412、500、503、default

#### POST /api/v1/billing/customers/{customerId}/invoices/simulate
- version：v1
- operationId：simulateInvoice
- 摘要：Simulate an invoice for a customer
- 说明：Simulate an invoice for a customer.

This call will simulate an invoice for a customer based on the pending line items.

The call will return the total amount of the invoice and the line items that will be included in the invoice.
- 参数：
- path 参数 **customerId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/InvoiceSimulationInput
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/billing/invoices
- version：v1
- operationId：listInvoices
- 摘要：List invoices
- 说明：List invoices based on the specified filters.

The expand option can be used to include additional information (besides the invoice header and totals)
in the response. For example by adding the expand=lines option the invoice lines will be included in the response.

Gathering invoices will always show the current usage calculated on the fly.
- 参数：
- query 参数 **statuses**（必填：否）：array<#/components/schemas/InvoiceStatus>，Filter by the invoice status.
- query 参数 **extendedStatuses**（必填：否）：array<string>，Filter by invoice extended statuses
- query 参数 **issuedAfter**（必填：否）：string，Filter by invoice issued time.
Inclusive.
- query 参数 **issuedBefore**（必填：否）：string，Filter by invoice issued time.
Inclusive.
- query 参数 **periodStartAfter**（必填：否）：string，Filter by period start time.
Inclusive.
- query 参数 **periodStartBefore**（必填：否）：string，Filter by period start time.
Inclusive.
- query 参数 **createdAfter**（必填：否）：string，Filter by invoice created time.
Inclusive.
- query 参数 **createdBefore**（必填：否）：string，Filter by invoice created time.
Inclusive.
- query 参数 **expand**（必填：否）：array<#/components/schemas/InvoiceExpand>，What parts of the list output to expand in listings
- query 参数 **customers**（必填：否）：array<string>，Filter by customer ID
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted invoices
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/InvoiceOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/billing/invoices/invoice
- version：v1
- operationId：invoicePendingLinesAction
- 摘要：Invoice a customer based on the pending line items
- 说明：Create a new invoice from the pending line items.

This should be only called if for some reason we need to invoice a customer outside of the normal billing cycle.

When creating an invoice, the pending line items will be marked as invoiced and the invoice will be created with the total amount of the pending items.

New pending line items will be created for the period between now() and the next billing cycle's begining date for any metered item.

The call can return multiple invoices if the pending line items are in different currencies.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/InvoicePendingLinesActionInput
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/billing/invoices/{invoiceId}
- version：v1
- operationId：getInvoice
- 摘要：Get an invoice
- 说明：Get an invoice by ID.

Gathering invoices will always show the current usage calculated on the fly.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- query 参数 **expand**（必填：否）：array<#/components/schemas/InvoiceExpand>，—
- query 参数 **includeDeletedLines**（必填：否）：boolean，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/billing/invoices/{invoiceId}
- version：v1
- operationId：updateInvoice
- 摘要：Update an invoice
- 说明：Update an invoice

Only invoices in draft or earlier status can be updated.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/InvoiceReplaceUpdate
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/billing/invoices/{invoiceId}
- version：v1
- operationId：deleteInvoice
- 摘要：Delete an invoice
- 说明：Delete an invoice

Only invoices that are in the draft (or earlier) status can be deleted.

Invoices that are post finalization can only be voided.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### POST /api/v1/billing/invoices/{invoiceId}/advance
- version：v1
- operationId：advanceInvoiceAction
- 摘要：Advance the invoice's state to the next status
- 说明：Advance the invoice's state to the next status.

The call doesn't "approve the invoice", it only advances the invoice to the next status if the transition would be automatic.

The action can be called when the invoice's statusDetails' actions field contain the "advance" action.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/billing/invoices/{invoiceId}/approve
- version：v1
- operationId：approveInvoiceAction
- 摘要：Send the invoice to the customer
- 说明：Approve an invoice and start executing the payment workflow.

This call instantly sends the invoice to the customer using the configured billing profile app.

This call is valid in two invoice statuses:
- `draft`: the invoice will be sent to the customer, the invluce state becomes issued
- `manual_approval_needed`: the invoice will be sent to the customer, the invoice state becomes issued
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/billing/invoices/{invoiceId}/retry
- version：v1
- operationId：retryInvoiceAction
- 摘要：Retry advancing the invoice after a failed attempt.
- 说明：Retry advancing the invoice after a failed attempt.

The action can be called when the invoice's statusDetails' actions field contain the "retry" action.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/billing/invoices/{invoiceId}/snapshot-quantities
- version：v1
- operationId：snapshotQuantitiesInvoiceAction
- 摘要：Snapshot quantities for usage based line items
- 说明：Snapshot quantities for usage based line items.

This call will snapshot the quantities for all usage based line items in the invoice.

This call is only valid in `draft.waiting_for_collection` status, where the collection period
can be skipped using this action.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/billing/invoices/{invoiceId}/taxes/recalculate
- version：v1
- operationId：recalculateInvoiceTaxAction
- 摘要：Recalculate an invoice's tax amounts
- 说明：Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)

Note: charges might apply, depending on the tax provider.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/billing/invoices/{invoiceId}/void
- version：v1
- operationId：voidInvoiceAction
- 摘要：Void an invoice
- 说明：Void an invoice

Only invoices that have been alread issued can be voided.

Voiding an invoice will mark it as voided, the user can specify how to handle the voided line items.
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/VoidInvoiceActionInput
- 响应码：200、400、401、403、404、412、500、503、default

#### GET /api/v1/billing/profiles
- version：v1
- operationId：listBillingProfiles
- 摘要：List billing profiles
- 说明：List all billing profiles matching the specified filters.

The expand option can be used to include additional information (besides the billing profile)
in the response. For example by adding the expand=apps option the apps used by the billing profile
will be included in the response.
- 参数：
- query 参数 **includeArchived**（必填：否）：boolean，—
- query 参数 **expand**（必填：否）：array<#/components/schemas/BillingProfileExpand>，—
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/BillingProfileOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/billing/profiles
- version：v1
- operationId：createBillingProfile
- 摘要：Create a new billing profile
- 说明：Create a new billing profile

Billing profiles are representations of a customer's billing information. Customer overrides
can be applied to a billing profile to customize the billing behavior for a specific customer.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingProfileCreate
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/billing/profiles/{id}
- version：v1
- operationId：getBillingProfile
- 摘要：Get a billing profile
- 说明：Get a billing profile by id.

The expand option can be used to include additional information (besides the billing profile)
in the response. For example by adding the expand=apps option the apps used by the billing profile
will be included in the response.
- 参数：
- path 参数 **id**（必填：是）：string，—
- query 参数 **expand**（必填：否）：array<#/components/schemas/BillingProfileExpand>，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/billing/profiles/{id}
- version：v1
- operationId：updateBillingProfile
- 摘要：Update a billing profile
- 说明：Update a billing profile by id.

The apps field cannot be updated directly, if an app change is desired a new
profile should be created.
- 参数：
- path 参数 **id**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/BillingProfileReplaceUpdateWithWorkflow
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/billing/profiles/{id}
- version：v1
- operationId：deleteBillingProfile
- 摘要：Delete a billing profile
- 说明：Delete a billing profile by id.

Only such billing profiles can be deleted that are:
- not the default one
- not pinned to any customer using customer overrides
- only have finalized invoices
- 参数：
- path 参数 **id**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default


### app（应用）

#### GET /api/v1/apps
- version：v1
- operationId：listApps
- 摘要：List apps
- 说明：List apps.
- 参数：
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/apps/custom-invoicing/{invoiceId}/draft/synchronized
- version：v1
- operationId：appCustomInvoicingDraftSynchronized
- 摘要：Submit draft synchronization results
- 说明：—
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CustomInvoicingDraftSynchronizedRequest
- 响应码：204、400、401、403、412、500、503、default

#### POST /api/v1/apps/custom-invoicing/{invoiceId}/issuing/synchronized
- version：v1
- operationId：appCustomInvoicingIssuingSynchronized
- 摘要：Submit issuing synchronization results
- 说明：—
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CustomInvoicingFinalizedRequest
- 响应码：204、400、401、403、412、500、503、default

#### POST /api/v1/apps/custom-invoicing/{invoiceId}/payment/status
- version：v1
- operationId：appCustomInvoicingUpdatePaymentStatus
- 摘要：Update payment status
- 说明：—
- 参数：
- path 参数 **invoiceId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/CustomInvoicingUpdatePaymentStatusRequest
- 响应码：204、400、401、403、412、500、503、default

#### GET /api/v1/apps/{id}
- version：v1
- operationId：getApp
- 摘要：Get app
- 说明：Get the app.
- 参数：
- path 参数 **id**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/apps/{id}
- version：v1
- operationId：updateApp
- 摘要：Update app
- 说明：Update an app.
- 参数：
- path 参数 **id**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/AppReplaceUpdate
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/apps/{id}
- version：v1
- operationId：uninstallApp
- 摘要：Uninstall app
- 说明：Uninstall an app.
- 参数：
- path 参数 **id**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### POST /api/v1/apps/{id}/stripe/webhook
- version：v1
- operationId：appStripeWebhook
- 摘要：Stripe webhook
- 说明：Handle stripe webhooks for apps.
- 参数：
- path 参数 **id**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/StripeWebhookEvent
- 响应码：200、400、401、403、404、412、500、503、default

#### GET /api/v1/marketplace/listings
- version：v1
- operationId：listMarketplaceListings
- 摘要：List available apps
- 说明：List available apps of the app marketplace.
- 参数：
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/marketplace/listings/{type}
- version：v1
- operationId：getMarketplaceListing
- 摘要：Get app details by type
- 说明：Get a marketplace listing by type.
- 参数：
- path 参数 **type**（必填：是）：#/components/schemas/AppType，—
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/marketplace/listings/{type}/install
- version：v1
- operationId：marketplaceAppInstall
- 摘要：Install app
- 说明：Install an app from the marketplace.
- 参数：
- path 参数 **type**（必填：是）：#/components/schemas/AppType，The type of the app to install.
- 请求体：
- required: 是
- application/json：#/components/schemas/MarketplaceInstallRequestPayload
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/marketplace/listings/{type}/install/apikey
- version：v1
- operationId：marketplaceAppAPIKeyInstall
- 摘要：Install app via API key
- 说明：Install an marketplace app via API Key.
- 参数：
- path 参数 **type**（必填：是）：#/components/schemas/AppType，The type of the app to install.
- 请求体：
- required: 是
- application/json：object
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/marketplace/listings/{type}/install/oauth2
- version：v1
- operationId：marketplaceOAuth2InstallGetURL
- 摘要：Get OAuth2 install URL
- 说明：Install an app via OAuth.
Returns a URL to start the OAuth 2.0 flow.
- 参数：
- path 参数 **type**（必填：是）：#/components/schemas/AppType，—
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/marketplace/listings/{type}/install/oauth2/authorize
- version：v1
- operationId：marketplaceOAuth2InstallAuthorize
- 摘要：Install app via OAuth2
- 说明：Authorize OAuth2 code.
Verifies the OAuth code and exchanges it for a token and refresh token
- 参数：
- query 参数 **state**（必填：否）：string，Required if the "state" parameter was present in the client authorization request.
The exact value received from the client:

Unique, randomly generated, opaque, and non-guessable string that is sent
when starting an authentication request and validated when processing the response.
- query 参数 **code**（必填：否）：string，Authorization code which the client will later exchange for an access token.
Required with the success response.
- query 参数 **error**（必填：否）：#/components/schemas/OAuth2AuthorizationCodeGrantErrorType，Error code.
Required with the error response.
- query 参数 **error_description**（必填：否）：string，Optional human-readable text providing additional information,
used to assist the client developer in understanding the error that occurred.
- query 参数 **error_uri**（必填：否）：string，Optional uri identifying a human-readable web page with
information about the error, used to provide the client
developer with additional information about the error
- path 参数 **type**（必填：是）：#/components/schemas/AppType，The type of the app to install.
- 请求体：
- 无
- 响应码：303、400、401、403、412、500、503、default

#### POST /api/v1/stripe/checkout/sessions
- version：v1
- operationId：createStripeCheckoutSession
- 摘要：Create checkout session
- 说明：Create checkout session.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/CreateStripeCheckoutSessionRequest
- 响应码：201、400、401、403、404、412、500、503、default


### notification（通知）

#### GET /api/v1/notification/channels
- version：v1
- operationId：listNotificationChannels
- 摘要：List notification channels
- 说明：List all notification channels.
- 参数：
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted notification channels in response.

Usage: `?includeDeleted=true`
- query 参数 **includeDisabled**（必填：否）：boolean，Include disabled notification channels in response.

Usage: `?includeDisabled=false`
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/NotificationChannelOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/notification/channels
- version：v1
- operationId：createNotificationChannel
- 摘要：Create a notification channel
- 说明：Create a new notification channel.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/NotificationChannelCreateRequest
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/notification/channels/{channelId}
- version：v1
- operationId：getNotificationChannel
- 摘要：Get notification channel
- 说明：Get a notification channel by id.
- 参数：
- path 参数 **channelId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/notification/channels/{channelId}
- version：v1
- operationId：updateNotificationChannel
- 摘要：Update a notification channel
- 说明：Update notification channel.
- 参数：
- path 参数 **channelId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/NotificationChannelCreateRequest
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/notification/channels/{channelId}
- version：v1
- operationId：deleteNotificationChannel
- 摘要：Delete a notification channel
- 说明：Soft delete notification channel by id.

Once a notification channel is deleted it cannot be undeleted.
- 参数：
- path 参数 **channelId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### GET /api/v1/notification/events
- version：v1
- operationId：listNotificationEvents
- 摘要：List notification events
- 说明：List all notification events.
- 参数：
- query 参数 **from**（必填：否）：string，Start date-time in RFC 3339 format.
Inclusive.
- query 参数 **to**（必填：否）：string，End date-time in RFC 3339 format.
Inclusive.
- query 参数 **feature**（必填：否）：array<string>，Filtering by multiple feature ids or keys.

Usage: `?feature=feature-1&feature=feature-2`
- query 参数 **subject**（必填：否）：array<string>，Filtering by multiple subject ids or keys.

Usage: `?subject=subject-1&subject=subject-2`
- query 参数 **rule**（必填：否）：array<string>，Filtering by multiple rule ids.

Usage: `?rule=01J8J2XYZ2N5WBYK09EDZFBSZM&rule=01J8J4R4VZH180KRKQ63NB2VA5`
- query 参数 **channel**（必填：否）：array<string>，Filtering by multiple channel ids.

Usage: `?channel=01J8J4RXH778XB056JS088PCYT&channel=01J8J4S1R1G9EVN62RG23A9M6J`
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/NotificationEventOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/notification/events/{eventId}
- version：v1
- operationId：getNotificationEvent
- 摘要：Get notification event
- 说明：Get a notification event by id.
- 参数：
- path 参数 **eventId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### POST /api/v1/notification/events/{eventId}/resend
- version：v1
- operationId：resendNotificationEvent
- 摘要：Re-send notification event
- 说明：—
- 参数：
- path 参数 **eventId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/NotificationEventResendRequest
- 响应码：202、400、401、403、404、412、500、503、default

#### GET /api/v1/notification/rules
- version：v1
- operationId：listNotificationRules
- 摘要：List notification rules
- 说明：List all notification rules.
- 参数：
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted notification rules in response.

Usage: `?includeDeleted=true`
- query 参数 **includeDisabled**（必填：否）：boolean，Include disabled notification rules in response.

Usage: `?includeDisabled=false`
- query 参数 **feature**（必填：否）：array<string>，Filtering by multiple feature ids/keys.

Usage: `?feature=feature-1&feature=feature-2`
- query 参数 **channel**（必填：否）：array<string>，Filtering by multiple notifiaction channel ids.

Usage: `?channel=01ARZ3NDEKTSV4RRFFQ69G5FAV&channel=01J8J2Y5X4NNGQS32CF81W95E3`
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/NotificationRuleOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/notification/rules
- version：v1
- operationId：createNotificationRule
- 摘要：Create a notification rule
- 说明：Create a new notification rule.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/NotificationRuleCreateRequest
- 响应码：201、400、401、403、412、500、503、default

#### GET /api/v1/notification/rules/{ruleId}
- version：v1
- operationId：getNotificationRule
- 摘要：Get notification rule
- 说明：Get a notification rule by id.
- 参数：
- path 参数 **ruleId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v1/notification/rules/{ruleId}
- version：v1
- operationId：updateNotificationRule
- 摘要：Update a notification rule
- 说明：Update notification rule.
- 参数：
- path 参数 **ruleId**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/NotificationRuleCreateRequest
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v1/notification/rules/{ruleId}
- version：v1
- operationId：deleteNotificationRule
- 摘要：Delete a notification rule
- 说明：Soft delete notification rule by id.

Once a notification rule is deleted it cannot be undeleted.
- 参数：
- path 参数 **ruleId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### POST /api/v1/notification/rules/{ruleId}/test
- version：v1
- operationId：testNotificationRule
- 摘要：Test notification rule
- 说明：Test a notification rule by sending a test event with random data.
- 参数：
- path 参数 **ruleId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：201、400、401、403、404、412、500、503、default


#### GET /api/v1/portal/tokens
- version：v1
- operationId：listPortalTokens
- 摘要：List consumer portal tokens
- 说明：List tokens.
- 参数：
- query 参数 **limit**（必填：否）：integer，—
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/portal/tokens
- version：v1
- operationId：createPortalToken
- 摘要：Create consumer portal token
- 说明：Create a consumer portal token.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：#/components/schemas/PortalToken
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v1/portal/tokens/invalidate
- version：v1
- operationId：invalidatePortalTokens
- 摘要：Invalidate portal tokens
- 说明：Invalidates consumer portal tokens by ID or subject.
- 参数：
- 无
- 请求体：
- required: 是
- application/json：object
- 响应码：204、400、401、403、412、500、503、default


### system（系统）

#### GET /api/v1/debug/metrics
- version：v1
- operationId：getDebugMetrics
- 摘要：Get event metrics
- 说明：Returns debug metrics (in OpenMetrics format) like the number of ingested events since mindnight UTC.

The OpenMetrics Counter(s) reset every day at midnight UTC.
- 参数：
- 无
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/info/currencies
- version：v1
- operationId：listCurrencies
- 摘要：List supported currencies
- 说明：List all supported currencies.
- 参数：
- 无
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v1/info/progress/{id}
- version：v1
- operationId：getProgress
- 摘要：Get progress
- 说明：Get progress
- 参数：
- path 参数 **id**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default


## v2

### entitlement（权限/权益）

#### GET /api/v2/customers/{customerIdOrKey}/entitlements
- version：v2
- operationId：listCustomerEntitlementsV2
- 摘要：List customer entitlements
- 说明：List all entitlements for a customer. For checking entitlement access, use the /value endpoint instead.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- query 参数 **includeDeleted**（必填：否）：boolean，—
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/EntitlementOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v2/customers/{customerIdOrKey}/entitlements
- version：v2
- operationId：createCustomerEntitlementV2
- 摘要：Create a customer entitlement
- 说明：OpenMeter has three types of entitlements: metered, boolean, and static. The type property determines the type of entitlement. The underlying feature has to be compatible with the entitlement type specified in the request (e.g., a metered entitlement needs a feature associated with a meter).

- Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
- Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config).
- Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.

A given customer can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.

Once an entitlement is created you cannot modify it, only delete it.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- required: 是
- application/json：#/components/schemas/EntitlementV2CreateInputs
- 响应码：201、400、401、403、409、412、500、503、default

#### GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}
- version：v2
- operationId：getCustomerEntitlementV2
- 摘要：Get customer entitlement
- 说明：Get entitlement by feature key. For checking entitlement access, use the /value endpoint instead.
If featureKey is used, the entitlement is resolved for the current timestamp.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### DELETE /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}
- version：v2
- operationId：deleteCustomerEntitlementV2
- 摘要：Delete customer entitlement
- 说明：Deleting an entitlement revokes access to the associated feature. As a single customer can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well.
As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：string，—
- 请求体：
- 无
- 响应码：204、400、401、403、404、412、500、503、default

#### GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants
- version：v2
- operationId：listCustomerEntitlementGrantsV2
- 摘要：List customer entitlement grants
- 说明：List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：string，—
- query 参数 **includeDeleted**（必填：否）：boolean，—
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **offset**（必填：否）：integer，Number of items to skip.

Default is 0.
- query 参数 **limit**（必填：否）：integer，Number of items to return.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/GrantOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants
- version：v2
- operationId：createCustomerEntitlementGrantV2
- 摘要：Create customer entitlement grant
- 说明：Grants define a behavior of granting usage for a metered entitlement. They can have complicated recurrence and rollover rules, thanks to which you can define a wide range of access patterns with a single grant, in most cases you don't have to periodically create new grants. You can only issue grants for active metered entitlements.

A grant defines a given amount of usage that can be consumed for the entitlement. The grant is in effect between its effective date and its expiration date. Specifying both is mandatory for new grants.

Grants have a priority setting that determines their order of use. Lower numbers have higher priority, with 0 being the highest priority.

Grants can have a recurrence setting intended to automate the manual reissuing of grants. For example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover settings).

Rollover settings define what happens to the remaining balance of a grant at a reset. Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

Grants cannot be changed once created, only deleted. This is to ensure that balance is deterministic regardless of when it is queried.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/EntitlementGrantCreateInputV2
- 响应码：201、400、401、403、409、412、500、503、default

#### GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history
- version：v2
- operationId：getCustomerEntitlementHistoryV2
- 摘要：Get customer entitlement history
- 说明：Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.

BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.

WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：string，—
- query 参数 **from**（必填：否）：string，Start of time range to query entitlement: date-time in RFC 3339 format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter.
- query 参数 **to**（必填：否）：string，End of time range to query entitlement: date-time in RFC 3339 format. Defaults to now.
If not now then gets truncated to the granularity of the underlying meter.
- query 参数 **windowSize**（必填：是）：#/components/schemas/WindowSize，Windowsize
- query 参数 **windowTimeZone**（必填：否）：string，The timezone used when calculating the windows.
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### PUT /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override
- version：v2
- operationId：overrideCustomerEntitlementV2
- 摘要：Override customer entitlement
- 说明：Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided customer-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.

This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- 请求体：
- required: 是
- application/json：#/components/schemas/EntitlementV2CreateInputs
- 响应码：201、400、401、403、404、409、412、500、503、default

#### POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset
- version：v2
- operationId：resetCustomerEntitlementUsageV2
- 摘要：Reset customer entitlement
- 说明：Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the customers billing period to enforce usage based on their subscription.

Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：string，—
- 请求体：
- required: 是
- application/json：#/components/schemas/ResetEntitlementUsageInput
- 响应码：204、400、401、403、404、412、500、503、default

#### GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value
- version：v2
- operationId：getCustomerEntitlementValueV2
- 摘要：Get customer entitlement value
- 说明：Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
- 参数：
- path 参数 **customerIdOrKey**（必填：是）：#/components/schemas/ULIDOrExternalKey，—
- path 参数 **entitlementIdOrFeatureKey**（必填：是）：string，—
- query 参数 **time**（必填：否）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### GET /api/v2/entitlements
- version：v2
- operationId：listEntitlementsV2
- 摘要：List all entitlements
- 说明：List all entitlements for all the customers and features. This endpoint is intended for administrative purposes only.
To fetch the entitlements of a specific subject please use the /api/v2/customers/{customerIdOrKey}/entitlements endpoint.
- 参数：
- query 参数 **feature**（必填：否）：array<string>，Filtering by multiple features.

Usage: `?feature=feature-1&feature=feature-2`
- query 参数 **customerKeys**（必填：否）：array<string>，Filtering by multiple customers.

Usage: `?customerKeys=customer-1&customerKeys=customer-3`
- query 参数 **customerIds**（必填：否）：array<string>，Filtering by multiple customers.

Usage: `?customerIds=01K4WAQ0J99ZZ0MD75HXR112H8&customerIds=01K4WAQ0J99ZZ0MD75HXR112H9`
- query 参数 **entitlementType**（必填：否）：array<#/components/schemas/EntitlementType>，Filtering by multiple entitlement types.

Usage: `?entitlementType=metered&entitlementType=boolean`
- query 参数 **excludeInactive**（必填：否）：boolean，Exclude inactive entitlements in the response (those scheduled for later or earlier)
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **offset**（必填：否）：integer，Number of items to skip.

Default is 0.
- query 参数 **limit**（必填：否）：integer，Number of items to return.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/EntitlementOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default

#### GET /api/v2/entitlements/{entitlementId}
- version：v2
- operationId：getEntitlementByIdV2
- 摘要：Get entitlement by ID
- 说明：Get entitlement by ID.
- 参数：
- path 参数 **entitlementId**（必填：是）：string，—
- 请求体：
- 无
- 响应码：200、400、401、403、404、412、500、503、default

#### GET /api/v2/grants
- version：v2
- operationId：listGrantsV2
- 摘要：List grants
- 说明：List all grants for all the customers and entitlements. This endpoint is intended for administrative purposes only.
To fetch the grants of a specific entitlement please use the /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants endpoint.
If page is provided that takes precedence and the paginated response is returned.
- 参数：
- query 参数 **feature**（必填：否）：array<string>，Filtering by multiple features.

Usage: `?feature=feature-1&feature=feature-2`
- query 参数 **customer**（必填：否）：array<#/components/schemas/ULIDOrExternalKey>，Filtering by multiple customers (either by ID or key).

Usage: `?customer=customer-1&customer=customer-2`
- query 参数 **includeDeleted**（必填：否）：boolean，Include deleted
- query 参数 **page**（必填：否）：integer，Page index.

Default is 1.
- query 参数 **pageSize**（必填：否）：integer，The maximum number of items per page.

Default is 100.
- query 参数 **offset**（必填：否）：integer，Number of items to skip.

Default is 0.
- query 参数 **limit**（必填：否）：integer，Number of items to return.

Default is 100.
- query 参数 **order**（必填：否）：—，The order direction.
- query 参数 **orderBy**（必填：否）：#/components/schemas/GrantOrderBy，The order by field.
- 请求体：
- 无
- 响应码：200、400、401、403、412、500、503、default
