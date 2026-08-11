# Task 6 implementation report

## Result

- Audited the inherited provider callback implementation from the cumulative Task 1-5 branch. The existing code already had WeChat `204` with an empty body, Alipay `200 text/plain` with an exact `success` body, a 1 MiB body reader limit, retryable `500` handling, provider-native ACKs for legal replays, and provider-specific TypeSpec success models.
- Corrected the remaining contract mismatch: `payment.ErrContradictoryPaymentFact` is now an explicit deterministic `400 Bad Request`, matching invalid signatures and payment fact mismatches. It is no longer returned as `409 Conflict` by the callback handler.
- Removed `409` from both provider callback TypeSpec operations and regenerated the v3 OpenAPI/server artifact. The Go v3 client generator was run; it produced no tracked client diff because the success request/response shapes did not change and the removed error response is not represented in its generated method signatures.
- Preserved the Task 3-5 payment/refund service semantics; no payment or refund domain code was changed.

## Changed files

- `api/spec/packages/aip/src/commerce/operations.tsp`
- `api/v3/handlers/commerce/handler.go`
- `api/v3/handlers/commerce/handler_test.go`
- `api/v3/openapi.yaml`
- `api/v3/api.gen.go`
- `task-6-report.md`

`api/spec/packages/aip/src/commerce/payments.tsp` and `api/v3/client/*` were audited and regenerated respectively, but required no tracked changes.

## TDD evidence

Red:

```text
GOWORK=off go test ./api/v3/handlers/commerce -run '^TestPaymentCallbackContradictoryFactIsBadRequest$' -count=1
--- FAIL: TestPaymentCallbackContradictoryFactIsBadRequest
expected 400, got 409
```

Green after the minimal handler mapping change:

```text
GOWORK=off go test ./api/v3/handlers/commerce -run '^TestPaymentCallbackContradictoryFactIsBadRequest$' -count=1
ok github.com/openmeterio/openmeter/api/v3/handlers/commerce
```

The callback tests now also assert:

- exactly 1 MiB is accepted and reaches verification;
- 1 MiB + 1 byte returns `413` before the payment service is called;
- duplicate provider events and already-fulfilled orders return WeChat and Alipay's native ACKs;
- TxRunner/database, wrapped cancellation, and wrapped deadline errors return retryable `500` responses;
- invalid signatures, payment fact mismatches, and contradictory facts return deterministic `400` responses without a success ACK.

## Generation

- `make update-openapi` was attempted. The ambient shell initially lacked Node; rerunning with the repository's `.nvmrc` Node v26.4.0 reached generation.
- The aggregate target then stalled in the unrelated legacy Python SDK emitter after reporting that Python >=3.10 was required while the host has Python 3.9.6. The idle process was interrupted after several minutes with no CPU, network activity, or output.
- The required AIP artifacts were generated successfully with the official package command:

```text
pnpm --dir api/spec --filter @openmeter/api-spec-aip run generate
```

- The same `api/spec/Makefile` AIP post-processing and OpenAPI bundle step was applied, followed by:

```text
GOWORK=off go generate ./api/...
```

- Generated output outside Task 6's allowed scope was discarded. Only `api/v3/openapi.yaml` and `api/v3/api.gen.go` remain changed. The AIP generator reported existing warnings, including documentation and repeated-prefix warnings; it exited successfully.

## Verification

```text
GOWORK=off go test ./api/v3 ./api/v3/handlers/commerce -count=1
ok github.com/openmeterio/openmeter/api/v3
ok github.com/openmeterio/openmeter/api/v3/handlers/commerce

GOWORK=off go test -C api/v3/client ./... -count=1
ok github.com/openmeterio/openmeter/api/v3/client

pnpm --dir api/spec --filter @openmeter/api-spec-aip run lint
All matched files use Prettier code style!

git diff --check
exit 0
```

Direct OpenAPI assertions passed:

- WeChat has `204` with no response content/schema;
- Alipay has `200` with a `text/plain` string response;
- both callbacks expose `400`, `413`, and `500`;
- neither callback exposes `409`.

## Not verified and risk

- The complete aggregate `make update-openapi` target could not finish because of the host's legacy Python SDK emitter requirement. The Task 6 AIP generator, bundle, v3 server generation, handler tests, and nested v3 client tests all completed successfully.
- Full repository tests and live provider callbacks were outside Task 6's focused scope and were not run.
- Residual risk is limited to the aggregate legacy generator environment; the changed v3 callback contract and generated artifacts were verified directly.

