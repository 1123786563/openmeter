# OpenMeter — Copilot Instructions

OpenMeter is a real-time usage metering and billing engine written in Go.
Full guidance also lives in `AGENTS.md` (source of truth) and
`docs/development/*.md` — read those for anything not covered here.

## Repository layout & module invariants

- Three Go modules: the root (production), `api/v3/client` (publishable Go
  SDK), and `e2e` (test-only). **Never** add `api/v3/client` as a root
  dependency — it is not independently tagged and local `replace` directives
  don't work for downstream consumers.
- Root `go build ./...`, `go test ./...`, `go vet ./...` exclude `e2e`. Use
  `make etoe` or `go test -C e2e ./...` for it.
- Entry points live in `cmd/{server,sink-worker,balance-worker,billing-worker,
  notification-service,jobs}`, each with its own Wire-generated `wire_gen.go`.
- Domain packages live under `openmeter/<domain>/` (e.g. `customer`,
  `billing`, `subscription`, `entitlement`, `productcatalog`, `credit`,
  `notification`). Dependency injection wiring lives in `app/common/` (one
  file per domain, e.g. `app/common/customer.go`) and is registered per
  binary in `cmd/<service>/wire.go`.
- `openmeter/ent/` holds the Ent ORM schema/generated code; migrations live in
  `tools/migrate/migrations` and must stay in sync with the schema (see
  `make migrate-check`).
- `api/spec` (TypeSpec) is the source of truth for the API — OpenAPI specs and
  client SDKs are generated from it (see `api/spec/AGENTS.md`).

## Architecture (big picture)

Kafka decouples ingestion from processing. PostgreSQL (via Ent) is the
source of truth for transactional state (customers, catalog, subscriptions,
billing, entitlements); ClickHouse stores/aggregates high-volume usage
events; Redis is optional (ingest dedup, query progress). See
`docs/architecture.md` for the full diagram. Runtime components:

| Component | Role |
|---|---|
| API server (`cmd/server`) | Ingest, management API, queries, webhook delivery (leader-elected Svix reconciler) |
| Sink worker | Validates/dedupes/persists usage events from Kafka into ClickHouse |
| Balance worker | Entitlements and credit balance recalculation |
| Billing worker | Subscription sync, rating, invoice lifecycle |
| Notification service | Rule evaluation, pending notification events |
| Jobs | Scheduled migrations and cross-domain maintenance |
| Collector (`collector/`) | Optional standalone forwarder of external events into the ingest API |

Every entity is namespaced (multi-tenant). Namespace flows through
`namespaceDecoder.GetNamespace(ctx)` at the HTTP layer and as an explicit
field on service/adapter input structs — never derived from ambient context
deeper in the stack.

## Build, test, lint

```bash
make up                 # start Postgres/Kafka/ClickHouse/etc via docker compose
make server              # run API server with hot reload (air)
make build               # build all binaries
make generate             # regenerate code (ent, wire, mocks, etc.)
make test                 # full test suite; requires `docker compose up -d postgres`
make lint                 # lint-go + lint-api-spec + lint-openapi + lint-helm
make etoe                 # end-to-end tests (via e2e module)
```

Run a targeted test (needs Postgres at 127.0.0.1):

```bash
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./openmeter/<domain>/...
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -run '^TestName$' ./openmeter/<domain>/...
```

- Root-package tests using `confluent-kafka-go` require `-tags=dynamic`. The
  `e2e` module does not need it.
- Do not wrap test invocations in `sh -lc`/`bash -lc`; run commands directly.
- `make lint-go` covers root + `api/v3/client` + `e2e` modules; `make mod`
  runs `go mod tidy` across all three plus `collector`.
- `make test-go-sdk` builds/vets/tests the `api/v3/client` module in
  isolation.
- When a Postgres-backed test fails, the output includes a `testdbconf:` URL
  for the per-test DB — `psql` into it (quote the connection string) before
  rerunning, to inspect state for RCA.

## Service package conventions

Each domain package under `openmeter/<domain>/` follows this shape:

```
openmeter/<domain>/
├── service.go     # Service interface + input DTOs + Validate()
├── adapter.go     # Adapter interface (persistence contract)
├── <domain>.go    # Domain types/models
├── errors.go      # Custom errors wrapping pkg/models/errors.go (optional)
├── event.go       # Domain events (optional; packages that mutate entities)
├── adapter/       # Ent-backed implementation: adapter.go, <op>.go, mapping.go
├── service/       # Business logic + orchestration: service.go
└── driver/        # v1 API only — do not add for new services
api/v3/handlers/<domain>/<api_operation>/   # v3 API handlers
```

