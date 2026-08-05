# OpenMeter API 接口清单（中文）

> 本文档由 OpenAPI 规范自动生成。接口含义为中文描述；参数与字段的结构信息（名称、位置、类型、是否必填）来自规范，说明文字保留规范原文以便核对。
> 仓库内三份规范：`api/openapi.yaml`（v1 经典）、`api/openapi.cloud.yaml`（云版本，接口与 v1 一致）、`api/v3/openapi.yaml`（v3 新一代，含 WeKnora 扩展）。

---

# 第一部分：V3 API（新一代）

## AI Usage


### POST `/ai-usage-batches`

**含义**：提交AI用量批次
（原文：Submit an AI usage batch）

> Submit a Canonical AI Usage Batch for settlement.  The first submit for a given `idempotency_key` returns HTTP 201 with the settled batch. An identical replay (same `idempotency_key` and `payload_hash`) returns HTTP 200 with the stored result. A replay with the same `idempotency_key` but a different `payload_hash` returns HTTP 409.

**请求体**（`AIUsageUsageBatchCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `idempotency_key` | string | 是 | Client-generated idempotency key. Replaying the same key with the same `payload_hash` returns the stored result with HTTP 200. A replay with a different `payload_hash` for the same key returns HTTP 409. |
| `payload_hash` | string | 是 | SHA-256 hex digest of the canonical request body, used for idempotency verification. |
| `billing_customer_id` |  | 是 | The billing customer (payer) in OpenMeter. |
| `subject_key` | string | 是 | The subject or tenant key that produced the usage. |
| `tenant_seq` | integer(int64) | 是 | Monotonic per-tenant sequence number for watermark tracking. Must be strictly increasing within a tenant. |
| `occurred_at` |  | 是 | When the business action occurred. Used for rate package version resolution. |
| `reservation_id` | string | 是 | Links to the WeKnora runtime reservation, if any. |
| `reservation_ceiling_credits` | integer(int64) | 是 | Caps the total Credit charge for this batch. The platform absorbs any excess above the ceiling. |
| `rate_package_version` | string | 是 | Rate package version snapshot used for settlement. |
| `billing_mode` |  | 是 | Billing mode for this batch. |
| `provider_managed` | boolean | 是 | Whether model resources are platform-managed. Set to `false` for bring-your-own-key (BYOK) models. |
| `lines` | array<AIUsageUsageLineCreate> | 是 | The individual resource consumption entries. Must contain at least one line for `component` billing mode. |

**响应**：200 UsageBatch response., 201 UsageBatch created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


### GET `/ai-usage-batches/{batchId}`

**含义**：查询AI用量批次
（原文：Get an AI usage batch）

> Retrieve a settled AI Usage Batch by its ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `batchId` | ULID | 是 |  |

**响应**：200 UsageBatch response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/customers/{customerId}/credit-balance`

**含义**：查询额度余额
（原文：Get AI usage credit balance）

> Get a customer's credit balance for AI usage. Returns the same balance model as the OpenMeter Credits endpoint but scoped to the AI Usage route.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `timestamp` | DateTime | 否 | Return the credit balance as of this timestamp. Defaults to now. |

**响应**：200 AICreditBalance response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/customers/{customerId}/credit-transactions`

**含义**：列出额度交易记录
（原文：List AI usage credit transactions）

> List credit transactions for a customer's AI usage. Returns the same transaction model as the OpenMeter Credits endpoint but scoped to the AI Usage route.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | CursorPaginationQueryPage | 否 |  |

**响应**：200 Cursor paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/customers/{customerId}/runtime-authorization`

**含义**：查询客户 AI 资源运行时授权
（原文：Get runtime authorization）

> Check whether a customer is authorized to consume AI resources.  Returns the current integer credit balance, reservation ceiling, and the covered tenant sequence watermark.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `filter` | AIUsageRuntimeAuthorizationQuery | 否 | Filter the authorization check by subject and reservation. |

**响应**：200 RuntimeAuthorization response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## Commerce


### GET `/checkout-sessions/{sessionId}`

**含义**：查询结账会话
（原文：Get checkout session）

> Retrieve a checkout session by its ID (for polling payment status after QR code expiry or page reload).

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `sessionId` | ULID | 是 |  |

**响应**：200 CheckoutSession response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/customers/{customerId}/offline-payments`

**含义**：创建线下付款
（原文：Create offline payment）

> Record an offline payment (bank transfer, enterprise remittance) for a customer. The payment is held for reconciliation before being applied to a receivable period.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`CommerceOfflinePaymentCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `idempotency_key` | string | 是 | Client-generated idempotency key. |
| `amount_fen` |  | 是 | Payment amount in fen. |
| `currency` |  | 是 | Currency of the payment. |
| `receivable_period_id` |  | 否 | The receivable period to apply this payment to, if applicable. |
| `external_reference` | string | 是 | External reference (bank transfer number, remittance advice). |
| `received_at` |  | 是 | When the payment was received. |
| `note` | string | 否 | Optional note from the submitter. |

**响应**：201 OfflinePayment created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 409 Conflict


### GET `/customers/{customerId}/receivable-periods`

**含义**：列出应收账期
（原文：List receivable periods）

> List receivable periods for a customer.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | CursorPaginationQueryPage | 否 |  |

**响应**：200 Cursor paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/customers/{customerId}/receivable-periods/{periodId}/external-invoice`

**含义**：更新外部发票
（原文：Update external invoice）

> Attach or update an external invoice reference on a receivable period.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |
| `periodId` | ULID | 是 |  |

**请求体**（`CommerceExternalInvoiceUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `idempotency_key` | string | 是 | Client-generated idempotency key for the update. |
| `invoice_number` | string | 是 | External invoice number or reference. |
| `invoice_url` | string | 否 | URL or identifier of the external invoice document. |
| `issuer` | string | 否 | The invoicing party or tax service. |
| `issued_at` |  | 否 | When the external invoice was issued. |

**响应**：200 ExternalInvoice updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/customers/{customerId}/wallet`

**含义**：查询钱包
（原文：Get customer wallet）

> Get a customer's Wallet view, including all credit buckets and recent transactions.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**响应**：200 Wallet response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/orders`

**含义**：创建订单
（原文：Create order）

> Create a new order (plan purchase, subscription renewal, or wallet top-up).  Returns HTTP 201 on first creation. Replaying the same idempotency key returns the stored order with HTTP 200.

**请求体**（`CommerceOrderCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `idempotency_key` | string | 是 | Client-generated idempotency key. Replaying the same key returns the stored order; a different payload for the same key returns HTTP 409. |
| `billing_customer_id` |  | 是 | The billing customer placing the order. |
| `kind` |  | 是 | The business type of the order. |
| `plan` |  | 否 | For `plan_purchase` or `subscription_renewal`: the plan and billing period being purchased. |
| `recharge_product_id` |  | 否 | For `wallet_top_up`: the recharge product being purchased. |
| `currency` |  | 是 | The currency for this order (three-letter ISO 4217). |

**响应**：200 Order response., 201 Order created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 409 Conflict


### GET `/orders/{orderId}`

**含义**：查询订单
（原文：Get order）

> Retrieve an order by its ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `orderId` | ULID | 是 |  |

**响应**：200 Order response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/orders/{orderId}/checkout-sessions`

**含义**：创建结账会话
（原文：Create checkout session）

> Create a checkout session for an order, initiating a payment attempt with the specified provider.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `orderId` | ULID | 是 |  |

**请求体**（`CommerceCheckoutSessionCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `idempotency_key` | string | 是 | Client-generated idempotency key for the checkout attempt. |
| `provider` |  | 是 | The payment channel to use for this checkout. |
| `external_reference` | string | 否 | For offline payments: the external reference (e.g. bank transfer number). |

**响应**：201 CheckoutSession created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/payment-providers/alipay/callback`

**含义**：支付宝支付回调
（原文：Alipay callback）

> Alipay payment callback. OpenMeter verifies the signature, confirms the payment fact, and fulfills the order.

**响应**：200 ProviderCallbackAck response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/payment-providers/wechat/callback`

**含义**：微信支付回调
（原文：WeChat Pay callback）

> WeChat Pay payment callback. OpenMeter verifies the signature, confirms the payment fact, and fulfills the order.

**响应**：200 ProviderCallbackAck response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/recharge-products`

**含义**：列出充值产品
（原文：List recharge products）

> List all active recharge products available for purchase.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `currency` | CurrencyCode | 否 | Filter by currency to show only products priced in the customer's currency. |

**响应**：200 RechargeProductList response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/refunds`

**含义**：创建退款
（原文：Create refund）

> Create a refund request for an order.

**请求体**（`CommerceRefundCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `idempotency_key` | string | 是 | Client-generated idempotency key. |
| `billing_customer_id` |  | 是 | The billing customer requesting the refund. |
| `order_id` |  | 是 | The original order being refunded (must be a wallet_top_up). |
| `amount_fen` |  | 是 | Refund amount in fen. Must not exceed the unspent portion of the original payment, in multiples of the credit quantum. |
| `reason` | string | 是 | The reason for the refund. |

**响应**：201 Refund created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 409 Conflict


### GET `/refunds/{refundId}`

**含义**：查询退款
（原文：Get refund）

> Retrieve a refund by its ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `refundId` | ULID | 是 |  |

**响应**：200 Refund response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## Metering Events


### GET `/openmeter/events`

**含义**：列出事件
（原文：List metering events）

> List ingested events.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | CursorPaginationQueryPage | 否 |  |
| `filter` | ListEventsParamsFilter | 否 | Filter events returned in the response.  To filter events by subject add the following query param: filter[subject][eq]=customer-1 |
| `sort` | SortQuery | 否 | Sort events returned in the response. Supported sort attributes are:  - `time` (default) - `ingested_at` - `stored_at`  When omitted, events are sorted by `time desc` (most recent first). When a sort field is provided without a suffix, it sorts descending. Append the `asc` suffix to sort ascending, or the `desc` suffix to sort descending. |

**响应**：200 Cursor paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/events`

**含义**：创建事件
（原文：Ingest metering events）

> Ingests an event or batch of events following the CloudEvents specification.

**响应**：202 The events have been ingested and are being processed asynchronously., 400 Bad Request, 401 Unauthorized, 403 Forbidden


## Meters


### POST `/openmeter/meters`

**含义**：创建计量器
（原文：Create meter）

> Create a meter.

**请求体**（`CreateMeterRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `key` | ResourceKey | 是 |  |
| `aggregation` |  | 是 | The aggregation type to use for the meter. |
| `event_type` | string | 是 | The event type to include in the aggregation. |
| `events_from` |  | 否 | The date since the meter should include events. Useful to skip old events. If not specified, all historical events are included. |
| `value_property` | string | 否 | JSONPath expression to extract the value from the ingested event's data property.  The ingested value for sum, avg, min, and max aggregations is a number or a string that can be parsed to a number.  For unique_count aggregation, the ingested value must be a string. For count aggregation the value_property is ignored. |
| `dimensions` | object | 否 | Named JSONPath expressions to extract the group by values from the event data.  Keys must be unique and consist only alphanumeric and underscore characters. |

**响应**：201 Meter created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/meters`

**含义**：列出计量器
（原文：List meters）

> List meters.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort meters returned in the response. Supported sort attributes are:  - `key` - `name` - `aggregation` - `created_at` (default) - `updated_at`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `filter` | ListMetersParamsFilter | 否 | Filter meters returned in the response.  To filter meters by key add the following query param: filter[key]=my-meter-key |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/meters/{meterId}`

**含义**：查询计量器
（原文：Get meter）

> Get a meter by ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterId` | ULID | 是 |  |

**响应**：200 Meter response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/meters/{meterId}`

**含义**：更新计量器
（原文：Update meter）

> Update a meter.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterId` | ULID | 是 |  |

**请求体**（`UpdateMeterRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 否 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `dimensions` | object | 否 | Named JSONPath expressions to extract the group by values from the event data.  Keys must be unique and consist only alphanumeric and underscore characters. |

**响应**：200 Meter updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### DELETE `/openmeter/meters/{meterId}`

**含义**：删除计量器
（原文：Delete meter）

> Delete a meter.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/meters/{meterId}/query`

**含义**：查询计量器
（原文：Query meter）

> Query a meter for usage.  Set `Accept: application/json` (the default) to get a structured JSON response. Set `Accept: text/csv` to download the same data as a CSV file suitable for spreadsheets. The CSV columns, in order, are:  `from, to, [subject,] [customer_id, customer_key, customer_name,] <dimensions...>, value`  The `subject` column is emitted only when `subject` is in the query's `group_by_dimensions`. The three `customer_*` columns are emitted together only when `customer_id` is in the query's `group_by_dimensions`.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterId` | ULID | 是 |  |

**请求体**（`MeterQueryRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `from` |  | 否 | The start of the period the usage is queried from. |
| `to` |  | 否 | The end of the period the usage is queried to. |
| `granularity` |  | 否 | The size of the time buckets to group the usage into. If not specified, the usage is aggregated over the entire period. |
| `time_zone` | string | 否 | The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). The time zone is used to determine the start and end of the time buckets. If not specified, the UTC timezone will be used. |
| `group_by_dimensions` | array<string> | 否 | The dimensions to group the results by. |
| `filters` |  | 否 | Filters to apply to the query. |

**响应**：200 The request has succeeded., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Apps


### GET `/openmeter/app-catalog`

**含义**：列出应用目录
（原文：List app catalog）

> List available apps.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/app-catalog/install`

**含义**：从应用目录安装应用
（原文：Install app from the catalog）

> Install an app from the catalog.

**响应**：201 InstallAppResponse created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/app-catalog/{appType}`

**含义**：查询应用目录
（原文：Get app catalog item by type）

> Get an app catalog item by type.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `appType` | BillingAppType | 是 |  |

**响应**：200 AppCatalogItem response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/apps`

**含义**：列出应用
（原文：List apps）

> List installed apps.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/apps/{appId}`

**含义**：查询应用
（原文：Get app）

> Get an installed app.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `appId` | ULID | 是 |  |

**响应**：200 App response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### DELETE `/openmeter/apps/{appId}`

**含义**：删除应用
（原文：Uninstall app）

> Uninstall an app by ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `appId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/apps/{appId}`

**含义**：更新应用
（原文：Update app）

> Update an installed app.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `appId` | ULID | 是 |  |

**响应**：200 App updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Billing Settings


### GET `/openmeter/billing/invoices`

**含义**：列出发票
（原文：List billing invoices）

> List billing invoices.  Returns a page of invoices. Gathering invoices are never included. Use `filter` to narrow by status, customer, dates, or service period start. Use `sort` to control ordering.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort invoices returned in the response. Supported sort attributes:  - `issued_at` - `created_at` (default) - `service_period_start`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `filter` | ListInvoicesParamsFilter | 否 | Filter invoices returned in the response.  Examples:  - `filter[status][oeq]=draft,issued` - `filter[customer_id]=01KPDB8K...` - `filter[issued_at][gte]=2024-01-01T00:00:00Z` |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/billing/invoices/{invoiceId}`