## Fix Round 1

### Result

- Removed the callback handler's generic `ValidationIssue`/commerce-status fallback. Callback failures now use an explicit deterministic whitelist: signature, provider protocol, payment fact mismatch/contradiction, and both payment-attempt-not-found sentinels return `400`; every unknown error defaults to retryable `500`.
- Added `payment.ErrRetryableCallback` provenance and applied it to unexpected attempt/fact repository failures, non-success fact inserts, all `PaidTxRunner` failures, missing transaction results, and post-transition/replay reload failures. The handler checks this marker before legal-replay or deterministic mapping, so a transaction failure wrapping `commerce.ErrOrderNotFound` cannot become `404`.
- Preserved provider-native legal replay acknowledgments: WeChat remains empty `204`; Alipay remains `200 text/plain; charset=utf-8` with body exactly `success`.
- TypeSpec/OpenAPI/generated outputs were not modified because the public contract did not change.

### TDD evidence

Red before implementation:

```text
GOWORK=off go test ./api/v3/handlers/commerce -run '^TestPaymentCallbackErrorClassification$' -count=1
unknown provider order from production repository: expected 400, got 500
unknown provider order from payment repository: expected 400, got 404
paid transition wraps missing order: expected 500, got 404
non-whitelisted payment validation issue: expected 500, got 402
non-whitelisted commerce validation issue: expected 500, got 409

GOWORK=off go test ./openmeter/commerce/payment -run 'TestApplyPaymentFact_(PaidTransitionFailureIsRetryable|AttemptLookupErrorProvenance|FactLookupFailureIsRetryable|InsertAndReloadFailuresAreRetryable)' -count=1
build failed: undefined: ErrRetryableCallback
```

Green after implementation:

```text
GOWORK=off go test ./openmeter/commerce/payment -run 'TestApplyPaymentFact_(PaidTransitionFailureIsRetryable|AttemptLookupErrorProvenance|FactLookupFailureIsRetryable|InsertAndReloadFailuresAreRetryable)' -count=1
ok github.com/openmeterio/openmeter/openmeter/commerce/payment

GOWORK=off go test ./api/v3/handlers/commerce -run '^(TestPaymentCallbackErrorClassification|TestPaymentCallbackLegalReplayUsesProviderSuccessAck)$' -count=1
ok github.com/openmeterio/openmeter/api/v3/handlers/commerce
```

The table-driven handler coverage includes unknown provider orders, a marked paid-transition failure wrapping `commerce.ErrOrderNotFound`, retryable provenance wrapping a deterministic sentinel, generic database failures, wrapped cancellation/deadline errors, non-whitelisted validation issues, and both legal replay sentinels for both providers.

### Verification

```text
GOWORK=off go test ./api/v3 ./api/v3/handlers/commerce ./openmeter/commerce ./openmeter/commerce/payment/... ./openmeter/commerce/refund -count=1
PASS (all seven packages)

GOWORK=off go test -C api/v3/client ./... -count=1
PASS

GOWORK=off go test -race ./api/v3/handlers/commerce ./openmeter/commerce/payment ./openmeter/commerce/payment/wechat ./openmeter/commerce/payment/alipay -count=1
PASS (all four packages)

GOWORK=off go vet ./api/v3/handlers/commerce ./openmeter/commerce/payment/...
PASS

git diff --check
PASS

git diff --exit-code ac78622 -- api/spec api/v3/openapi.yaml api/v3/api.gen.go api/v3/client
PASS (no TypeSpec/OpenAPI/generated-client drift)
```

Direct inspection confirms the callback OpenAPI responses remain Alipay `200` and WeChat `204`, plus `400`, inherited `401/403`, `413`, and `500`; no `402`, `404`, or `409` callback response is declared.

### Not verified and risk

- Live WeChat/Alipay delivery was not exercised; verification is focused on service/handler behavior and generated-contract consistency.
- The aggregate legacy generator was not rerun because this round has no contract change and the prior report documents the host Python fallback blocker. The checked-in TypeSpec/OpenAPI/server/client artifacts have no diff from the reviewed Task 6 commit.
- Remaining risk is limited to unexercised live provider retry timing; the production typed error paths and callback response classifications are covered by focused tests.
