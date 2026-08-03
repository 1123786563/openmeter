# OpenMeter AI Billing — Phase 1 Implementation Plan

## Context

This plan extends the OpenMeter fork with a domain-native AI Usage billing system for WeKnora Standard SaaS integration. It implements Phase 1 ("AI Billing") of a two-phase design approved through brainstorming.

Phase 1 delivers: formally executable AI billing — Canonical Usage Batch settlement, full AI Meter resource classification, Cost Catalog + Customer Rate Card pricing with cost/sales separation, and Credit Settlement Engine with strict settlement order. No online payment or recharge (Phase 2).

### Key Design Decisions (approved)

1. Extension approach A: domain-native isolated extension — new `aiusage` domain package, reuse existing credit/ledger/billing infrastructure.
2. Canonical Usage Batch: one business action = one atomic batch with multiple line items (tokens, RAG, MCP, web search, etc.).
3. Identity model A: Billing Customer (payer) vs Subject/Tenant (usage producer); Enterprise can map multiple Tenants to one Billing Customer.
4. Unified integer Credit: all pricing output is integer Credit; CNY decimal only for display.
5. Cost/sales separation: provider cost and customer sales price tracked independently.
6. Settlement order: plan Credit, gift Credit, recharge Credit, enterprise receivable.
7. BYOK: model lines cost 0 and sales 0; platform resource lines still charged.
8. Double-billing prevention: one batch is either component or bundle, never both.

## Global Constraints

- Go module `github.com/openmeterio/openmeter`, Go 1.26.
- Follow existing OpenMeter conventions: ent schemas, slog logger, `models.NillableGenericValidationError`, `entutils` mixins, `transaction.Run` for atomic operations.
- String enum constants: `<Type><Value>` (e.g., `BatchStatusSettled`).
- No `context.Background()` or `panic` in production code.
- Tests use `t.Context()`, PostgreSQL-backed tests require `POSTGRES_HOST=127.0.0.1`.
- `go build ./...` must pass after each task.
- New ent schemas use `entutils.IDMixin{}`, `entutils.MetadataMixin{}`, `entutils.TimeMixin{}`.
- `go generate ./...` after adding ent schemas.

---

## Task 1: AI Usage Domain Types and Resource Classification

**Goal:** Establish the foundational domain types, constants, and errors for the `aiusage` package.

**Files to create:**
- `openmeter/aiusage/types.go` — core domain types
- `openmeter/aiusage/resource.go` — resource classification constants
- `openmeter/aiusage/errors.go` — domain errors
- `openmeter/aiusage/types_test.go` — validation tests

**Types to define in `types.go`:**
- `BillingMode` (string enum: `component`, `bundle`) with `Validate()`
- `BatchStatus` (string enum: `pending`, `settled`, `rejected`, `compensated`)
- `UsageLineItem` struct (resource_code, quantity, provider, model, provider_managed, dimensions) with `Validate()`
- `AIUsageBatch` struct (namespace, customer_id, subject_id, usage_batch_id, tenant_seq, occurred_at, reservation_id, ceiling_credits, rate_version, billing_mode, payload_hash, status, line_items) with `Validate()`
- `RatingSnapshot` struct (resource_code, cost_snapshot, sales_snapshot, credits)
- `CostSnapshot` struct (currency, amount, source)
- `SalesSnapshot` struct (currency, amount, rate_card_version)
- `BatchSettlementResult` struct (batch_id, status, total_credits, rating_snapshots, ledger_entries, covered_tenant_seq)
- `LedgerEntryRef` struct (grant_id, amount, priority)
- `IngestBatchInput` struct (the API input matching AIUsageBatch fields)