**含义**：查询发票
（原文：Get a billing invoice）

> Get a billing invoice by ID.  Returns the full invoice resource including line items, status details, totals, and workflow configuration snapshot.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | ULID | 是 |  |

**响应**：200 Invoice response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/billing/invoices/{invoiceId}`

**含义**：更新发票
（原文：Update a billing invoice）

> Update a billing invoice.  Only the mutable fields of the invoice can be edited: description, labels, supplier, customer, workflow settings, and top-level lines. Top-level lines are matched by `id`; lines without an `id` are created, and existing lines omitted from `lines` are deleted. Detailed (child) lines are always computed and cannot be edited directly. Only invoices in draft status can be updated.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | ULID | 是 |  |

**响应**：200 Invoice updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### DELETE `/openmeter/billing/invoices/{invoiceId}`

**含义**：删除发票
（原文：Delete a billing invoice）

> Delete a billing invoice.  Only standard invoices in draft status can be deleted. Deleting an invoice will also delete all associated line items and workflow configuration.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/billing/invoices/{invoiceId}/advance`

**含义**：推进发票到下一状态
（原文：Advance billing invoice's next status）

> Advance a billing invoice.  Advances the invoice to the next workflow state. The next state is determined by the invoice's current status and workflow configuration. Only invoices in draft or issued status can be advanced.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | ULID | 是 |  |

**响应**：200 The updated invoice after advancing to the next state., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/billing/invoices/{invoiceId}/approve`

**含义**：发送发票
（原文：Send the invoice to the customer）

> Approve a billing invoice.  This call instantly sends the invoice to the customer using the configured billing profile app.  This call is valid in two invoice statuses:  - draft: the invoice will be sent to the customer, the invoice state becomes issued - manual_approval_needed: the invoice will be sent to the customer, the invoice state becomes issued

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | ULID | 是 |  |

**响应**：200 The updated invoice after sending to the customer., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/billing/invoices/{invoiceId}/retry`

**含义**：重试推进发票状态
（原文：Retry advancing the invoice after a failed attempt）

> Retry sending a billing invoice.  Retry advancing the invoice after a failed attempt.  The action can be called when the invoice's statusDetails' actions field contain the "retry" action.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | ULID | 是 |  |

**响应**：200 The updated invoice after retrying the action., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/billing/invoices/{invoiceId}/snapshot-quantities`

**含义**：对发票按量明细进行快照
（原文：Snapshot quantities for usage based line items）

> Snapshot quantities for usage-based line items.  This call will snapshot the quantities for all usage based line items in the invoice.  This call is only valid in draft.waiting_for_collection status, where the collection period can be skipped using this action.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | ULID | 是 |  |

**响应**：200 The updated invoice with the snapshot quantities for usage based line
items., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/profiles`

**含义**：列出账单档案
（原文：List billing profiles）

> List billing profiles.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/profiles`

**含义**：创建账单档案
（原文：Create a new billing profile）

> Create a new billing profile.  Billing profiles contain the settings for billing and controls invoice generation. An organization can have multiple billing profiles defined. A billing profile is linked to a specific app. This association is established during the billing profile's creation and remains immutable.

**请求体**（`CreateBillingProfileRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `supplier` |  | 是 | The name and contact information for the supplier this billing profile represents |
| `workflow` |  | 是 | The billing workflow settings for this profile |
| `apps` |  | 是 | The applications used by this billing profile. |
| `default` | boolean | 是 | Whether this is the default profile. |

**响应**：201 BillingProfile created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/profiles/{id}`

**含义**：查询账单档案
（原文：Get a billing profile）

> Get a billing profile.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | ULID | 是 |  |

**响应**：200 BillingProfile response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/profiles/{id}`

**含义**：更新账单档案
（原文：Update a billing profile）

> Update a billing profile.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | ULID | 是 |  |

**请求体**（`UpsertBillingProfileRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `supplier` |  | 是 | The name and contact information for the supplier this billing profile represents |
| `workflow` |  | 是 | The billing workflow settings for this profile |
| `default` | boolean | 是 | Whether this is the default profile. |

**响应**：200 BillingProfile updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### DELETE `/openmeter/profiles/{id}`

**含义**：删除账单档案
（原文：Delete a billing profile）

> Delete a billing profile.  Only such billing profiles can be deleted that are:  - not the default profile - not pinned to any customer using customer overrides - only have finalized invoices

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Currencies


### GET `/openmeter/currencies`

**含义**：列出货币
（原文：List currencies）

> List currencies supported by the billing system.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort currencies returned in the response. Supported sort attributes are:  - `code` (default) - `name`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `filter` | ListCurrenciesParamsFilter | 否 | Filter currencies returned in the response.  To filter currencies by type add the following query param: filter[type]=custom |
| `expand` | array<BillingCurrencyExpand> | 否 | Expand the currencies returned in the response.  To include the currently-active cost basis add: expand=cost_basis |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/currencies/custom`

**含义**：创建自定义
（原文：Create custom currency）

> Create a custom currency. This operation allows defining your own custom currency for billing purposes.

**请求体**（`CreateCurrencyCustomRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | The name of the currency. It should be a human-readable string that represents the name of the currency, such as "US Dollar" or "Euro". |
| `symbol` | string | 否 | The symbol of the currency. It should be a string that represents the symbol of the currency, such as "$" for US Dollar or "€" for Euro. |
| `precision` | integer(uint32) | 是 | The precision of the currency. It should be a number that represents the number of decimal places used for the currency, such as 2 for US Dollar or Euro. |
| `decimal_mark` | string | 是 | The decimal mark for the currency. It should be a string that represents the decimal mark of the currency, such as "." for US Dollar or "," for Euro. |
| `thousand_separator` | string | 是 | The thousand separator for the currency. It should be a string that represents the thousand separator of the currency, such as "," for US Dollar or "." for Euro. |
| `code` | BillingCurrencyCodeCustom | 是 |  |

**响应**：201 CurrencyCustom created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/currencies/custom/{currencyId}`

**含义**：查询自定义
（原文：Get custom currency）

> Get a custom currency.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `currencyId` | ULID | 是 |  |

**响应**：200 CurrencyCustom response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/currencies/custom/{currencyId}/cost-bases`

**含义**：列出成本基准
（原文：List cost bases）

> List cost bases for a currency. For custom currencies, there can be multiple cost bases with different `effective_from` dates.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `currencyId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `filter` | ListCostBasesParamsFilter | 否 | Filter cost bases returned in the response.  To filter cost bases by fiat currency code add the following query param: filter[fiat_code]=USD |
| `page` | object | 否 | Determines which page of the collection to retrieve. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/currencies/custom/{currencyId}/cost-bases`

**含义**：创建成本基准
（原文：Create cost basis）

> Create a cost basis for a currency.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `currencyId` | ULID | 是 |  |

**请求体**（`CreateCostBasisRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `fiat_code` |  | 是 | The fiat currency code for the cost basis. |
| `rate` |  | 是 | The cost rate for the currency. |
| `effective_from` |  | 否 | An ISO-8601 timestamp representation of the date from which the cost basis is effective. If not provided, it will be effective immediately and will be set to `now` by the system. |
| `effective_to` |  | 否 | An ISO-8601 timestamp representation of the date until which the cost basis is effective. If provided, it must be later than `effective_from`. If not provided, it remains effective until superseded. |

**响应**：201 CostBasis created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


## OpenMeter Customers


### POST `/openmeter/customers`

**含义**：创建客户
（原文：Create customer）

**请求体**（`CreateCustomerRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `key` | ExternalResourceKey | 是 |  |
| `usage_attribution` |  | 否 | Mapping to attribute metered usage to the customer by the event subject. |
| `primary_email` | string | 否 | The primary email address of the customer. |
| `currency` |  | 否 | Currency of the customer. Used for billing, tax and invoicing. |
| `billing_address` |  | 否 | The billing address of the customer. Used for tax and invoicing. |

**响应**：201 Customer created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/customers`

**含义**：列出客户
（原文：List customers）

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort customers returned in the response. Supported sort attributes are:  - `id` - `name` (default) - `created_at`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `filter` | ListCustomersParamsFilter | 否 | Filter customers returned in the response.  To filter customers by key add the following query param: filter[key]=my-db-id |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/customers/{customerId}`

**含义**：查询客户
（原文：Get customer）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**响应**：200 Customer response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/customers/{customerId}`

**含义**：创建或更新客户
（原文：Upsert customer）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`UpsertCustomerRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `usage_attribution` |  | 否 | Mapping to attribute metered usage to the customer by the event subject. |
| `primary_email` | string | 否 | The primary email address of the customer. |
| `currency` |  | 否 | Currency of the customer. Used for billing, tax and invoicing. |
| `billing_address` |  | 否 | The billing address of the customer. Used for tax and invoicing. |

**响应**：200 Customer upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### DELETE `/openmeter/customers/{customerId}`

**含义**：删除客户
（原文：Delete customer）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/customers/{customerId}/billing`

**含义**：查询账单
（原文：Get customer billing data）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**响应**：200 CustomerBillingData response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/customers/{customerId}/billing`

**含义**：更新账单
（原文：Update customer billing data）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`UpsertCustomerBillingDataRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `billing_profile` |  | 否 | The billing profile for the customer.  If not provided, the default billing profile will be used. |
| `app_data` |  | 否 | App customer data. |

**响应**：200 CustomerBillingData upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### PUT `/openmeter/customers/{customerId}/billing/app-data`

**含义**：更新应用数据
（原文：Update customer billing app data）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`UpsertAppCustomerDataRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `stripe` |  | 否 | Used if the customer has a linked Stripe app. |
| `external_invoicing` |  | 否 | Used if the customer has a linked external invoicing app. |

**响应**：200 AppCustomerData upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### POST `/openmeter/customers/{customerId}/billing/stripe/checkout-sessions`

**含义**：创建结账会话
（原文：Create Stripe Checkout Session）

> Create a [Stripe Checkout Session](https://docs.stripe.com/payments/checkout) for the customer.  Creates a Checkout Session for collecting payment method information from customers. The session operates in "setup" mode, which collects payment details without charging the customer immediately. The collected payment method can be used for future subscription billing.  For hosted checkout sessions, redirect customers to the returned URL. For embedded sessions, use the client_secret to initialize Stripe.js in your application.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`BillingCustomerStripeCreateCheckoutSessionRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `stripe_options` |  | 是 | Options for configuring the Stripe Checkout Session.  These options are passed directly to Stripe's [checkout session creation API](https://docs.stripe.com/api/checkout/sessions/create). |

**响应**：201 CreateStripeCheckoutSessionResult created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### POST `/openmeter/customers/{customerId}/billing/stripe/portal-sessions`

**含义**：创建门户会话
（原文：Create Stripe customer portal session）

> Create Stripe Customer Portal Session.  Useful to redirect the customer to the Stripe Customer Portal to manage their payment methods, change their billing address and access their invoice history. Only returns URL if the customer billing profile is linked to a stripe app and customer.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`BillingCustomerStripeCreateCustomerPortalSessionRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `stripe_options` |  | 是 | Options for configuring the Stripe Customer Portal Session. |

**响应**：201 CreateStripeCustomerPortalSessionResult created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### GET `/openmeter/customers/{customerId}/charges`

**含义**：列出费用
（原文：List customer charges）

> List customer charges.  Returns the customer's charges that are represented as either flat fee or usage-based charges.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort charges returned in the response.  Supported sort attributes are:  - `id` - `created_at` - `service_period.from` - `billing_period.from` |
| `filter` | ListChargesParamsFilter | 否 | Filter charges.  To filter charges by status add the following query param: `filter[status][oeq]=created,active` |
| `expand` | array<BillingChargesExpand> | 否 | Expand full objects for referenced entities.  Supported values are:  - `real_time_usage`: Expand the charge's real-time usage. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/customers/{customerId}/charges`

**含义**：创建费用
（原文：Create customer charge）

> Create customer charge.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**响应**：201 Charge created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/customers/{customerId}/credits/adjustments`

**含义**：创建调整
（原文：Create a credit adjustment）

> A credit adjustment can be used to make manual adjustments to a customer's credit balance.  Supported use-cases:  - Usage correction

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`CreateCreditAdjustmentRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `currency` |  | 是 | The currency of the granted credits. |
| `amount` |  | 是 | Granted credit amount. |

**响应**：201 CreditAdjustment created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/customers/{customerId}/credits/balance`

**含义**：查询余额
（原文：Get a customer's credit balance）

> Get a credit balance.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `timestamp` | DateTime | 否 | Return the credit balance as of this timestamp.  Defaults to the current time. Historical responses return `live` as zero because live charge impacts are only available for current balances. |
| `filter` | GetCreditBalanceParamsFilter | 否 |  |

**响应**：200 CreditBalances response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/customers/{customerId}/credits/grants`

**含义**：创建授予
（原文：Create a new credit grant）

> Create a new credit grant. A credit grant represents an allocation of prepaid credits to a customer.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**请求体**（`CreateCreditGrantRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `funding_method` |  | 是 | Funding method of the grant. |
| `currency` |  | 是 | The currency of the granted credits. |
| `amount` |  | 是 | Granted credit amount. |
| `purchase` |  | 否 | Present when a funding workflow applies (funding_method is not `none`). |
| `tax_config` |  | 否 | Tax configuration for the grant.  For `invoice` and `external` funding methods, tax configuration should be provided to ensure correct revenue recognition. When not provided, the default credit grant tax code is applied, if that's not set the global default taxcode is used. |
| `filters` | CreateCreditGrantFilters | 否 |  |
| `priority` | integer(int16) | 否 | Draw-down priority of the grant. Lower values have higher priority. |
| `effective_at` |  | 否 | The timestamp when the credit grant becomes effective.  Defaults to the current date and time. |
| `expires_after` |  | 否 | The duration after which the credit grant expires.  Defaults to never expiring. |
| `key` |  | 否 | Idempotency key for the credit grant creation request.  Unique per customer: reusing the same key for the same customer returns an HTTP 409 Conflict instead of creating a duplicate grant, which makes create requests safe to retry. The same key may be reused across different customers. |

**响应**：201 CreditGrant created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


### GET `/openmeter/customers/{customerId}/credits/grants`

**含义**：列出授予
（原文：List credit grants）

> List credit grants.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `filter` | ListCreditGrantsParamsFilter | 否 | Filter credit grants returned in the response. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/customers/{customerId}/credits/grants/{creditGrantId}`