- **Root** owns types/interfaces/DTOs/validation only, no implementation.
  **Service** owns business rules, transaction orchestration, event
  publishing, hooks. **Adapter** owns Ent queries and entity↔domain mapping
  only — no business decisions.
- All input structs implement `Validate() error`, collecting field errors and
  returning `models.NewNillableGenericValidationError(errors.Join(errs...))`.
- Use `models.NamespacedID` for Get/Delete inputs identifying a namespaced
  entity.
- Wrap service-layer transactions with `transaction.Run()` /
  `transaction.RunWithNoValue()`. Never introduce
  `transaction.RunInNewTransaction()` (breaks atomicity with the caller's
  transaction) without explicit human review.
- Adapters implement the `Tx`/`WithTx`/`Self` transaction boilerplate and wrap
  each method in `entutils.TransactingRepo()` /
  `TransactingRepoWithNoValue()`.
- Custom errors wrap generic types from `pkg/models/errors.go`
  (`NewGenericNotFoundError`, `NewGenericConflictError`,
  `NewGenericValidationError`, etc.) — only add them when they add real value.
- Wire dependency injection lives in `app/common/<domain>.go`; register the
  provider set in the relevant `cmd/<service>/wire.go`, then run
  `make generate`.
- Anti-patterns: no nested subdirectories beyond `adapter/`/`service/`; no
  "connector" layer between service/adapter; no domain types in subpackages;
  no business logic in adapters; no package-level global state; no
  `driver`/`httpdriver` packages for new services (handlers live in
  `api/v3/handlers/`).

Full detail: `docs/development/service-patterns.md`. New/changed Ent schemas
follow the `db-migration` skill; new API handlers follow the `api` skill.

## Go conventions

- String enum constants: `<Type><Value>` (e.g. `InvoiceStatusDraft`).
- Don't extract trivial/single-use helpers unless the name captures
  non-obvious domain intent; inline pass-through wrappers.
- Don't hide type switching, validation, persistence mapping, or domain
  translation in local closures — use a named helper.
- Never introduce `context.Background()`/`context.TODO()` to bypass context
  propagation; never `panic` in non-test code.
- Production constructors take a `*slog.Logger` explicitly — no
  `slog.Default()` fallback.
- Prefer standard `slices`/`maps`, then `github.com/samber/lo`. Reuse
  `pkg/slicesx` and `pkg/syncx.OnceValues` instead of local `ptr`/`must`
  wrappers.
- Type-translation helpers between API/domain/DB use `map`/`mapped` naming
  (see `docs/development/type-conversions.md`), not `project`/`projected`.

## Testing conventions

- Test types: unit (`openmeter/<domain>/<domain>_test.go`, no DB), integration
  (`adapter/*_test.go`, real Postgres), service (`service/*_test.go`, full
  stack via `TestEnv`).
- Build domain `testutils/` packages independent of `app/common`; construct
  repositories/adapters/services/locks from their own package constructors.
  Reference pattern: `openmeter/customer/testutils/env.go`.
- Integration/service tests auto-skip when `POSTGRES_HOST` is unset (via
  `testutils.InitPostgresDB`).
- Use `t.Context()`, `require` for fatal assertions / `assert` for soft ones,
  table-driven tests with `t.Run()`, and `testutils.NewTestULID(t)` /
  `testutils.NewTestNamespace(t)` for random identifiers (one namespace per
  test).
- Pair `clock.FreezeTime(...)`/`clock.SetTime(...)` immediately with
  `defer clock.ResetTime()`/`UnFreeze()`. Don't use `WithinDuration(...)` for
  clock-sensitive assertions — assert exact equality after freezing time.
- Compare `alpacadecimal.Decimal` via `InexactFloat64()` with `require.Equal`.
- Begin non-trivial service/lifecycle subtests with `given`/`when`/`then`
  intent comments.
- For usage-based billing lifecycle tests, drive behavior through
  `charges.Service.Create` / `AdvanceCharges` / `ApplyPatches` rather than
  low-level charge adapters.

Full detail: `docs/development/testing.md`.

## Documentation conventions

- Comments/docstrings explain intent, domain constraints, lifecycle state,
  and failure consequences not obvious from the code — don't narrate what the
  reader can already see.
- Preserve explanatory comments during refactors unless the change makes them
  false or misleading.

## Further guidance

- TypeSpec/SDK generator work: `api/spec/AGENTS.md`.
- Deep dives: `docs/development/service-patterns.md`, `testing.md`,
  `refactoring.md`, `type-conversions.md`, `collection-helpers.md`.
- Domain behavior changes: use the `domain-docs` skill to locate relevant
  package documentation before making changes.
- Ambiguous/cross-cutting design work: use the `iterative-engineering-design`
  skill.