**Resource classification in `resource.go`:**
- `ResourceCode` type (string)
- 16 constants: chat_input_token, chat_output_token, chat_cache_read_token, chat_cache_write_token, chat_reasoning_token, embedding_token, rerank_call, vlm_input_token, vlm_output_token, vlm_image, asr_seconds, rag_retrieval, doc_parse_page, mcp_call, web_search, agent_run
- `IsProviderManaged()` method: true for model tokens, false for platform resources
- `IsPlatformResource()` method: true for RAG/MCP/web/agent/doc_parse
- `Unit()` method: returns "token", "call", "second", "page", "image"
- `ValidResourceCode()` function

**Errors in `errors.go`:**
- `ErrInvalidBillingMode`, `ErrInvalidBatchStatus`, `ErrInvalidResourceCode`
- `ErrBatchAlreadyExists`, `ErrBatchPayloadConflict` (for idempotency 409)
- `ErrMissingRateCard`, `ErrInsufficientCredits`, `ErrCeilingExceeded`
- Follow existing pattern using `pkg/errors` with `.WithAttr()` chain

**Validation requirements:**
- AIUsageBatch.Validate(): namespace/customer/subject non-empty, usage_batch_id non-empty, tenant_seq > 0, billing_mode valid, line_items non-empty unless bundle, payload_hash non-empty.
- UsageLineItem.Validate(): resource_code valid, quantity > 0.
- Bundle mode requires CeilingCredits set; component mode requires LineItems non-empty.
- Provider-managed resources must have provider and model set.

**Test requirements:**
- Valid batch passes validation.
- Missing required fields fail with field-level errors.
- Invalid billing_mode fails.
- Each ResourceCode classification returns expected values.

---

## Task 2: AI Pricing — Customer Rate Card and Credit Calculation

**Goal:** Create the Customer Rate Card domain that maps provider costs to customer sales prices with cost/sales separation and integer Credit output.

**Files to create:**
- `openmeter/aiusage/ratecard.go` — Rate Card types and resolver interface
- `openmeter/aiusage/ratecard_test.go` — tests for resolution priority
- `openmeter/aiusage/creditcalc.go` — integer Credit calculation
- `openmeter/aiusage/creditcalc_test.go` — tests

**Rate Card types:**
- `CustomerRateCardEntry` struct (namespace, customer_id, resource_code, provider, model, price_per_unit_cny, credit_rate, effective_from, effective_to)
- `RateCardResolver` interface with `Resolve(ctx, namespace, customerID, resource, provider, model, at)` method

**Resolution priority (most specific wins):**
1. Customer + provider + model specific
2. Customer + provider
3. Customer + resource only
4. Namespace default + provider + model
5. Namespace default + resource only

**Credit calculation:**
- `CalculateCredits(salesAmountCNY, creditRate)`: credits = ceil(salesAmountCNY * creditRate), rounding always up
- `CalculateLineCredits(quantity, pricePerUnitCNY, creditRate)`: per-line credit cost

**Key rules:**
- Credits are always positive integers (use math.Ceil on decimal multiplication)
- BYOK resources: cost=0, sales=0, credits=0
- Missing rate for provider-managed resource: batch rejected (fail-closed)
- Missing rate for platform resource: use namespace default

**Test requirements:**
- Credit calc: ceil(0.002 * 1000) = 2, ceil(0.0015 * 1000) = 2
- Rate resolution priority: customer-specific overrides namespace default
- Time-windowed: effective_from/effective_to respected
- BYOK returns 0 credits

---

## Task 3: ent Schema for AI Usage Domain

**Goal:** Create ent entity schemas for AIUsageBatch, UsageLineItem, RatingSnapshot, and CustomerRateCardEntry, then generate ent code.

**Files to create:**
- `openmeter/ent/schema/aiusage_batch.go`
- `openmeter/ent/schema/aiusage_line_item.go`
- `openmeter/ent/schema/aiusage_rating_snapshot.go`
- `openmeter/ent/schema/aiusage_ratecard_entry.go`

**After creating schemas, run:** `go generate ./openmeter/ent/...`