**含义**：查询授予
（原文：Get a credit grant）

> Get a credit grant.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |
| `creditGrantId` | ULID | 是 |  |

**响应**：200 CreditGrant response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external`

**含义**：更新external
（原文：Update credit grant external settlement status）

> Update the payment settlement status of an externally funded credit grant.  Use this endpoint to synchronize the payment state of an external payment with the system so that revenue recognition and credit availability work as expected.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |
| `creditGrantId` | ULID | 是 |  |

**请求体**（`UpdateCreditGrantExternalSettlementRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `status` |  | 是 | The new payment settlement status. |

**响应**：200 CreditGrant updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/customers/{customerId}/credits/grants/{creditGrantId}/void`

**含义**：作废授予
（原文：Void credit grant）

> Void a credit grant, forfeiting the remaining unused balance.  Voiding is a forward-looking, irreversible operation. Credits already consumed by usage remain unaffected — only the remaining balance is forfeited. The grant reads as `voided` status afterwards. Payment state is not adjusted when `payment_adjustment` is `none`, so invoice-backed or externally collected payments may still collect the original amount. Only `active` grants can be voided; voiding a pending, expired, or fully consumed grant returns a conflict. Retrying a successful void is an idempotent success.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |
| `creditGrantId` | ULID | 是 |  |

**请求体**（`VoidCreditGrantRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `payment_adjustment` |  | 否 | How voiding adjusts payment state related to the grant.  Currently only `none` is supported: voiding does not adjust invoices, payment authorization, settlement, payment intents, or external collection state. If payment later completes, the original invoiced amount may still be collected. |

**响应**：200 CreditGrant updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


### GET `/openmeter/customers/{customerId}/credits/transactions`

**含义**：列出交易记录
（原文：List credit transactions）

> List credit transactions for a customer.  Returns an immutable, chronological record of credit movements: funded credits and consumed credits. Transactions are returned in reverse chronological order by default.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | CursorPaginationQueryPage | 否 |  |
| `filter` | ListCreditTransactionsParamsFilter | 否 | Filter credit transactions returned in the response. |

**响应**：200 Cursor paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Defaults


### GET `/openmeter/defaults/tax-codes`

**含义**：查询组织默认税码
（原文：Get organization default tax codes）

**响应**：200 OrganizationDefaultTaxCodes response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/defaults/tax-codes`

**含义**：更新组织默认税码
（原文：Update organization default tax codes）

**请求体**（`UpdateOrganizationDefaultTaxCodesRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoicing_tax_code` |  | 否 | Default tax code for invoicing. |
| `credit_grant_tax_code` |  | 否 | Default tax code for credit grants. |

**响应**：200 OrganizationDefaultTaxCodes upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Entitlements


### GET `/openmeter/customers/{customerId}/entitlement-access`

**含义**：列出客户
（原文：List customer entitlement access）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | ULID | 是 |  |

**响应**：200 List the customer's active features and their access., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Features


### GET `/openmeter/features`

**含义**：列出功能特性
（原文：List features）

> List all features.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort features returned in the response. Supported sort attributes are:  - `key` - `name` - `created_at` (default) - `updated_at`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `filter` | ListFeatureParamsFilter | 否 | Filter features returned in the response.  To filter features by meter_id add the following query param: filter[meter_id][oeq]=<id> |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/features`

**含义**：创建功能特性
（原文：Create feature）

> Create a feature.

**请求体**（`CreateFeatureRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `key` | ResourceKey | 是 |  |
| `meter` |  | 否 | The meter that the feature is associated with and based on which usage is calculated. If not specified, the feature is static. |
| `unit_cost` |  | 否 | Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or "llm" to look up cost from the LLM cost database based on meter group-by properties. |

**响应**：201 Feature created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/features/{featureId}`

**含义**：查询功能特性
（原文：Get feature）

> Get a feature by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `featureId` | ULID | 是 |  |

**响应**：200 Feature response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### PATCH `/openmeter/features/{featureId}`

**含义**：更新功能特性
（原文：Update feature）

> Update a feature by id. Currently only the unit_cost field can be updated.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `featureId` | ULID | 是 |  |

**请求体**（`UpdateFeatureRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `unit_cost` |  | 否 | Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or "llm" to look up cost from the LLM cost database based on meter group-by properties. Set to `null` to clear the existing unit cost; omit to leave it unchanged. |

**响应**：200 Feature updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### DELETE `/openmeter/features/{featureId}`

**含义**：删除功能特性
（原文：Delete feature）

> Delete a feature by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `featureId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/features/{featureId}/cost/query`

**含义**：查询功能特性的成本
（原文：Query feature cost）

> Query the cost of a feature.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `featureId` | ULID | 是 |  |

**请求体**（`MeterQueryRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `from` |  | 否 | The start of the period the usage is queried from. |
| `to` |  | 否 | The end of the period the usage is queried to. |
| `granularity` |  | 否 | The size of the time buckets to group the usage into. If not specified, the usage is aggregated over the entire period. |
| `time_zone` | string | 否 | The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). The time zone is used to determine the start and end of the time buckets. If not specified, the UTC timezone will be used. |
| `group_by_dimensions` | array<string> | 否 | The dimensions to group the results by. |
| `filters` |  | 否 | Filters to apply to the query. |

**响应**：200 The request has succeeded., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Governance


### POST `/openmeter/governance/query`

**含义**：查询治理访问权限
（原文：Query governance access）

> Query feature access for a list of customers.  The endpoint resolves each provided identifier to a customer and returns the access status for the requested features, plus optional credit balance availability.  _Designed to be called on a fixed refresh interval and the query response is intended to be cached._

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | CursorPaginationQueryPage | 否 |  |

**请求体**（`GovernanceQueryRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `include_credits` | boolean | 否 | Whether to include credit balance availability for each resolved customer. When true, each feature evaluation includes credit balance checks.  Defaults to `false`. |
| `customer` |  | 是 |  |
| `feature` |  | 否 |  |

**响应**：200 The request has succeeded., 400 Bad Request, 401 Unauthorized, 403 Forbidden


## OpenMeter LLM Cost


### GET `/openmeter/llm-cost/overrides`

**含义**：列出覆盖价格
（原文：List LLM cost overrides）

> List per-namespace price overrides.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `filter` | ListLLMCostPricesParamsFilter | 否 |  |
| `page` | object | 否 | Determines which page of the collection to retrieve. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/llm-cost/overrides`

**含义**：创建覆盖价格
（原文：Create LLM cost override）

> Create a per-namespace price override.

**请求体**（`LLMCostOverrideCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `provider` | string | 是 | Provider/vendor of the model. |
| `model_id` | string | 是 | Canonical model identifier. |
| `model_name` | string | 否 | Human-readable model name. |
| `pricing` |  | 是 | Token pricing data. |
| `currency` |  | 是 | Currency code. |
| `effective_from` |  | 是 | When this override becomes effective. |
| `effective_to` |  | 否 | When this override expires. |

**响应**：201 Price created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### DELETE `/openmeter/llm-cost/overrides/{priceId}`

**含义**：删除覆盖价格
（原文：Delete LLM cost override）

> Delete a per-namespace price override.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `priceId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/llm-cost/prices`

**含义**：列出价格
（原文：List LLM cost prices）

> List global LLM cost prices. Returns prices with overrides applied if any.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `filter` | ListLLMCostPricesParamsFilter | 否 | Filter prices. |
| `sort` | SortQuery | 否 | Sort prices returned in the response. Supported sort attributes are:  - `id` - `provider.id` - `model.id` (default) - `effective_from` - `effective_to`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `page` | object | 否 | Determines which page of the collection to retrieve. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/llm-cost/prices/{priceId}`

**含义**：查询价格
（原文：Get LLM cost price）

> Get a specific LLM cost price by ID. Returns the price with overrides applied if any.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `priceId` | ULID | 是 |  |

**响应**：200 The request has succeeded., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Product Catalog


### GET `/openmeter/addons`

**含义**：列出附加组件
（原文：List add-ons）

> List all add-ons.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort add-ons returned in the response. Supported sort attributes are:  - `id` - `key` - `name` - `created_at` (default) - `updated_at`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `filter` | ListAddonsParamsFilter | 否 | Filter add-ons returned in the response. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/addons`

**含义**：创建附加组件
（原文：Create add-on）

> Create a new add-on.

**请求体**（`CreateAddonRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `key` |  | 是 | A key is a semi-unique string that is used to identify the add-on. It is used to reference the latest `active` version of the add-on and is unique with the version number. |
| `instance_type` |  | 是 | The InstanceType of the add-ons. Can be "single" or "multiple". |
| `currency` |  | 是 | The currency code of the add-on. |
| `rate_cards` | array<BillingRateCard> | 是 | The rate cards of the add-on. |

**响应**：201 Addon created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### PUT `/openmeter/addons/{addonId}`

**含义**：更新附加组件
（原文：Update add-on）

> Update an add-on by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | ULID | 是 |  |

**请求体**（`UpsertAddonRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `instance_type` |  | 是 | The InstanceType of the add-ons. Can be "single" or "multiple". |
| `rate_cards` | array<BillingRateCard> | 是 | The rate cards of the add-on. |

**响应**：200 Addon upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### GET `/openmeter/addons/{addonId}`

**含义**：查询附加组件
（原文：Get add-on）

> Get add-on by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | ULID | 是 |  |

**响应**：200 Addon response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### DELETE `/openmeter/addons/{addonId}`

**含义**：删除附加组件
（原文：Soft delete add-on）

> Soft delete add-on by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/addons/{addonId}/archive`

**含义**：归档附加组件
（原文：Archive add-on version）

> Archive an add-on version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | ULID | 是 |  |

**响应**：200 Addon updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/addons/{addonId}/publish`

**含义**：发布附加组件
（原文：Publish add-on version）

> Publish an add-on version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | ULID | 是 |  |

**响应**：200 Addon updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/plans`

**含义**：列出套餐
（原文：List plans）

> List all plans.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort plans returned in the response. Supported sort attributes are:  - `id` - `key` - `version` - `created_at` (default) - `updated_at` |
| `filter` | ListPlansParamsFilter | 否 | Filter plans returned in the response. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### POST `/openmeter/plans`

**含义**：创建套餐
（原文：Create plan）

> Create a new plan.

**请求体**（`CreatePlanRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `key` |  | 是 | A key is a semi-unique string that is used to identify the plan. It is used to reference the latest `active` version of the plan and is unique with the version number. |
| `currency` |  | 是 | The currency code of the plan. |
| `billing_cadence` |  | 是 | The billing cadence for subscriptions using this plan. |
| `pro_rating_enabled` | boolean | 否 | Whether pro-rating is enabled for this plan. |
| `phases` | array<BillingPlanPhase> | 是 | The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required. |

**响应**：201 Plan created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### PUT `/openmeter/plans/{planId}`

**含义**：更新套餐
（原文：Update plan）

> Update a plan by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |

**请求体**（`UpsertPlanRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `pro_rating_enabled` | boolean | 否 | Whether pro-rating is enabled for this plan. |
| `phases` | array<BillingPlanPhase> | 是 | The plan phases define the pricing ramp for a subscription. A phase switch occurs only at the end of a billing period. At least one phase is required. |

**响应**：200 Plan upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### GET `/openmeter/plans/{planId}`

**含义**：查询套餐
（原文：Get plan）

> Get a plan by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |

**响应**：200 Plan response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### DELETE `/openmeter/plans/{planId}`

**含义**：删除套餐
（原文：Delete plan）

> Delete a plan by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/plans/{planId}/addons`

**含义**：列出附加组件
（原文：List add-ons for plan）

> List add-ons associated with a plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/plans/{planId}/addons`

**含义**：创建附加组件
（原文：Add add-on to plan）

> Add an add-on to a plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |

**请求体**（`CreatePlanAddonRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `addon` |  | 是 | The add-on associated with the plan. |
| `from_plan_phase` |  | 是 | The key of the plan phase from which the add-on becomes available for purchase. |
| `max_quantity` | integer | 否 | The maximum number of times the add-on can be purchased for the plan. For single-instance add-ons this field must be omitted. For multi-instance add-ons when omitted, unlimited quantity can be purchased. |

**响应**：201 PlanAddon created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/plans/{planId}/addons/{planAddonId}`

**含义**：查询附加组件
（原文：Get add-on association for plan）

> Get an add-on association for a plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |
| `planAddonId` | ULID | 是 |  |

**响应**：200 PlanAddon response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/plans/{planId}/addons/{planAddonId}`

**含义**：更新附加组件
（原文：Update add-on association for plan）

> Update an add-on association for a plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |
| `planAddonId` | ULID | 是 |  |

**请求体**（`UpsertPlanAddonRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `from_plan_phase` |  | 是 | The key of the plan phase from which the add-on becomes available for purchase. |
| `max_quantity` | integer | 否 | The maximum number of times the add-on can be purchased for the plan. For single-instance add-ons this field must be omitted. For multi-instance add-ons when omitted, unlimited quantity can be purchased. |

**响应**：200 PlanAddon upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### DELETE `/openmeter/plans/{planId}/addons/{planAddonId}`

**含义**：删除附加组件
（原文：Remove add-on from plan）

> Remove an add-on from a plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |
| `planAddonId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/plans/{planId}/archive`

**含义**：归档套餐
（原文：Archive plan version）

> Archive a plan version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |

**响应**：200 Plan updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/plans/{planId}/publish`

**含义**：发布套餐
（原文：Publish plan version）

> Publish a plan version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | ULID | 是 |  |

**响应**：200 Plan updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


## OpenMeter Subscriptions


### POST `/openmeter/subscriptions`

**含义**：创建订阅
（原文：Create subscription）

**请求体**（`BillingSubscriptionCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `labels` | Labels | 否 |  |
| `settlement_mode` |  | 否 | Settlement mode for billing.  Values:  - `credit_then_invoice`: Credits are applied first, then any remainder is invoiced. - `credit_only`: Usage is settled exclusively against credits. |
| `customer` | object | 是 | The customer to create the subscription for. |
| `plan` | object | 是 | The plan reference of the subscription. |
| `billing_anchor` |  | 否 | A billing anchor is the fixed point in time that determines the subscription's recurring billing cycle. It affects when charges occur and how prorations are calculated. Common anchors:  - Calendar month (1st of each month): `2025-01-01T00:00:00Z` - Subscription anniversary (day customer signed up) - Custom date (customer-specified day)  If not provided, the subscription will be created with the subscription's creation time as the billing anchor. |

**响应**：201 Subscription created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


### GET `/openmeter/subscriptions`

**含义**：列出订阅
（原文：List subscriptions）

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort subscriptions returned in the response. Supported sort attributes are:  - `id` - `active_from` (default) - `active_to`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |
| `filter` | ListSubscriptionsParamsFilter | 否 | Filter subscriptions. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/subscriptions/{subscriptionId}`

**含义**：查询订阅
（原文：Get subscription）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | ULID | 是 |  |

**响应**：200 Subscription response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/subscriptions/{subscriptionId}/addons`

**含义**：创建附加组件
（原文：Create a new subscription add-on）

> Add add-on to a subscription.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | ULID | 是 |  |

**请求体**（`CreateSubscriptionAddonRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `labels` | Labels | 否 |  |
| `addon` |  | 是 | The add-on associated with the subscription. |
| `quantity` | integer | 是 | The quantity of the add-on. Always 1 for single instance add-ons. |
| `timing` |  | 是 | The timing of the operation. After the create or update, a new entry will be created in the timeline. |

**响应**：201 SubscriptionAddon created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


### GET `/openmeter/subscriptions/{subscriptionId}/addons`

**含义**：列出附加组件
（原文：List subscription addons）

> List the add-ons of a subscription.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | ULID | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `sort` | SortQuery | 否 | Sort subscription addons returned in the response. Supported sort attributes are:  - `id` - `created_at` (default) - `updated_at`  The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### GET `/openmeter/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}`

**含义**：查询附加组件
（原文：Get add-on association for subscription）

> Get an add-on association for a subscription.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | ULID | 是 |  |
| `subscriptionAddonId` | ULID | 是 |  |

**响应**：200 SubscriptionAddon response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### POST `/openmeter/subscriptions/{subscriptionId}/cancel`

**含义**：取消订阅
（原文：Cancel subscription）

> Cancels the subscription. Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancelation time.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | ULID | 是 |  |

**请求体**（`BillingSubscriptionCancel`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `timing` |  | 否 | If not provided the subscription is canceled immediately. |

**响应**：200 Subscription updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


### POST `/openmeter/subscriptions/{subscriptionId}/change`

**含义**：变更订阅
（原文：Change subscription）

> Closes a running subscription and starts a new one according to the specification. Can be used for upgrades, downgrades, and plan changes.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | ULID | 是 |  |

**请求体**（`BillingSubscriptionChange`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `labels` | Labels | 否 |  |
| `settlement_mode` |  | 否 | Settlement mode for billing.  Values:  - `credit_then_invoice`: Credits are applied first, then any remainder is invoiced. - `credit_only`: Usage is settled exclusively against credits. |
| `customer` | object | 是 | The customer to create the subscription for. |
| `plan` | object | 是 | The plan reference of the subscription. |
| `billing_anchor` |  | 否 | A billing anchor is the fixed point in time that determines the subscription's recurring billing cycle. It affects when charges occur and how prorations are calculated. Common anchors:  - Calendar month (1st of each month): `2025-01-01T00:00:00Z` - Subscription anniversary (day customer signed up) - Custom date (customer-specified day)  If not provided, the subscription will be created with the subscription's creation time as the billing anchor. |
| `timing` |  | 是 | Timing configuration for the change, when the change should take effect. For changing a subscription, the accepted values depend on the subscription configuration. |

**响应**：200 The request has succeeded., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


### POST `/openmeter/subscriptions/{subscriptionId}/unschedule-cancelation`

**含义**：取消计划的订阅注销
（原文：Unschedule subscription cancelation）

> Unschedules the subscription cancelation.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | ULID | 是 |  |

**响应**：200 Subscription updated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict


## OpenMeter Tax


### POST `/openmeter/tax-codes`

**含义**：创建税码
（原文：Create tax code）

**请求体**（`CreateTaxCodeRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `key` | ResourceKey | 是 |  |
| `app_mappings` | array<BillingTaxCodeAppMapping> | 是 | Mapping of app types to tax codes. |

**响应**：201 TaxCode created response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/tax-codes`

**含义**：列出税码
（原文：List tax codes）

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | object | 否 | Determines which page of the collection to retrieve. |
| `include_deleted` | boolean | 否 | Include deleted tax codes in the response. |

**响应**：200 Page paginated response., 400 Bad Request, 401 Unauthorized, 403 Forbidden


### GET `/openmeter/tax-codes/{taxCodeId}`

**含义**：查询税码
（原文：Get tax code）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `taxCodeId` | ULID | 是 |  |

**响应**：200 TaxCode response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


### PUT `/openmeter/tax-codes/{taxCodeId}`

**含义**：创建或更新税码
（原文：Upsert tax code）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `taxCodeId` | ULID | 是 |  |

**请求体**（`UpsertTaxCodeRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Display name of the resource.  Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource.  Maximum 1024 characters. |
| `labels` | Labels | 否 |  |
| `app_mappings` | array<BillingTaxCodeAppMapping> | 是 | Mapping of app types to tax codes. |

**响应**：200 TaxCode upsert response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 410 Gone


### DELETE `/openmeter/tax-codes/{taxCodeId}`

**含义**：删除税码
（原文：Delete tax code）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `taxCodeId` | ULID | 是 |  |

**响应**：204 Deleted response., 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found


---

# 第二部分：V1 API（经典）
> 注：`api/openapi.cloud.yaml`（云版本）的接口集合与 v1 完全相同，不再重复列出。

## App: Custom Invoicing


### POST `/api/v1/apps/custom-invoicing/{invoiceId}/draft/synchronized`

**含义**：提交自定义开票草稿同步结果
（原文：Submit draft synchronization results）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**请求体**（`CustomInvoicingDraftSynchronizedRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoicing` |  | 否 | The result of the synchronization. |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/apps/custom-invoicing/{invoiceId}/issuing/synchronized`

**含义**：提交自定义开票开具同步结果
（原文：Submit issuing synchronization results）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**请求体**（`CustomInvoicingFinalizedRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoicing` |  | 否 | The result of the synchronization. |
| `payment` |  | 否 | The result of the payment synchronization. |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/apps/custom-invoicing/{invoiceId}/payment/status`

**含义**：更新自定义开票支付状态
（原文：Update payment status）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**请求体**（`CustomInvoicingUpdatePaymentStatusRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `trigger` |  | 是 | The trigger to be executed on the invoice. |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## App: Stripe


### PUT `/api/v1/apps/{id}/stripe/api-key`

**含义**：更新 Stripe 应用的 API 密钥
（原文：Update Stripe API key）

> Update the Stripe API key.  ⚠️ __Deprecated__: Use [`PUT /api/v1/apps/{id}`](#tag/apps/put/api/v1/apps/{id}) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**请求体**（`StripeAPIKeyInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `secretAPIKey` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/apps/{id}/stripe/webhook`

**含义**：接收 Stripe Webhook 回调
（原文：Stripe webhook）

> Handle stripe webhooks for apps.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**请求体**（`StripeWebhookEvent`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 | The event ID. |
| `type` | string | 是 | The event type. |
| `livemode` | boolean | 是 | Live mode. |
| `created` | integer(int32) | 是 | The event created timestamp. |
| `data` | object | 是 | The event data. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/stripe/checkout/sessions`

**含义**：创建 Stripe 结账会话
（原文：Create checkout session）

> Create checkout session.

**请求体**（`CreateStripeCheckoutSessionRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `appId` | string | 否 | If not provided, the default Stripe app is used if any. |
| `customer` |  | 是 | Provide a customer ID or key to use an existing OpenMeter customer. or provide a customer object to create a new customer. |
| `stripeCustomerId` | string | 否 | Stripe customer ID. If not provided OpenMeter creates a new Stripe customer or uses the OpenMeter customer's default Stripe customer ID. |
| `options` |  | 是 | Options passed to Stripe when creating the checkout session. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Apps


### GET `/api/v1/apps`

**含义**：列出应用
（原文：List apps）

> List apps.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/apps/{id}`

**含义**：查询应用
（原文：Get app）

> Get the app.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/apps/{id}`

**含义**：更新应用
（原文：Update app）

> Update an app.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/apps/{id}`

**含义**：删除应用
（原文：Uninstall app）

> Uninstall an app.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/marketplace/listings`

**含义**：列出应用
（原文：List available apps）

> List available apps of the app marketplace.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/marketplace/listings/{type}`

**含义**：查询应用
（原文：Get app details by type）

> Get a marketplace listing by type.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `type` | AppType | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/marketplace/listings/{type}/install`

**含义**：安装应用
（原文：Install app）

> Install an app from the marketplace.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `type` | AppType | 是 | The type of the app to install. |

**请求体**（`MarketplaceInstallRequestPayload`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 否 | Name of the application to install.  If name is not provided defaults to the marketplace listing's name. |
| `createBillingProfile` | boolean | 否 | If true, a billing profile will be created for the app. The Stripe app will be also set as the default billing profile if the current default is a Sandbox app. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/marketplace/listings/{type}/install/apikey`

**含义**：安装apikey
（原文：Install app via API key）

> Install an marketplace app via API Key.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `type` | AppType | 是 | The type of the app to install. |

**请求体**

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 否 | Name of the application to install.  If name is not provided defaults to the marketplace listing's name. |
| `createBillingProfile` | boolean | 否 | If true, a billing profile will be created for the app. The Stripe app will be also set as the default billing profile if the current default is a Sandbox app. |
| `apiKey` | string | 是 | The API key for the provider. For example, the Stripe API key. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/marketplace/listings/{type}/install/oauth2`

**含义**：查询oauth2
（原文：Get OAuth2 install URL）

> Install an app via OAuth. Returns a URL to start the OAuth 2.0 flow.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `type` | AppType | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/marketplace/listings/{type}/install/oauth2/authorize`

**含义**：安装oauth2
（原文：Install app via OAuth2）

> Authorize OAuth2 code. Verifies the OAuth code and exchanges it for a token and refresh token

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `type` | AppType | 是 | The type of the app to install. |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `state` | string | 否 | Required if the "state" parameter was present in the client authorization request. The exact value received from the client:  Unique, randomly generated, opaque, and non-guessable string that is sent when starting an authentication request and validated when processing the response. |
| `code` | string | 否 | Authorization code which the client will later exchange for an access token. Required with the success response. |
| `error` | OAuth2AuthorizationCodeGrantErrorType | 否 | Error code. Required with the error response. |
| `error_description` | string | 否 | Optional human-readable text providing additional information, used to assist the client developer in understanding the error that occurred. |
| `error_uri` | string | 否 | Optional uri identifying a human-readable web page with information about the error, used to provide the client developer with additional information about the error |

**响应**：303 Redirection, 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Billing


### GET `/api/v1/billing/customers`

**含义**：列出客户
（原文：List customer overrides）

> List customer overrides using the specified filters.  The response will include the customer override values and the merged billing profile values.  If the includeAllCustomers is set to true, the list contains all customers. This mode is useful for getting the current effective billing workflow settings for all users regardless if they have customer orverrides or not.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `billingProfile` | array<string> | 否 | Filter by billing profile. |
| `customersWithoutPinnedProfile` | boolean | 否 | Only return customers without pinned billing profiles. This implicitly sets includeAllCustomers to true. |
| `includeAllCustomers` | boolean | 否 | Include customers without customer overrides.  If set to false only the customers specifically associated with a billing profile will be returned.  If set to true, in case of the default billing profile, all customers will be returned. |
| `customerId` | array<string> | 否 | Filter by customer id. |
| `customerName` | string | 否 | Filter by customer name. |
| `customerKey` | string | 否 | Filter by customer key |
| `customerPrimaryEmail` | string | 否 | Filter by customer primary email |
| `expand` | array<BillingProfileCustomerOverrideExpand> | 否 | Expand the response with additional details. |
| `order` |  | 否 | The order direction. |
| `orderBy` | BillingProfileCustomerOverrideOrderBy | 否 | The order by field. |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/billing/customers/{customerId}`

**含义**：创建客户
（原文：Create a new or update a customer override）

> The customer override can be used to pin a given customer to a billing profile different from the default one.  This can be used to test the effect of different billing profiles before making them the default ones or have different workflow settings for example for enterprise customers.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | string | 是 |  |

**请求体**（`BillingProfileCustomerOverrideCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `billingProfileId` | string | 否 | The billing profile this override is associated with.  If not provided, the default billing profile is chosen if available. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/billing/customers/{customerId}`

**含义**：查询客户
（原文：Get a customer override）

> Get a customer override by customer id.  The response will include the customer override values and the merged billing profile values.  If the customer override is not found, the default billing profile's values are returned. This behavior allows for getting a merged profile regardless of the customer override existence.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `expand` | array<BillingProfileCustomerOverrideExpand> | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/billing/customers/{customerId}`

**含义**：删除客户
（原文：Delete a customer override）

> Delete a customer override by customer id.  This will remove the customer override and the customer will be subject to the default billing profile's settings again.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/customers/{customerId}/invoices/pending-lines`

**含义**：创建发票
（原文：Create pending line items）

> Create a new pending line item (charge).  This call is used to create a new pending line item for the customer if required a new gathering invoice will be created.  A new invoice will be created if: - there is no invoice in gathering state - the currency of the line item doesn't match the currency of any invoices in gathering state

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | string | 是 |  |

**请求体**（`InvoicePendingLineCreateInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `currency` |  | 是 | The currency of the lines to be created. |
| `lines` | array<InvoicePendingLineCreate> | 是 | The lines to be created. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/customers/{customerId}/invoices/simulate`

**含义**：创建发票
（原文：Simulate an invoice for a customer）