**AIUsageBatch schema:**
- Mixin: IDMixin, MetadataMixin, TimeMixin
- Fields: namespace, customer_id, subject_id, usage_batch_id, tenant_seq (int64), occurred_at (time), reservation_id (optional), ceiling_credits (int64, optional, nillable), rate_version, billing_mode, payload_hash, status (default "pending"), total_credits (int64, default 0), covered_tenant_seq (int64, default 0)
- Edges: line_items (one-to-many AIUsageLineItem), rating_snapshots (one-to-many AIUsageRatingSnapshot)
- Indexes: unique (namespace, usage_batch_id) WHERE deleted_at IS NULL; index (namespace, customer_id, tenant_seq); index (namespace, customer_id, status)

**AIUsageLineItem schema:**
- Mixin: IDMixin, MetadataMixin, TimeMixin
- Fields: namespace, resource_code, quantity (int64), provider, model, provider_managed (bool, default true), dimensions (jsonb, optional)
- Edge: batch (many-to-one AIUsageBatch)

**AIUsageRatingSnapshot schema:**
- Fields: namespace, resource_code, cost_currency, cost_amount (numeric), cost_source, sales_currency, sales_amount (numeric), rate_card_version, credits (int64)
- Edge: batch (many-to-one AIUsageBatch)

**AIUsageRatecardEntry schema:**
- Fields: namespace, customer_id (optional, nillable), resource_code, provider (optional, nillable), model (optional, nillable), price_per_unit_cny (numeric), credit_rate (int64), effective_from (time), effective_to (time, optional, nillable)
- Indexes: unique (namespace, customer_id, resource_code, provider, model, effective_from) WHERE deleted_at IS NULL; index (namespace, resource_code)

**Reference for patterns:** `openmeter/ent/schema/llmcostprice.go` and `openmeter/ent/schema/grant.go`

**Acceptance criteria:**
- `go generate ./openmeter/ent/...` succeeds
- `go build ./...` succeeds
- All four entity types appear in generated ent client

---

## Task 4: Credit Settlement Engine

**Goal:** Implement settlement orchestration that deducts Credits in strict priority order within a single PostgreSQL transaction.

**Files to create:**
- `openmeter/aiusage/settlement.go` — settlement engine
- `openmeter/aiusage/settlement_test.go` — tests

**Settlement engine interface:**
- `SettlementEngine` interface with `Settle(ctx, batch, snapshots, ceiling)` returning `BatchSettlementResult`

**Settlement order (strict):**
1. Plan Credit (grant priority 0): subscription monthly Credits
2. Gift Credit (grant priority 10): promotional Credits
3. Recharge Credit (grant priority 20): paid wallet Credits
4. Enterprise receivable (priority 30): for Enterprise, unpaid usage becomes receivable

**Key logic:**
- Total credits = sum of all RatingSnapshot.Credits for chargeable resources
- If ceiling set and total > ceiling: total = ceiling (platform absorbs difference)
- Use existing credit.BalanceConnector and credit engine burn mechanism
- Enterprise: grants exhausted then remainder to receivable, batch succeeds
- Non-Enterprise: grants exhausted then batch rejected (fail-closed), no partial deduction

**Integration with existing credit system:**
- Reference: `openmeter/credit/engine/run.go` for burn pattern
- Reference: `openmeter/credit/balance/balance.go` for Map/Burn
- Reference: `openmeter/credit/grant.go` for grant structure

**Test requirements:**
- Single grant type deducts correctly
- Multi-grant priority: plan before gift before recharge
- Ceiling enforcement: total capped
- Enterprise receivable: remainder to receivable
- Non-Enterprise insufficient: batch rejected, no partial deduction
- Zero-credit batch (BYOK only): settled with 0 credits

---

## Task 5: AI Usage Batch Service and Adapter

**Goal:** Implement the core service that receives, validates, rates, and settles a Canonical Usage Batch in one orchestrated flow.