> Simulate an invoice for a customer.  This call will simulate an invoice for a customer based on the pending line items.  The call will return the total amount of the invoice and the line items that will be included in the invoice.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerId` | string | 是 |  |

**请求体**（`InvoiceSimulationInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `number` |  | 否 | The number of the invoice. |
| `currency` |  | 是 | Currency for all invoice line items.  Multi currency invoices are not supported yet. |
| `lines` | array<InvoiceSimulationLine> | 是 | Lines to be included in the generated invoice. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/billing/invoices`

**含义**：列出发票
（原文：List invoices）

> List invoices based on the specified filters.  The expand option can be used to include additional information (besides the invoice header and totals) in the response. For example by adding the expand=lines option the invoice lines will be included in the response.  Gathering invoices will always show the current usage calculated on the fly.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `statuses` | array<InvoiceStatus> | 否 | Filter by the invoice status. |
| `extendedStatuses` | array<string> | 否 | Filter by invoice extended statuses |
| `issuedAfter` | string(date-time) | 否 | Filter by invoice issued time. Inclusive. |
| `issuedBefore` | string(date-time) | 否 | Filter by invoice issued time. Inclusive. |
| `periodStartAfter` | string(date-time) | 否 | Filter by period start time. Inclusive. |
| `periodStartBefore` | string(date-time) | 否 | Filter by period start time. Inclusive. |
| `createdAfter` | string(date-time) | 否 | Filter by invoice created time. Inclusive. |
| `createdBefore` | string(date-time) | 否 | Filter by invoice created time. Inclusive. |
| `expand` | array<InvoiceExpand> | 否 | What parts of the list output to expand in listings |
| `customers` | array<string> | 否 | Filter by customer ID |
| `includeDeleted` | boolean | 否 | Include deleted invoices |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | InvoiceOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/invoices/invoice`

**含义**：根据待计费明细为客户开票
（原文：Invoice a customer based on the pending line items）

> Create a new invoice from the pending line items.  This should be only called if for some reason we need to invoice a customer outside of the normal billing cycle.  When creating an invoice, the pending line items will be marked as invoiced and the invoice will be created with the total amount of the pending items.  New pending line items will be created for the period between now() and the next billing cycle's begining date for any metered item.  The call can return multiple invoices if the pending line items are in different currencies.

**请求体**（`InvoicePendingLinesActionInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `filters` |  | 否 | Filters to apply when creating the invoice. |
| `asOf` | string(date-time) | 否 | The time as of which the invoice is created.  If not provided, the current time is used. |
| `customerId` | string | 是 | The customer ID for which to create the invoice. |
| `progressiveBillingOverride` | boolean | 否 | Override the progressive billing setting of the customer.  Can be used to disable/enable progressive billing in case the business logic requires it, if not provided the billing profile's progressive billing setting will be used. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/billing/invoices/{invoiceId}`

**含义**：查询发票
（原文：Get an invoice）

> Get an invoice by ID.  Gathering invoices will always show the current usage calculated on the fly.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `expand` | array<InvoiceExpand> | 否 |  |
| `includeDeletedLines` | boolean | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/billing/invoices/{invoiceId}`

**含义**：删除发票
（原文：Delete an invoice）

> Delete an invoice  Only invoices that are in the draft (or earlier) status can be deleted.  Invoices that are post finalization can only be voided.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/billing/invoices/{invoiceId}`

**含义**：更新发票
（原文：Update an invoice）

> Update an invoice  Only invoices in draft or earlier status can be updated.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**请求体**（`InvoiceReplaceUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `supplier` |  | 是 | The supplier of the lines included in the invoice. |
| `customer` |  | 是 | The customer the invoice is sent to. |
| `lines` | array<InvoiceLineReplaceUpdate> | 是 | The lines included in the invoice. |
| `workflow` |  | 是 | The workflow settings for the invoice. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/invoices/{invoiceId}/advance`

**含义**：推进发票到下一状态
（原文：Advance the invoice's state to the next status）

> Advance the invoice's state to the next status.  The call doesn't "approve the invoice", it only advances the invoice to the next status if the transition would be automatic.  The action can be called when the invoice's statusDetails' actions field contain the "advance" action.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/invoices/{invoiceId}/approve`

**含义**：发送发票
（原文：Send the invoice to the customer）

> Approve an invoice and start executing the payment workflow.  This call instantly sends the invoice to the customer using the configured billing profile app.  This call is valid in two invoice statuses: - `draft`: the invoice will be sent to the customer, the invluce state becomes issued - `manual_approval_needed`: the invoice will be sent to the customer, the invoice state becomes issued

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/invoices/{invoiceId}/retry`

**含义**：重试推进发票状态
（原文：Retry advancing the invoice after a failed attempt.）

> Retry advancing the invoice after a failed attempt.  The action can be called when the invoice's statusDetails' actions field contain the "retry" action.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/invoices/{invoiceId}/snapshot-quantities`

**含义**：对发票按量明细进行快照
（原文：Snapshot quantities for usage based line items）

> Snapshot quantities for usage based line items.  This call will snapshot the quantities for all usage based line items in the invoice.  This call is only valid in `draft.waiting_for_collection` status, where the collection period can be skipped using this action.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/invoices/{invoiceId}/taxes/recalculate`

**含义**：重新计算税额
（原文：Recalculate an invoice's tax amounts）

> Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)  Note: charges might apply, depending on the tax provider.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/invoices/{invoiceId}/void`

**含义**：作废发票
（原文：Void an invoice）

> Void an invoice  Only invoices that have been alread issued can be voided.  Voiding an invoice will mark it as voided, the user can specify how to handle the voided line items.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `invoiceId` | string | 是 |  |

**请求体**（`VoidInvoiceActionInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `action` |  | 是 | The action to take on the voided line items. |
| `reason` | string | 是 | The reason for voiding the invoice. |
| `overrides` | array<VoidInvoiceActionLineOverride> | 否 | Per line item overrides for the action.  If not specified, the `action` will be applied to all line items. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/billing/profiles`

**含义**：列出账单档案
（原文：List billing profiles）

> List all billing profiles matching the specified filters.  The expand option can be used to include additional information (besides the billing profile) in the response. For example by adding the expand=apps option the apps used by the billing profile will be included in the response.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeArchived` | boolean | 否 |  |
| `expand` | array<BillingProfileExpand> | 否 |  |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | BillingProfileOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/billing/profiles`

**含义**：创建账单档案
（原文：Create a new billing profile）

> Create a new billing profile  Billing profiles are representations of a customer's billing information. Customer overrides can be applied to a billing profile to customize the billing behavior for a specific customer.

**请求体**（`BillingProfileCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `supplier` |  | 是 | The name and contact information for the supplier this billing profile represents |
| `default` | boolean | 是 | Is this the default profile? |
| `workflow` |  | 是 | The billing workflow settings for this profile. |
| `apps` |  | 是 | The apps used by this billing profile. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/billing/profiles/{id}`

**含义**：删除账单档案
（原文：Delete a billing profile）

> Delete a billing profile by id.  Only such billing profiles can be deleted that are: - not the default one - not pinned to any customer using customer overrides - only have finalized invoices

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/billing/profiles/{id}`

**含义**：查询账单档案
（原文：Get a billing profile）

> Get a billing profile by id.  The expand option can be used to include additional information (besides the billing profile) in the response. For example by adding the expand=apps option the apps used by the billing profile will be included in the response.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `expand` | array<BillingProfileExpand> | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/billing/profiles/{id}`

**含义**：更新账单档案
（原文：Update a billing profile）

> Update a billing profile by id.  The apps field cannot be updated directly, if an app change is desired a new profile should be created.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**请求体**（`BillingProfileReplaceUpdateWithWorkflow`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `supplier` |  | 是 | The name and contact information for the supplier this billing profile represents |
| `default` | boolean | 是 | Is this the default profile? |
| `workflow` |  | 是 | The billing workflow settings for this profile. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Customers


### POST `/api/v1/customers`

**含义**：创建客户
（原文：Create customer）

> Create a new customer.

**请求体**（`CustomerCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `key` | string | 否 | An optional unique key of the customer. Either key or usageAttribution.subjectKeys must be provided. Useful to reference the customer in external systems. For example, your database ID. |
| `usageAttribution` |  | 否 | Mapping to attribute metered usage to the customer Either key or usageAttribution.subjectKeys must be provided. |
| `primaryEmail` | string | 否 | The primary email address of the customer. |
| `currency` |  | 否 | Currency of the customer. Used for billing, tax and invoicing. |
| `billingAddress` |  | 否 | The billing address of the customer. Used for tax and invoicing. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/customers`

**含义**：列出客户
（原文：List customers）

> List customers.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | CustomerOrderBy | 否 | The order by field. |
| `includeDeleted` | boolean | 否 | Include deleted customers. |
| `key` | string | 否 | Filter customers by key. Case-insensitive partial match. |
| `name` | string | 否 | Filter customers by name. Case-insensitive partial match. |
| `primaryEmail` | string | 否 | Filter customers by primary email. Case-insensitive partial match. |
| `subject` | string | 否 | Filter customers by usage attribution subject. Case-insensitive partial match. |
| `planKey` | string | 否 | Filter customers by the plan key of their susbcription. |
| `expand` | array<CustomerExpand> | 否 | What parts of the list output to expand in listings |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/customers/{customerIdOrKey}`

**含义**：查询客户
（原文：Get customer）

> Get a customer by ID or key.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `expand` | array<CustomerExpand> | 否 | What parts of the customer output to expand |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/customers/{customerIdOrKey}`

**含义**：更新客户
（原文：Update customer）

> Update a customer by ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**请求体**（`CustomerReplaceUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `key` | string | 否 | An optional unique key of the customer. Either key or usageAttribution.subjectKeys must be provided. Useful to reference the customer in external systems. For example, your database ID. |
| `usageAttribution` |  | 否 | Mapping to attribute metered usage to the customer Either key or usageAttribution.subjectKeys must be provided. |
| `primaryEmail` | string | 否 | The primary email address of the customer. |
| `currency` |  | 否 | Currency of the customer. Used for billing, tax and invoicing. |
| `billingAddress` |  | 否 | The billing address of the customer. Used for tax and invoicing. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/customers/{customerIdOrKey}`

**含义**：删除客户
（原文：Delete customer）

> Delete a customer by ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/customers/{customerIdOrKey}/apps`

**含义**：列出应用
（原文：List customer app data）

> List customers app data.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `type` | AppType | 否 | Filter customer data by app type. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/customers/{customerIdOrKey}/apps`

**含义**：创建或更新应用
（原文：Upsert customer app data）

> Upsert customer app data.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/customers/{customerIdOrKey}/apps/{appId}`

**含义**：删除应用
（原文：Delete customer app data）

> Delete customer app data.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `appId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/customers/{customerIdOrKey}/stripe`

**含义**：查询Stripe
（原文：Get customer stripe app data）

> Get stripe app data for a customer. Only returns data if the customer billing profile is linked to a stripe app.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/customers/{customerIdOrKey}/stripe`

**含义**：创建或更新Stripe
（原文：Upsert customer stripe app data）

> Upsert stripe app data for a customer. Only updates data if the customer billing profile is linked to a stripe app.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**请求体**（`StripeCustomerAppDataBase`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `stripeCustomerId` | string | 是 | The Stripe customer ID. |
| `stripeDefaultPaymentMethodId` | string | 否 | The Stripe default payment method ID. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/customers/{customerIdOrKey}/stripe/portal`

**含义**：创建消费门户
（原文：Create Stripe customer portal session）

> Create Stripe customer portal session. Only returns URL if the customer billing profile is linked to a stripe app and customer.  Useful to redirect the customer to the Stripe customer portal to manage their payment methods, change their billing address and access their invoice history.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**请求体**（`CreateStripeCustomerPortalSessionParams`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `configurationId` | string | 否 | The ID of an existing configuration to use for this session, describing its functionality and features. If not specified, the session uses the default configuration.  See https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-configuration |
| `locale` | string | 否 | The IETF language tag of the locale customer portal is displayed in. If blank or auto, the customer’s preferred_locales or browser’s locale is used.  See: https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-locale |
| `returnUrl` | string | 否 | The URL to redirect the customer to after they have completed their requested actions.  See: https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-return_url |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/customers/{customerIdOrKey}/subscriptions`

**含义**：列出订阅
（原文：List customer subscriptions）

> Lists all subscriptions for a customer.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `status` | array<SubscriptionStatus> | 否 |  |
| `order` |  | 否 | The order direction. |
| `orderBy` | CustomerSubscriptionOrderBy | 否 | The order by field. |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Debug


### GET `/api/v1/debug/metrics`

**含义**：查询指标
（原文：Get event metrics）

> Returns debug metrics (in OpenMetrics format) like the number of ingested events since mindnight UTC.  The OpenMetrics Counter(s) reset every day at midnight UTC.

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Entitlements


### GET `/api/v1/customers/{customerIdOrKey}/access`

**含义**：查询客户的功能访问权限
（原文：Get customer access）

> Get the overall access of a customer.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/value`

**含义**：查询客户某功能特性的权益值
（原文：Get customer entitlement value）

> Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `featureKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `time` | string(date-time) | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/entitlements`

**含义**：列出权益
（原文：List all entitlements）

> List all entitlements for all the subjects and features. This endpoint is intended for administrative purposes only. To fetch the entitlements of a specific subject please use the /api/v1/subjects/{subjectKeyOrID}/entitlements endpoint. If page is provided that takes precedence and the paginated response is returned.  ⚠️ __Deprecated__: Use [`GET /api/v2/entitlements`](#tag/entitlements/get/api/v2/entitlements) instead.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `feature` | array<string> | 否 | Filtering by multiple features.  Usage: `?feature=feature-1&feature=feature-2` |
| `subject` | array<string> | 否 | Filtering by multiple subjects.  Usage: `?subject=customer-1&subject=customer-2` |
| `entitlementType` | array<EntitlementType> | 否 | Filtering by multiple entitlement types.  Usage: `?entitlementType=metered&entitlementType=boolean` |
| `excludeInactive` | boolean | 否 | Exclude inactive entitlements in the response (those scheduled for later or earlier) |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `offset` | integer | 否 | Number of items to skip.  Default is 0. |
| `limit` | integer | 否 | Number of items to return.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | EntitlementOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/entitlements/{entitlementId}`

**含义**：查询权益
（原文：Get entitlement by ID）

> Get entitlement by ID.  ⚠️ __Deprecated__: Use [`GET /api/v2/entitlements/{entitlementId}`](#tag/entitlements/get/api/v2/entitlements/{entitlementId}) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `entitlementId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/grants`

**含义**：列出授予
（原文：List grants）

> List all grants for all the subjects and entitlements. This endpoint is intended for administrative purposes only. To fetch the grants of a specific entitlement please use the /api/v1/subjects/{subjectKeyOrID}/entitlements/{entitlementOrFeatureID}/grants endpoint. If page is provided that takes precedence and the paginated response is returned.  ⚠️ __Deprecated__: Use [`GET /api/v2/grants`](#tag/entitlements/get/api/v2/grants) instead.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `feature` | array<string> | 否 | Filtering by multiple features.  Usage: `?feature=feature-1&feature=feature-2` |
| `subject` | array<string> | 否 | Filtering by multiple subjects.  Usage: `?subject=customer-1&subject=customer-2` |
| `includeDeleted` | boolean | 否 | Include deleted |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `offset` | integer | 否 | Number of items to skip.  Default is 0. |
| `limit` | integer | 否 | Number of items to return.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | GrantOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/grants/{grantId}`

**含义**：作废授予
（原文：Void grant）

> Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations. Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed to the grant will remain, but future usage cannot be burnt down from the grant. For example, if you have a single grant for your metered entitlement with an initial amount of 100, and so far 60 usage has been metered, the grant (and the entitlement itself) would have a balance of 40. If you then void that grant, balance becomes 0, but the 60 previous usage will not be affected.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `grantId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `at` | string(date-time) | 否 | The time at which the grant should be voided. Must not be in the future and must be within the current usage period of the entitlement. Defaults to the current time if not specified. |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subjects/{subjectIdOrKey}/entitlements`

**含义**：为主体创建权益
（原文：Create a subject entitlement）

> OpenMeter has three types of entitlements: metered, boolean, and static. The type property determines the type of entitlement. The underlying feature has to be compatible with the entitlement type specified in the request (e.g., a metered entitlement needs a feature associated with a meter).  - Boolean entitlements define static feature access, e.g. "Can use SSO authentication". - Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config). - Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.  A given subject can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.  Once an entitlement is created you cannot modify it, only delete it.  ⚠️ __Deprecated__: Use [`POST /api/v2/customers/{customerIdOrKey}/entitlements`](#tag/entitlements/post/api/v2/customers/{customerIdOrKey}/entitlements) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subjects/{subjectIdOrKey}/entitlements`

**含义**：列出权益
（原文：List subject entitlements）

> List all entitlements for a subject. For checking entitlement access, use the /value endpoint instead.  ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`

**含义**：列出授予
（原文：List subject entitlement grants）

> List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.  ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 |  |
| `orderBy` |  | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`

**含义**：创建授予
（原文：Create subject entitlement grant）

> Grants define a behavior of granting usage for a metered entitlement. They can have complicated recurrence and rollover rules, thanks to which you can define a wide range of access patterns with a single grant, in most cases you don't have to periodically create new grants. You can only issue grants for active metered entitlements.  A grant defines a given amount of usage that can be consumed for the entitlement. The grant is in effect between its effective date and its expiration date. Specifying both is mandatory for new grants.  Grants have a priority setting that determines their order of use. Lower numbers have higher priority, with 0 being the highest priority.  Grants can have a recurrence setting intended to automate the manual reissuing of grants. For example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover settings).  Rollover settings define what happens to the remaining balance of a grant at a reset. Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))  Grants cannot be changed once created, only deleted. This is to ensure that balance is deterministic regardless of when it is queried.  ⚠️ __Deprecated__: Use [`POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`](#tag/entitlements/post/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**请求体**（`EntitlementGrantCreateInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `amount` | number(double) | 是 | The amount to grant. Should be a positive number. |
| `priority` | integer(uint8) | 否 | The priority of the grant. Grants with higher priority are applied first. Priority is a positive decimal numbers. With lower numbers indicating higher importance. For example, a priority of 1 is more urgent than a priority of 2. When there are several grants available for the same subject, the system selects the grant with the highest priority. In cases where grants share the same priority level, the grant closest to its expiration will be used first. In the case of two grants have identical priorities and expiration dates, the system will use the grant that was created first. |
| `effectiveAt` | string(date-time) | 是 | Effective date for grants and anchor for recurring grants. Provided value will be ceiled to metering windowSize (minute). |
| `expiration` |  | 是 | The grant expiration definition |
| `maxRolloverAmount` | number(double) | 否 | Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset. Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount)) |
| `minRolloverAmount` | number(double) | 否 | Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset. Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount)) |
| `metadata` |  | 否 | The grant metadata. |
| `recurrence` |  | 否 | The subject of the grant. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override`

**含义**：覆盖主体的权益设置
（原文：Override subject entitlement）

> Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided subject-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.  This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.  ⚠️ __Deprecated__: Use [`PUT /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override`](#tag/entitlements/put/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value`

**含义**：查询当前值
（原文：Get subject entitlement value）

> This endpoint should be used for access checks and enforcement. All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.  For convenience reasons, /value works with both entitlementId and featureKey.  ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `time` | string(date-time) | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}`

**含义**：查询权益
（原文：Get subject entitlement）

> Get entitlement by id. For checking entitlement access, use the /value endpoint instead.  ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}`

**含义**：删除权益
（原文：Delete subject entitlement）

> Deleting an entitlement revokes access to the associated feature. As a single subject can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well. As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.  ⚠️ __Deprecated__: Use [`DELETE /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}`](#tag/entitlements/delete/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/history`

**含义**：查询历史记录
（原文：Get subject entitlement history）

> Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.  BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.  WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.  ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `from` | string(date-time) | 否 | Start of time range to query entitlement: date-time in RFC 3339 format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter. |
| `to` | string(date-time) | 否 | End of time range to query entitlement: date-time in RFC 3339 format. Defaults to now. If not now then gets truncated to the granularity of the underlying meter. |
| `windowSize` | WindowSize | 是 | Windowsize |
| `windowTimeZone` | string | 否 | The timezone used when calculating the windows. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/reset`

**含义**：重置权益
（原文：Reset subject entitlement）

> Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the subjects billing period to enforce usage based on their subscription.  Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.  ⚠️ __Deprecated__: Use [`POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset`](#tag/entitlements/post/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset) instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |
| `entitlementId` | string | 是 |  |

**请求体**（`ResetEntitlementUsageInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `effectiveAt` | string(date-time) | 否 | The time at which the reset takes effect, defaults to now. The reset cannot be in the future. The provided value is truncated to the minute due to how historical meter data is stored. |
| `retainAnchor` | boolean | 否 | Determines whether the usage period anchor is retained or reset to the effectiveAt time. - If true, the usage period anchor is retained. - If false, the usage period anchor is reset to the effectiveAt time. |
| `preserveOverage` | boolean | 否 | Determines whether the overage is preserved or forgiven, overriding the entitlement's default behavior. - If true, the overage is preserved. - If false, the overage is forgiven. |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v2/customers/{customerIdOrKey}/entitlements`

**含义**：为客户创建权益
（原文：Create a customer entitlement）

> OpenMeter has three types of entitlements: metered, boolean, and static. The type property determines the type of entitlement. The underlying feature has to be compatible with the entitlement type specified in the request (e.g., a metered entitlement needs a feature associated with a meter).  - Boolean entitlements define static feature access, e.g. "Can use SSO authentication". - Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config). - Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.  A given customer can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.  Once an entitlement is created you cannot modify it, only delete it.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/customers/{customerIdOrKey}/entitlements`

**含义**：列出权益
（原文：List customer entitlements）

> List all entitlements for a customer. For checking entitlement access, use the /value endpoint instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 |  |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | EntitlementOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}`

**含义**：查询权益
（原文：Get customer entitlement）

> Get entitlement by feature key. For checking entitlement access, use the /value endpoint instead. If featureKey is used, the entitlement is resolved for the current timestamp.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}`

**含义**：删除权益
（原文：Delete customer entitlement）

> Deleting an entitlement revokes access to the associated feature. As a single customer can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well. As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`

**含义**：列出授予
（原文：List customer entitlement grants）

> List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 |  |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `offset` | integer | 否 | Number of items to skip.  Default is 0. |
| `limit` | integer | 否 | Number of items to return.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | GrantOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`

**含义**：创建授予
（原文：Create customer entitlement grant）

> Grants define a behavior of granting usage for a metered entitlement. They can have complicated recurrence and rollover rules, thanks to which you can define a wide range of access patterns with a single grant, in most cases you don't have to periodically create new grants. You can only issue grants for active metered entitlements.  A grant defines a given amount of usage that can be consumed for the entitlement. The grant is in effect between its effective date and its expiration date. Specifying both is mandatory for new grants.  Grants have a priority setting that determines their order of use. Lower numbers have higher priority, with 0 being the highest priority.  Grants can have a recurrence setting intended to automate the manual reissuing of grants. For example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover settings).  Rollover settings define what happens to the remaining balance of a grant at a reset. Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))  Grants cannot be changed once created, only deleted. This is to ensure that balance is deterministic regardless of when it is queried.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**请求体**（`EntitlementGrantCreateInputV2`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `amount` | number(double) | 是 | The amount to grant. Should be a positive number. |
| `priority` | integer(uint8) | 否 | The priority of the grant. Grants with higher priority are applied first. Priority is a positive decimal numbers. With lower numbers indicating higher importance. For example, a priority of 1 is more urgent than a priority of 2. When there are several grants available for the same subject, the system selects the grant with the highest priority. In cases where grants share the same priority level, the grant closest to its expiration will be used first. In the case of two grants have identical priorities and expiration dates, the system will use the grant that was created first. |
| `effectiveAt` | string(date-time) | 是 | Effective date for grants and anchor for recurring grants. Provided value will be ceiled to metering windowSize (minute). |
| `minRolloverAmount` | number(double) | 否 | Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset. Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount)) |
| `metadata` |  | 否 | The grant metadata. |
| `recurrence` |  | 否 | The subject of the grant. |
| `maxRolloverAmount` | number(double) | 否 | Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset. The default value equals grant amount. Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount)) |
| `expiration` |  | 否 | The grant expiration definition. If no expiration is provided, the grant can be active indefinitely. |
| `annotations` |  | 否 | Grant annotations |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history`

**含义**：查询历史记录
（原文：Get customer entitlement history）

> Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.  BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.  WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `from` | string(date-time) | 否 | Start of time range to query entitlement: date-time in RFC 3339 format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter. |
| `to` | string(date-time) | 否 | End of time range to query entitlement: date-time in RFC 3339 format. Defaults to now. If not now then gets truncated to the granularity of the underlying meter. |
| `windowSize` | WindowSize | 是 | Windowsize |
| `windowTimeZone` | string | 否 | The timezone used when calculating the windows. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override`

**含义**：覆盖客户的权益设置
（原文：Override customer entitlement）

> Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided customer-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.  This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | ULIDOrExternalKey | 是 |  |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset`

**含义**：重置权益
（原文：Reset customer entitlement）

> Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the customers billing period to enforce usage based on their subscription.  Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**请求体**（`ResetEntitlementUsageInput`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `effectiveAt` | string(date-time) | 否 | The time at which the reset takes effect, defaults to now. The reset cannot be in the future. The provided value is truncated to the minute due to how historical meter data is stored. |
| `retainAnchor` | boolean | 否 | Determines whether the usage period anchor is retained or reset to the effectiveAt time. - If true, the usage period anchor is retained. - If false, the usage period anchor is reset to the effectiveAt time. |
| `preserveOverage` | boolean | 否 | Determines whether the overage is preserved or forgiven, overriding the entitlement's default behavior. - If true, the overage is preserved. - If false, the overage is forgiven. |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value`

**含义**：查询客户某权益当前值
（原文：Get customer entitlement value）

> Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customerIdOrKey` | ULIDOrExternalKey | 是 |  |
| `entitlementIdOrFeatureKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `time` | string(date-time) | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/entitlements`

**含义**：列出权益
（原文：List all entitlements）

> List all entitlements for all the customers and features. This endpoint is intended for administrative purposes only. To fetch the entitlements of a specific subject please use the /api/v2/customers/{customerIdOrKey}/entitlements endpoint.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `feature` | array<string> | 否 | Filtering by multiple features.  Usage: `?feature=feature-1&feature=feature-2` |
| `customerKeys` | array<string> | 否 | Filtering by multiple customers.  Usage: `?customerKeys=customer-1&customerKeys=customer-3` |
| `customerIds` | array<string> | 否 | Filtering by multiple customers.  Usage: `?customerIds=01K4WAQ0J99ZZ0MD75HXR112H8&customerIds=01K4WAQ0J99ZZ0MD75HXR112H9` |
| `entitlementType` | array<EntitlementType> | 否 | Filtering by multiple entitlement types.  Usage: `?entitlementType=metered&entitlementType=boolean` |
| `excludeInactive` | boolean | 否 | Exclude inactive entitlements in the response (those scheduled for later or earlier) |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `offset` | integer | 否 | Number of items to skip.  Default is 0. |
| `limit` | integer | 否 | Number of items to return.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | EntitlementOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/entitlements/{entitlementId}`

**含义**：查询权益
（原文：Get entitlement by ID）

> Get entitlement by ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `entitlementId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/grants`

**含义**：列出授予
（原文：List grants）

> List all grants for all the customers and entitlements. This endpoint is intended for administrative purposes only. To fetch the grants of a specific entitlement please use the /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants endpoint. If page is provided that takes precedence and the paginated response is returned.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `feature` | array<string> | 否 | Filtering by multiple features.  Usage: `?feature=feature-1&feature=feature-2` |
| `customer` | array<ULIDOrExternalKey> | 否 | Filtering by multiple customers (either by ID or key).  Usage: `?customer=customer-1&customer=customer-2` |
| `includeDeleted` | boolean | 否 | Include deleted |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `offset` | integer | 否 | Number of items to skip.  Default is 0. |
| `limit` | integer | 否 | Number of items to return.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | GrantOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Events


### GET `/api/v1/events`

**含义**：列出事件
（原文：List ingested events）

> List ingested events within a time range.  If the from query param is not provided it defaults to last 72 hours.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `clientId` | string | 否 | Client ID Useful to track progress of a query. |
| `ingestedAtFrom` | string(date-time) | 否 | Start date-time in RFC 3339 format.  Inclusive. |
| `ingestedAtTo` | string(date-time) | 否 | End date-time in RFC 3339 format.  Inclusive. |
| `id` | string | 否 | The event ID.  Accepts partial ID. |
| `subject` | string | 否 | The event subject.  Accepts partial subject. |
| `customerId` | array<string> | 否 | The event customer ID. |
| `from` | string(date-time) | 否 | Start date-time in RFC 3339 format.  Inclusive. |
| `to` | string(date-time) | 否 | End date-time in RFC 3339 format.  Inclusive. |
| `limit` | integer | 否 | Number of events to return. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/events`

**含义**：创建事件
（原文：Ingest events）

> Ingests an event or batch of events following the CloudEvents specification.

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v2/events`

**含义**：列出事件
（原文：List ingested events）