**Files to create:**
- `openmeter/aiusage/service.go` — service interface and implementation
- `openmeter/aiusage/adapter.go` — batch adapter (validation, normalization, idempotency)
- `openmeter/aiusage/connector.go` — DI connector
- `openmeter/aiusage/service_test.go` — tests with mock dependencies

**Service interface:**
- `IngestBatch(ctx, input)`: validate, resolve rates, rate lines, apply ceiling, settle, persist atomically
- `GetBatch(ctx, namespace, batchID)`: retrieve
- `GetCoveredSeq(ctx, namespace, customerID)`: watermark

**IngestBatch flow:**
1. Validate schema, identity, sequence, billing_mode, payload_hash
2. Idempotency check: usage_batch_id exists + same payload_hash then return stored result
3. Idempotency conflict: usage_batch_id exists + different payload_hash then return 409
4. Resolve rate version for occurred_at
5. Rate every line: cost from llmcost, sales from Rate Card, calculate Credits
6. Apply batch ceiling if set
7. SettlementEngine.Settle() in PostgreSQL transaction
8. Persist AIUsageBatch + LineItems + RatingSnapshots + ledger entries atomically
9. Return BatchSettlementResult

**Repository interface (defined in service.go, implemented in Task 6):**
- CreateBatch, GetBatch, GetBatchByBatchID, GetCoveredSeq

**Test requirements:**
- Happy path: mixed resources settle correctly
- Idempotency: duplicate batch_id + same hash returns original
- Conflict: duplicate batch_id + different hash returns 409
- BYOK batch: model lines 0, platform lines charged
- Bundle mode: flat ceiling, no per-line rating
- Missing rate: fail-closed
- Ceiling exceeded: total capped

---

## Task 6: ent Repository Implementation

**Goal:** Implement Repository interface using generated ent client.

**Files to create:**
- `openmeter/aiusage/adapter/repository.go` — ent-backed repository
- `openmeter/aiusage/adapter/repository_test.go` — tests (skip without PostgreSQL)

**Key methods:**
- CreateBatch: transaction, create batch + line items + snapshots; check idempotency first
- GetBatch: eager-load line_items and rating_snapshots
- GetCoveredSeq: compute highest continuously settled tenant_seq

**Pattern reference:** `openmeter/credit/adapter/` and `openmeter/llmcost/adapter/`

**Test requirements:**
- Create and retrieve batch with children
- Idempotency: duplicate returns existing
- Conflict: different hash returns error
- GetCoveredSeq returns correct watermark

---

## Task 7: App Wiring and Service Registration

**Goal:** Wire the AI Usage service into the OpenMeter application container.

**Files to create:**
- `openmeter/aiusage/app.go` — app module definition
- `openmeter/app/aiusage.go` — DI registration (following `openmeter/app/registry.go` pattern)

**Acceptance criteria:**
- `go build ./...` succeeds
- Service constructable with mock dependencies

---

## Task 8: Comprehensive Integration Test Suite

**Goal:** End-to-end integration tests covering the full batch settlement flow.

**Files to create:**
- `openmeter/aiusage/integration_test.go` — integration tests (skip without PostgreSQL)

**Test scenarios:**
1. Full batch settlement: chat tokens + RAG + MCP in one batch
2. Settlement order: plan before gift before recharge
3. BYOK mixed batch: model lines 0 + platform lines charged
4. Bundle mode: ceiling enforced
5. Idempotency: replay same batch returns same result
6. Conflict: same batch_id + different payload = 409
7. Ceiling enforcement: total > ceiling capped
8. Enterprise receivable: grants exhausted then remainder to receivable
9. Non-enterprise insufficient: grants exhausted then rejected
10. Sequence gap: batch processes but watermark waits for gap fill

**Acceptance criteria:**
- All tests pass or skip gracefully without PostgreSQL
- Tests use real ent client with test database