> List ingested events with advanced filtering and cursor pagination.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `cursor` | string | 否 | The cursor after which to start the pagination. |
| `limit` | integer | 否 | The limit of the pagination. |
| `clientId` | string | 否 | Client ID Useful to track progress of a query. |
| `filter` |  | 否 | The filter for the events encoded as JSON string. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Lookup Information


### GET `/api/v1/info/currencies`

**含义**：列出货币
（原文：List supported currencies）

> List all supported currencies.

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/info/progress/{id}`

**含义**：查询进度
（原文：Get progress）

> Get progress

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Meters


### GET `/api/v1/meters`

**含义**：列出计量器
（原文：List meters）

> List meters.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | MeterOrderBy | 否 | The order by field. |
| `includeDeleted` | boolean | 否 | Include deleted meters. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/meters`

**含义**：创建计量器
（原文：Create meter）

> Create a meter.

**请求体**（`MeterCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `name` | string | 否 | Human-readable name for the resource. Between 1 and 256 characters. Defaults to the slug if not specified. |
| `slug` | string | 是 | A unique, human-readable identifier for the meter. Must consist only alphanumeric and underscore characters. |
| `aggregation` |  | 是 | The aggregation type to use for the meter. |
| `eventType` | string | 是 | The event type to aggregate. |
| `eventFrom` | string(date-time) | 否 | The date since the meter should include events. Useful to skip old events. If not specified, all historical events are included. |
| `valueProperty` | string | 否 | JSONPath expression to extract the value from the ingested event's data property.  The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be parsed to a number.  For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the valueProperty is ignored. |
| `groupBy` | object | 否 | Named JSONPath expressions to extract the group by values from the event data.  Keys must be unique and consist only alphanumeric and underscore characters. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/meters/{meterIdOrSlug}`

**含义**：查询计量器
（原文：Get meter）

> Get a meter by ID or slug.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterIdOrSlug` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/meters/{meterIdOrSlug}`

**含义**：更新计量器
（原文：Update meter）

> Update a meter.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterIdOrSlug` | string | 是 |  |

**请求体**（`MeterUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `name` | string | 否 | Human-readable name for the resource. Between 1 and 256 characters. Defaults to the slug if not specified. |
| `groupBy` | object | 否 | Named JSONPath expressions to extract the group by values from the event data.  Keys must be unique and consist only alphanumeric and underscore characters. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/meters/{meterIdOrSlug}`

**含义**：删除计量器
（原文：Delete meter）

> Delete a meter.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterIdOrSlug` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/meters/{meterIdOrSlug}/group-by/{groupByKey}/values`

**含义**：列出分组
（原文：List meter group by values）

> List meter group by values.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterIdOrSlug` | string | 是 |  |
| `groupByKey` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `from` | string(date-time) | 否 | Start date-time in RFC 3339 format.  Inclusive. Defaults to 24 hours ago.  For example: ?from=2025-01-01T00%3A00%3A00.000Z |
| `to` | string(date-time) | 否 | End date-time in RFC 3339 format.  Inclusive.  For example: ?to=2025-02-01T00%3A00%3A00.000Z |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/meters/{meterIdOrSlug}/query`

**含义**：查询计量器
（原文：Query meter）

> Query meter for usage.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterIdOrSlug` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `clientId` | string | 否 | Client ID Useful to track progress of a query. |
| `from` | string(date-time) | 否 | Start date-time in RFC 3339 format.  Inclusive.  For example: ?from=2025-01-01T00%3A00%3A00.000Z |
| `to` | string(date-time) | 否 | End date-time in RFC 3339 format.  Inclusive.  For example: ?to=2025-02-01T00%3A00%3A00.000Z |
| `windowSize` | WindowSize | 否 | If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.  For example: ?windowSize=DAY |
| `windowTimeZone` | string | 否 | The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). If not specified, the UTC timezone will be used.  For example: ?windowTimeZone=UTC |
| `subject` | array<string> | 否 | Filtering by multiple subjects.  For example: ?subject=subject-1&subject=subject-2 |
| `filterCustomerId` | array<string> | 否 | Filtering by multiple customers.  For example: ?filterCustomerId=customer-1&filterCustomerId=customer-2 |
| `filterGroupBy` | object | 否 | Simple filter for group bys with exact match.  For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo  ⚠️ __Deprecated__: Use `advancedMeterGroupByFilters` instead |
| `advancedMeterGroupByFilters` |  | 否 | Optional advanced meter group by filters. You can use this to filter for values of the meter groupBy fields. |
| `groupBy` | array<string> | 否 | If not specified a single aggregate will be returned for each subject and time window. `subject` is a reserved group by value.  For example: ?groupBy=subject&groupBy=model |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/meters/{meterIdOrSlug}/query`

**含义**：查询计量器
（原文：Query meter）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterIdOrSlug` | string | 是 |  |

**请求体**（`MeterQueryRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `clientId` | string | 否 | Client ID Useful to track progress of a query. |
| `from` | string(date-time) | 否 | Start date-time in RFC 3339 format.  Inclusive. |
| `to` | string(date-time) | 否 | End date-time in RFC 3339 format.  Inclusive. |
| `windowSize` |  | 否 | If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group. |
| `windowTimeZone` | string | 否 | The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). If not specified, the UTC timezone will be used. |
| `subject` | array<string> | 否 | Filtering by multiple subjects. |
| `filterCustomerId` | array<string> | 否 | Filtering by multiple customers. |
| `filterGroupBy` | object | 否 | Simple filter for group bys with exact match. |
| `advancedMeterGroupByFilters` | object | 否 | Optional advanced meter group by filters. You can use this to filter for values of the meter groupBy fields. |
| `groupBy` | array<string> | 否 | If not specified a single aggregate will be returned for each subject and time window. `subject` is a reserved group by value. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/meters/{meterIdOrSlug}/subjects`

**含义**：列出计量器
（原文：List meter subjects）

> List subjects for a meter.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterIdOrSlug` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `from` | string(date-time) | 否 | Start date-time in RFC 3339 format.  Inclusive. Defaults to the beginning of time.  For example: ?from=2025-01-01T00%3A00%3A00.000Z |
| `to` | string(date-time) | 否 | End date-time in RFC 3339 format.  Inclusive.  For example: ?to=2025-02-01T00%3A00%3A00.000Z |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Notifications


### GET `/api/v1/notification/channels`

**含义**：列出通知渠道
（原文：List notification channels）

> List all notification channels.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 | Include deleted notification channels in response.  Usage: `?includeDeleted=true` |
| `includeDisabled` | boolean | 否 | Include disabled notification channels in response.  Usage: `?includeDisabled=false` |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | NotificationChannelOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/notification/channels`

**含义**：创建通知渠道
（原文：Create a notification channel）

> Create a new notification channel.

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/notification/channels/{channelId}`

**含义**：更新通知渠道
（原文：Update a notification channel）

> Update notification channel.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `channelId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/notification/channels/{channelId}`

**含义**：查询通知渠道
（原文：Get notification channel）

> Get a notification channel by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `channelId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/notification/channels/{channelId}`

**含义**：删除通知渠道
（原文：Delete a notification channel）

> Soft delete notification channel by id.  Once a notification channel is deleted it cannot be undeleted.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `channelId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/notification/events`

**含义**：列出事件
（原文：List notification events）

> List all notification events.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `from` | string(date-time) | 否 | Start date-time in RFC 3339 format. Inclusive. |
| `to` | string(date-time) | 否 | End date-time in RFC 3339 format. Inclusive. |
| `feature` | array<string> | 否 | Filtering by multiple feature ids or keys.  Usage: `?feature=feature-1&feature=feature-2` |
| `subject` | array<string> | 否 | Filtering by multiple subject ids or keys.  Usage: `?subject=subject-1&subject=subject-2` |
| `rule` | array<string> | 否 | Filtering by multiple rule ids.  Usage: `?rule=01J8J2XYZ2N5WBYK09EDZFBSZM&rule=01J8J4R4VZH180KRKQ63NB2VA5` |
| `channel` | array<string> | 否 | Filtering by multiple channel ids.  Usage: `?channel=01J8J4RXH778XB056JS088PCYT&channel=01J8J4S1R1G9EVN62RG23A9M6J` |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | NotificationEventOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/notification/events/{eventId}`

**含义**：查询事件
（原文：Get notification event）

> Get a notification event by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `eventId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/notification/events/{eventId}/resend`

**含义**：创建事件
（原文：Re-send notification event）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `eventId` | string | 是 |  |

**请求体**（`NotificationEventResendRequest`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `channels` | array<string> | 否 | Notification channels to which the event should be re-sent. |

**响应**：202 The request has been accepted for processing, but processing has not yet completed., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/notification/rules`

**含义**：列出通知规则
（原文：List notification rules）

> List all notification rules.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 | Include deleted notification rules in response.  Usage: `?includeDeleted=true` |
| `includeDisabled` | boolean | 否 | Include disabled notification rules in response.  Usage: `?includeDisabled=false` |
| `feature` | array<string> | 否 | Filtering by multiple feature ids/keys.  Usage: `?feature=feature-1&feature=feature-2` |
| `channel` | array<string> | 否 | Filtering by multiple notifiaction channel ids.  Usage: `?channel=01ARZ3NDEKTSV4RRFFQ69G5FAV&channel=01J8J2Y5X4NNGQS32CF81W95E3` |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | NotificationRuleOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/notification/rules`

**含义**：创建通知规则
（原文：Create a notification rule）

> Create a new notification rule.

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/notification/rules/{ruleId}`

**含义**：更新通知规则
（原文：Update a notification rule）

> Update notification rule.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `ruleId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/notification/rules/{ruleId}`

**含义**：查询通知规则
（原文：Get notification rule）

> Get a notification rule by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `ruleId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/notification/rules/{ruleId}`

**含义**：删除通知规则
（原文：Delete a notification rule）

> Soft delete notification rule by id.  Once a notification rule is deleted it cannot be undeleted.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `ruleId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/notification/rules/{ruleId}/test`

**含义**：测试通知规则
（原文：Test notification rule）

> Test a notification rule by sending a test event with random data.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `ruleId` | string | 是 |  |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Portal


### GET `/api/v1/portal/meters/{meterSlug}/query`

**含义**：查询计量器
（原文：Query meter Query meter）

> Query meter for consumer portal. This endpoint is publicly exposable to consumers. Query meter for consumer portal. This endpoint is publicly exposable to consumers.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterSlug` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `clientId` | string | 否 | Client ID Useful to track progress of a query. |
| `from` | string(date-time) | 否 | Start date-time in RFC 3339 format.  Inclusive.  For example: ?from=2025-01-01T00%3A00%3A00.000Z |
| `to` | string(date-time) | 否 | End date-time in RFC 3339 format.  Inclusive.  For example: ?to=2025-02-01T00%3A00%3A00.000Z |
| `windowSize` | WindowSize | 否 | If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.  For example: ?windowSize=DAY |
| `windowTimeZone` | string | 否 | The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). If not specified, the UTC timezone will be used.  For example: ?windowTimeZone=UTC |
| `filterCustomerId` | array<string> | 否 | Filtering by multiple customers.  For example: ?filterCustomerId=customer-1&filterCustomerId=customer-2 |
| `filterGroupBy` | object | 否 | Simple filter for group bys with exact match.  For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo  ⚠️ __Deprecated__: Use `advancedMeterGroupByFilters` instead |
| `advancedMeterGroupByFilters` |  | 否 | Optional advanced meter group by filters. You can use this to filter for values of the meter groupBy fields. |
| `groupBy` | array<string> | 否 | If not specified a single aggregate will be returned for each subject and time window. `subject` is a reserved group by value.  For example: ?groupBy=subject&groupBy=model |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/portal/tokens`

**含义**：创建令牌
（原文：Create consumer portal token）

> Create a consumer portal token.

**请求体**（`PortalToken`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 否 | ULID (Universally Unique Lexicographically Sortable Identifier). |
| `subject` | string | 是 |  |
| `expiresAt` | string(date-time) | 否 | [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC. |
| `expired` | boolean | 否 |  |
| `createdAt` | string(date-time) | 否 | [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC. |
| `token` | string | 否 | The token is only returned at creation. |
| `allowedMeterSlugs` | array<string> | 否 | Optional, if defined only the specified meters will be allowed. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/portal/tokens`

**含义**：列出令牌
（原文：List consumer portal tokens）

> List tokens.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `limit` | integer | 否 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/portal/tokens/invalidate`

**含义**：使消费门户令牌失效
（原文：Invalidate portal tokens）

> Invalidates consumer portal tokens by ID or subject.

**请求体**

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `id` | string | 否 | Invalidate a portal token by ID. |
| `subject` | string | 否 | Invalidate all portal tokens for a subject. |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Product Catalog


### GET `/api/v1/addons`

**含义**：列出附加组件
（原文：List add-ons）

> List all add-ons.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 | Include deleted add-ons in response.  Usage: `?includeDeleted=true` |
| `id` | array<string> | 否 | Filter by addon.id attribute |
| `key` | array<string> | 否 | Filter by addon.key attribute |
| `keyVersion` | object | 否 | Filter by addon.key and addon.version attributes |
| `status` | array<AddonStatus> | 否 | Only return add-ons with the given status.  Usage: - `?status=active`: return only the currently active add-ons - `?status=draft`: return only the draft add-ons - `?status=archived`: return only the archived add-ons |
| `currency` | array<CurrencyCode> | 否 | Filter by addon.currency attribute |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | AddonOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/addons`

**含义**：创建附加组件
（原文：Create an add-on）

> Create a new add-on.

**请求体**（`AddonCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `key` | string | 是 | A semi-unique identifier for the resource. |
| `instanceType` |  | 是 | The instanceType of the add-ons. Can be "single" or "multiple". |
| `currency` |  | 是 | The currency code of the add-on. |
| `rateCards` | array<RateCard> | 是 | The rate cards of the add-on. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/addons/{addonId}`

**含义**：更新附加组件
（原文：Update add-on）

> Update add-on by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | string | 是 |  |

**请求体**（`AddonReplaceUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `instanceType` |  | 是 | The instanceType of the add-ons. Can be "single" or "multiple". |
| `rateCards` | array<RateCard> | 是 | The rate cards of the add-on. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/addons/{addonId}`

**含义**：查询附加组件
（原文：Get add-on）

> Get add-on by id or key. The latest published version is returned if latter is used.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeLatest` | boolean | 否 | Include latest version of the add-on instead of the version in active state.  Usage: `?includeLatest=true` |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/addons/{addonId}`

**含义**：删除附加组件
（原文：Delete add-on）

> Soft delete add-on by id.  Once a add-on is deleted it cannot be undeleted.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/addons/{addonId}/archive`

**含义**：归档附加组件
（原文：Archive add-on version）

> Archive a add-on version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/addons/{addonId}/publish`

**含义**：发布附加组件
（原文：Publish add-on）

> Publish a add-on version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `addonId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/features`

**含义**：列出功能特性
（原文：List features）

> List features.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `meterSlug` | array<string> | 否 | Filter by meterSlug |
| `includeArchived` | boolean | 否 | Include archived features in response. |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `offset` | integer | 否 | Number of items to skip.  Default is 0. |
| `limit` | integer | 否 | Number of items to return.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | FeatureOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/features`

**含义**：创建功能特性
（原文：Create feature）

> Features are either metered or static. A feature is metered if meterSlug is provided at creation. For metered features you can pass additional filters that will be applied when calculating feature usage, based on the meter's groupBy fields. Meters with SUM, COUNT, UNIQUE_COUNT and LATEST aggregations are supported for features.

**请求体**（`FeatureCreateInputs`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `key` | string | 是 | A key is a unique string that is used to identify a resource. |
| `name` | string | 是 |  |
| `metadata` |  | 否 |  |
| `meterSlug` | string | 否 | A key is a unique string that is used to identify a resource. |
| `meterGroupByFilters` | object | 否 | Optional meter group by filters. Useful if the meter scope is broader than what feature tracks. Example scenario would be a meter tracking all token use with groupBy fields for the model, then the feature could filter for model=gpt-4.  ⚠️ __Deprecated__: Use advancedMeterGroupByFilters instead |
| `advancedMeterGroupByFilters` | object | 否 | Optional advanced meter group by filters. You can use this to filter for values of the meter groupBy fields. |
| `unitCost` |  | 否 | Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or "llm" to look up cost from the LLM cost database based on meter group-by properties. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/features/{featureId}`

**含义**：查询功能特性
（原文：Get feature）

> Get a feature by ID.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `featureId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/features/{featureId}`

**含义**：删除功能特性
（原文：Delete feature）

> Archive a feature by ID.  Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements cannot be created for it, but archiving the feature does not affect existing entitlements. This means, if you want to create a new feature with the same key, and then create entitlements for it, the previous entitlements have to be deleted first on a per subject basis.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `featureId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/plans`

**含义**：列出套餐
（原文：List plans）

> List all plans.

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 | Include deleted plans in response.  Usage: `?includeDeleted=true` |
| `id` | array<string> | 否 | Filter by plan.id attribute |
| `key` | array<string> | 否 | Filter by plan.key attribute |
| `keyVersion` | object | 否 | Filter by plan.key and plan.version attributes |
| `status` | array<PlanStatus> | 否 | Only return plans with the given status.  Usage: - `?status=active`: return only the currently active plan - `?status=draft`: return only the draft plan - `?status=archived`: return only the archived plans |
| `currency` | array<CurrencyCode> | 否 | Filter by plan.currency attribute |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | PlanOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/plans`

**含义**：创建套餐
（原文：Create a plan）

> Create a new plan.

**请求体**（`PlanCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `key` | string | 是 | A semi-unique identifier for the resource. |
| `alignment` |  | 否 | Alignment configuration for the plan. |
| `currency` |  | 是 | The currency code of the plan. |
| `billingCadence` | string(duration) | 是 | The default billing cadence for subscriptions using this plan. Defines how often customers are billed using ISO8601 duration format. Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually). |
| `proRatingConfig` |  | 否 | Default pro-rating configuration for subscriptions using this plan. |
| `settlementMode` |  | 否 | The settlement mode of the plan. It determines how the billing system generates invoices and credits for the subscriptions using this plan. - credit_then_invoice: credits from the previous billing period are applied first, then the remaining balance is invoiced. - credit_only: only credits from the previous billing period are generated and applied. No invoices are generated for the subscription. This is the default and most common settlement mode. |
| `phases` | array<PlanPhase> | 是 | The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses. A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/plans/{planIdOrKey}/next`

**含义**：基于当前套餐创建下一草稿版本
（原文：New draft plan）

> Create a new draft version from plan. It returns error if there is already a plan in draft or planId does not reference the latest published version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planIdOrKey` | string | 是 |  |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/plans/{planId}`

**含义**：更新套餐
（原文：Update a plan）

> Update plan by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |

**请求体**（`PlanReplaceUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `alignment` |  | 否 | Alignment configuration for the plan. |
| `billingCadence` | string(duration) | 是 | The default billing cadence for subscriptions using this plan. Defines how often customers are billed using ISO8601 duration format. Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually). |
| `proRatingConfig` |  | 否 | Default pro-rating configuration for subscriptions using this plan. |
| `settlementMode` |  | 否 | The settlement mode of the plan. It determines how the billing system generates invoices and credits for the subscriptions using this plan. - credit_then_invoice: credits from the previous billing period are applied first, then the remaining balance is invoiced. - credit_only: only credits from the previous billing period are generated and applied. No invoices are generated for the subscription. This is the default and most common settlement mode. |
| `phases` | array<PlanPhase> | 是 | The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses. A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/plans/{planId}`

**含义**：查询套餐
（原文：Get plan）

> Get a plan by id or key. The latest published version is returned if latter is used.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeLatest` | boolean | 否 | Include latest version of the Plan instead of the version in active state.  Usage: `?includeLatest=true` |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/plans/{planId}`

**含义**：删除套餐
（原文：Delete plan）

> Soft delete plan by plan.id.  Once a plan is deleted it cannot be undeleted.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/plans/{planId}/addons`

**含义**：列出附加组件
（原文：List all available add-ons for plan）

> List all available add-ons for plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `includeDeleted` | boolean | 否 | Include deleted plan add-on assignments.  Usage: `?includeDeleted=true` |
| `id` | array<string> | 否 | Filter by addon.id attribute. |
| `key` | array<string> | 否 | Filter by addon.key attribute. |
| `keyVersion` | object | 否 | Filter by addon.key and addon.version attributes. |
| `page` | integer | 否 | Page index.  Default is 1. |
| `pageSize` | integer | 否 | The maximum number of items per page.  Default is 100. |
| `order` |  | 否 | The order direction. |
| `orderBy` | PlanAddonOrderBy | 否 | The order by field. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/plans/{planId}/addons`

**含义**：创建附加组件
（原文：Create new add-on assignment for plan）

> Create new add-on assignment for plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |

**请求体**（`PlanAddonCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `metadata` |  | 否 | Additional metadata for the resource. |
| `fromPlanPhase` | string | 是 | The key of the plan phase from the add-on becomes available for purchase. |
| `maxQuantity` | integer | 否 | The maximum number of times the add-on can be purchased for the plan. It is not applicable for add-ons with single instance type. |
| `addonId` | string | 是 | The add-on unique identifier in ULID format. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PUT `/api/v1/plans/{planId}/addons/{planAddonId}`

**含义**：更新附加组件
（原文：Update add-on assignment for plan）

> Update add-on assignment for plan.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |
| `planAddonId` | string | 是 |  |

**请求体**（`PlanAddonReplaceUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `metadata` |  | 否 | Additional metadata for the resource. |
| `fromPlanPhase` | string | 是 | The key of the plan phase from the add-on becomes available for purchase. |
| `maxQuantity` | integer | 否 | The maximum number of times the add-on can be purchased for the plan. It is not applicable for add-ons with single instance type. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/plans/{planId}/addons/{planAddonId}`

**含义**：查询附加组件
（原文：Get add-on assignment for plan）

> Get add-on assignment for plan by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |
| `planAddonId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/plans/{planId}/addons/{planAddonId}`

**含义**：删除附加组件
（原文：Delete add-on assignment for plan）

> Delete add-on assignment for plan.  Once a plan is deleted it cannot be undeleted.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |
| `planAddonId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/plans/{planId}/archive`

**含义**：归档套餐
（原文：Archive plan version）

> Archive a plan version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/plans/{planId}/publish`

**含义**：发布套餐
（原文：Publish plan）

> Publish a plan version.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `planId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Subjects


### GET `/api/v1/subjects`

**含义**：列出主体
（原文：List subjects）

> List subjects.  ⚠️ __Deprecated__: Subjects as managable entities are being depracated, use customers with subject key usage attribution instead.

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subjects`

**含义**：创建或更新主体
（原文：Upsert subject）

> Upserts a subject. Creates or updates subject.  If the subject doesn't exist, it will be created. If the subject exists, it will be partially updated with the provided fields.  ⚠️ __Deprecated__: Subjects as managable entities are being depracated, use customers with subject key usage attribution instead.

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subjects/{subjectIdOrKey}`

**含义**：查询主体
（原文：Get subject）

> Get subject by ID or key.  ⚠️ __Deprecated__: Subjects as managable entities are being depracated, use customers with subject key usage attribution instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/subjects/{subjectIdOrKey}`

**含义**：删除主体
（原文：Delete subject）

> Delete subject by ID or key.  ⚠️ __Deprecated__: Subjects as managable entities are being depracated, use customers with subject key usage attribution instead.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subjectIdOrKey` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


## Subscriptions


### POST `/api/v1/subscriptions`

**含义**：创建订阅
（原文：Create subscription）

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 409 The request could not be completed due to a conflict with the current state of the target resource.
Variants with ErrorExtensions specific to subscriptions., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subscriptions/{subscriptionId}`

**含义**：查询订阅
（原文：Get subscription）

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**查询参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `at` | string(date-time) | 否 | The time at which the subscription should be queried. If not provided the current time is used. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PATCH `/api/v1/subscriptions/{subscriptionId}`

**含义**：编辑订阅
（原文：Edit subscription）

> Batch processing commands for manipulating running subscriptions. The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**请求体**（`SubscriptionEdit`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `customizations` | array<SubscriptionEditOperation> | 是 | Batch processing commands for manipulating running subscriptions. The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`. |
| `timing` |  | 否 | Whether the billing period should be restarted.Timing configuration to allow for the changes to take effect at different times. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource.
Variants with ErrorExtensions specific to subscriptions., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### DELETE `/api/v1/subscriptions/{subscriptionId}`

**含义**：删除订阅
（原文：Delete subscription）

> Deletes a subscription. Only scheduled subscriptions can be deleted.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**响应**：204 There is no content to send for this request, but the headers may be useful. , 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource.
Variants with ErrorExtensions specific to subscriptions., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subscriptions/{subscriptionId}/addons`

**含义**：为订阅添加附加组件
（原文：Create subscription addon）

> Create a new subscription addon, either providing the key or the id of the addon.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**请求体**（`SubscriptionAddonCreate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `quantity` | integer | 是 | The quantity of the add-on. Always 1 for single instance add-ons. |
| `timing` |  | 是 | The timing of the operation. After the create or update, a new entry will be created in the timeline. |
| `addon` | object | 是 | The add-on to create. |

**响应**：201 The request has succeeded and a new resource has been created as a result., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subscriptions/{subscriptionId}/addons`

**含义**：列出附加组件
（原文：List subscription addons）

> List all addons of a subscription. In the returned list will match to a set unique by addonId.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### GET `/api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}`

**含义**：查询附加组件
（原文：Get subscription addon）

> Get a subscription addon by id.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |
| `subscriptionAddonId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### PATCH `/api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}`

**含义**：更新附加组件
（原文：Update subscription addon）

> Updates a subscription addon (allows changing the quantity: purchasing more instances or cancelling the current instances)

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |
| `subscriptionAddonId` | string | 是 |  |

**请求体**（`SubscriptionAddonUpdate`）

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 否 | Human-readable name for the resource. Between 1 and 256 characters. |
| `description` | string | 否 | Optional description of the resource. Maximum 1024 characters. |
| `metadata` | object | 否 | Additional metadata for the resource. |
| `quantity` | integer | 否 | The quantity of the add-on. Always 1 for single instance add-ons. |
| `timing` |  | 否 | The timing of the operation. After the create or update, a new entry will be created in the timeline. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subscriptions/{subscriptionId}/cancel`

**含义**：取消订阅
（原文：Cancel subscription）

> Cancels the subscription. Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancellation time.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**请求体**

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `timing` |  | 否 | If not provided the subscription is canceled immediately. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource.
Variants with ErrorExtensions specific to subscriptions., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subscriptions/{subscriptionId}/change`

**含义**：变更订阅
（原文：Change subscription）

> Closes a running subscription and starts a new one according to the specification. Can be used for upgrades, downgrades, and plan changes.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource.
Variants with ErrorExtensions specific to subscriptions., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subscriptions/{subscriptionId}/migrate`

**含义**：迁移订阅
（原文：Migrate subscription）

> Migrates the subscripiton to the provided version of the current plan. If possible, the migration will be done immediately. If not, the migration will be scheduled to the end of the current billing period.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**请求体**

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `timing` |  | 否 | Timing configuration for the migration, when the migration should take effect. If not supported by the subscription, 400 will be returned. |
| `targetVersion` | integer | 否 | The version of the plan to migrate to. If not provided, the subscription will migrate to the latest version of the current plan. |
| `startingPhase` | string | 否 | The key of the phase to start the subscription in. If not provided, the subscription will start in the first phase of the plan. |
| `billingAnchor` | string(date-time) | 否 | The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the previous subscription billing anchor will be used. |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource.
Variants with ErrorExtensions specific to subscriptions., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subscriptions/{subscriptionId}/restore`

**含义**：恢复订阅
（原文：Restore subscription）

> Restores a canceled subscription. Any subscription scheduled to start later will be deleted and this subscription will be continued indefinitely.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing)., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.


### POST `/api/v1/subscriptions/{subscriptionId}/unschedule-cancelation`

**含义**：取消计划的订阅注销
（原文：Unschedule cancelation）

> Cancels the scheduled cancelation.

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `subscriptionId` | string | 是 |  |

**响应**：200 The request has succeeded., 400 The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions., 401 The request has not been applied because it lacks valid authentication credentials for the target resource., 403 The server understood the request but refuses to authorize it., 404 The origin server did not find a current representation for the target resource or is not willing to disclose that one exists., 409 The request could not be completed due to a conflict with the current state of the target resource.
Variants with ErrorExtensions specific to subscriptions., 412 One or more conditions given in the request header fields evaluated to false when tested on the server., 500 The server encountered an unexpected condition that prevented it from fulfilling the request., 503 The server is currently unable to handle the request due to a temporary overload or scheduled maintenance, which will likely be alleviated after some delay., default An unexpected error response.
