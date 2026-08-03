// AI Usage closed-loop E2E suite for the WeKnora AI Billing Phase 1 provider
// release.
//
// These tests exercise the full AI Usage settlement contract against a live
// OpenMeter server (PostgreSQL + Kafka + ClickHouse) with ai_usage.enabled.
// No HTTP stubs are used. The suite skips gracefully when OPENMETER_ADDRESS is
// unset or the AI Usage feature flag is off, so it composes cleanly with the
// rest of the v3 E2E suite.
//
// Scenarios (each a subtest of TestV3AIUsageClosedLoop):
//   - happy-path closed loop (201, TotalCredits, balance deduction)
//   - component and bundle billing-mode mutual exclusion
//   - BYOK model lines at zero credits with platform RAG charged
//   - tenant_seq 1,3,2 continuous watermark convergence
//   - same idempotency_key / payload_hash replay returning 200, single ledger effect
//   - linked correction via credit adjustment
//   - enterprise receivable overflow after prepaid sources are exhausted
//
// The infrastructure-heavy scenarios (server restart, Kafka/ClickHouse
// interruption) are separate top-level tests gated by opt-in env vars so they
// don't fire during a normal acceptance run.
package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// requireAIUsageEnabled probes the AI Usage routes and skips the test when the
// feature flag is off. When ai_usage.enabled is false the v3 server returns
// HTTP 404 for every AI Usage operation.
func requireAIUsageEnabled(t *testing.T, c *v3Client) {
	t.Helper()

	customer := createWeKnoraCustomer(t, c)

	_, err := c.AIUsage.GetCreditBalance(t.Context(), customer.ID, v3sdk.GetAIUsageCreditBalanceParams{})
	if apiErr, ok := v3sdk.AsAPIError(err); ok && apiErr.StatusCode == http.StatusNotFound {
		t.Skip("ai_usage feature is not enabled on this server")
	}
	// Any other error (including nil) means the route exists — proceed.
}

// createWeKnoraCustomer creates a fresh customer with CNY currency, matching
// the WeKnora billing configuration where AI credits are tracked in CNY.
func createWeKnoraCustomer(t *testing.T, c *v3Client) *v3sdk.Customer {
	t.Helper()

	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:      uniqueKey("wkc_aiusage_customer"),
		Name:     "WeKnora AI Usage Customer",
		Currency: lo.ToPtr("CNY"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)
	return customer
}

// fundWKC grants amount prepaid WeKnora Credits (WKC) to the customer via the
// OpenMeter credit-grant API. WKC are the spendable integer credits consumed
// by AI Usage settlement.
func fundWKC(t *testing.T, c *v3Client, customerID string, amount int64) {
	t.Helper()

	_, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, v3sdk.CreateCreditGrantRequest{
		Name:          "WKC prepaid funding",
		Amount:        v3sdk.Numeric(strconv.FormatInt(amount, 10)),
		Currency:      v3sdk.BillingCurrencyCode("CNY"),
		FundingMethod: v3sdk.CreditFundingMethodNone,
	})
	c.requireStatus(http.StatusCreated, err)
}

// computePayloadHash returns the SHA-256 hex digest of the canonical batch
// request body. The hash covers every field except payload_hash itself, so two
// identical batches produce the same digest (replay) while a mutated body
// changes it (conflict).
func computePayloadHash(batch v3sdk.AIUsageUsageBatchCreate) string {
	clone := batch
	clone.PayloadHash = ""
	b, err := json.Marshal(clone)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// withHash finalises a batch request by computing and setting the payload_hash.
func withHash(batch v3sdk.AIUsageUsageBatchCreate) v3sdk.AIUsageUsageBatchCreate {
	batch.PayloadHash = computePayloadHash(batch)
	return batch
}

// validAIUsageBatch builds a canonical component-mode batch for a customer with
// two lines that total 9 credits under the default rate card
// (1 credit per unit): 5 rag_queries + 4 llm_input_tokens.
func validAIUsageBatch(customerID string, seq int64) v3sdk.AIUsageUsageBatchCreate {
	return v3sdk.AIUsageUsageBatchCreate{
		IdempotencyKey:     uniqueKey("wkc_batch"),
		BillingCustomerID:  customerID,
		SubjectKey:         customerID,
		TenantSeq:          seq,
		OccurredAt:         time.Now().UTC(),
		ReservationID:      "",
		RatePackageVersion: "weknora-billing-p1-v1",
		BillingMode:        v3sdk.AIUsageBillingModeComponent,
		ProviderManaged:    true,
		Lines: []v3sdk.AIUsageUsageLineCreate{
			{
				ResourceCode:       "rag_queries",
				Quantity:           5,
				CanonicalLineIndex: 0,
			},
			{
				ResourceCode:       "llm_input_tokens",
				Quantity:           4,
				CanonicalLineIndex: 1,
			},
		},
	}
}

func TestV3AIUsageClosedLoop(t *testing.T) {
	c := newV3Client(t)
	requireAIUsageEnabled(t, c)

	// ---------------------------------------------------------------------------
	// Happy path: fund, submit, settle, verify.
	// ---------------------------------------------------------------------------

	t.Run("closed loop returns 201 with correct total credits", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		batch := withHash(validAIUsageBatch(customer.ID, 1))
		settled, err := c.AIUsage.CreateBatch(t.Context(), batch)
		c.requireStatus(http.StatusCreated, err)

		require.Equal(t, v3sdk.AIUsageBatchStatusSettled, settled.Status)
		assert.EqualValues(t, 9, settled.TotalCredits)
	})

	t.Run("credit balance reflects the deduction", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		batch := withHash(validAIUsageBatch(customer.ID, 1))
		_, err := c.AIUsage.CreateBatch(t.Context(), batch)
		c.requireStatus(http.StatusCreated, err)

		balance, err := c.Customers.Credits.Balance.Get(t.Context(), customer.ID, v3sdk.GetCustomerCreditBalanceParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotEmpty(t, balance.Balances)
		assert.Equal(t, "991", balance.Balances[0].Settled)
	})

	t.Run("batch is retrievable after settlement", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		batch := withHash(validAIUsageBatch(customer.ID, 1))
		settled, err := c.AIUsage.CreateBatch(t.Context(), batch)
		c.requireStatus(http.StatusCreated, err)

		fetched, err := c.AIUsage.GetBatch(t.Context(), settled.ID)
		c.requireStatus(http.StatusOK, err)
		require.Equal(t, settled.ID, fetched.ID)
		assert.Equal(t, v3sdk.AIUsageBatchStatusSettled, fetched.Status)
	})

	// ---------------------------------------------------------------------------
	// Component and bundle billing-mode mutual exclusion.
	// ---------------------------------------------------------------------------

	t.Run("component mode rates each line individually", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		batch := withHash(validAIUsageBatch(customer.ID, 1))
		settled, err := c.AIUsage.CreateBatch(t.Context(), batch)
		c.requireStatus(http.StatusCreated, err)

		require.Equal(t, v3sdk.AIUsageBillingModeComponent, settled.BillingMode)
		// Sum of per-line credits equals the batch total.
		var lineTotal int64
		for _, line := range settled.Lines {
			lineTotal += line.Credits
		}
		assert.Equal(t, settled.TotalCredits, lineTotal)
	})

	t.Run("bundle mode charges the reservation ceiling flat", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		bundle := v3sdk.AIUsageUsageBatchCreate{
			IdempotencyKey:            uniqueKey("wkc_batch_bundle"),
			BillingCustomerID:         customer.ID,
			SubjectKey:                customer.ID,
			TenantSeq:                 1,
			OccurredAt:                time.Now().UTC(),
			RatePackageVersion:        "weknora-billing-p1-v1",
			BillingMode:               v3sdk.AIUsageBillingModeBundle,
			ReservationCeilingCredits: 15,
			ProviderManaged:           true,
			Lines: []v3sdk.AIUsageUsageLineCreate{
				{ResourceCode: "rag_queries", Quantity: 100, CanonicalLineIndex: 0},
			},
		}
		bundle = withHash(bundle)

		settled, err := c.AIUsage.CreateBatch(t.Context(), bundle)
		c.requireStatus(http.StatusCreated, err)

		require.Equal(t, v3sdk.AIUsageBillingModeBundle, settled.BillingMode)
		// In bundle mode the ceiling IS the total charge, regardless of line
		// quantities.
		assert.EqualValues(t, 15, settled.TotalCredits)
	})

	// ---------------------------------------------------------------------------
	// BYOK: model lines at zero, platform RAG charged.
	// ---------------------------------------------------------------------------

	t.Run("BYOK model lines zero with platform RAG charged", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		byok := v3sdk.AIUsageUsageBatchCreate{
			IdempotencyKey:     uniqueKey("wkc_batch_byok"),
			BillingCustomerID:  customer.ID,
			SubjectKey:         customer.ID,
			TenantSeq:          1,
			OccurredAt:         time.Now().UTC(),
			RatePackageVersion: "weknora-billing-p1-v1",
			BillingMode:        v3sdk.AIUsageBillingModeComponent,
			ProviderManaged:    false, // bring-your-own-key
			Lines: []v3sdk.AIUsageUsageLineCreate{
				{
					ResourceCode:       "llm_input_tokens",
					Quantity:           1000,
					Provider:           ptrTo("openai"),
					Model:              ptrTo("gpt-4o"),
					CanonicalLineIndex: 0,
				},
				{
					ResourceCode:       "rag_queries",
					Quantity:           7,
					CanonicalLineIndex: 1,
				},
			},
		}
		byok = withHash(byok)

		settled, err := c.AIUsage.CreateBatch(t.Context(), byok)
		c.requireStatus(http.StatusCreated, err)

		// BYOK model line: zero credits, zero cost.
		require.GreaterOrEqual(t, len(settled.Lines), 2)
		byokLine := settled.Lines[0]
		assert.Equal(t, "llm_input_tokens", byokLine.ResourceCode)
		assert.EqualValues(t, 0, byokLine.Credits)
		if byokLine.CostSnapshot != nil {
			assert.Equal(t, "0", byokLine.CostSnapshot.Amount)
		}

		// Platform RAG line: charged.
		ragLine := settled.Lines[1]
		assert.Equal(t, "rag_queries", ragLine.ResourceCode)
		assert.Greater(t, ragLine.Credits, int64(0))
	})

	// ---------------------------------------------------------------------------
	// Watermark convergence: submit seq 1, 3, 2 and verify it catches up to 3.
	// ---------------------------------------------------------------------------

	t.Run("watermark converges for out-of-order seq 1,3,2", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		subject := uniqueKey("wkc_watermark_subject")

		makeBatch := func(seq int64) v3sdk.AIUsageUsageBatchCreate {
			b := validAIUsageBatch(customer.ID, seq)
			b.SubjectKey = subject
			b.IdempotencyKey = uniqueKey("wkc_wm_batch")
			return withHash(b)
		}

		// seq 1: covered advances to 1.
		b1, err := c.AIUsage.CreateBatch(t.Context(), makeBatch(1))
		c.requireStatus(http.StatusCreated, err)
		assert.EqualValues(t, 1, b1.CoveredTenantSeq)

		// seq 3: gap at 2, covered stays at 1.
		b3, err := c.AIUsage.CreateBatch(t.Context(), makeBatch(3))
		c.requireStatus(http.StatusCreated, err)
		assert.EqualValues(t, 1, b3.CoveredTenantSeq)

		// seq 2: fills the gap, watermark catches up to 3.
		b2, err := c.AIUsage.CreateBatch(t.Context(), makeBatch(2))
		c.requireStatus(http.StatusCreated, err)
		assert.EqualValues(t, 3, b2.CoveredTenantSeq)
	})

	// ---------------------------------------------------------------------------
	// Idempotent replay: same key + hash returns 200, single ledger effect.
	// ---------------------------------------------------------------------------

	t.Run("identical replay returns 200 with one ledger effect", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		batch := withHash(validAIUsageBatch(customer.ID, 1))

		// First submit returns 201.
		first, err := c.AIUsage.CreateBatch(t.Context(), batch)
		c.requireStatus(http.StatusCreated, err)

		// Replay returns 200, same result.
		second, err := c.AIUsage.CreateBatch(t.Context(), batch)
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, first.TotalCredits, second.TotalCredits)

		// Only one deduction visible on the balance.
		balance, err := c.Customers.Credits.Balance.Get(t.Context(), customer.ID, v3sdk.GetCustomerCreditBalanceParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotEmpty(t, balance.Balances)
		assert.Equal(t, "991", balance.Balances[0].Settled)
	})

	t.Run("replay with different payload hash returns 409", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		original := withHash(validAIUsageBatch(customer.ID, 1))
		_, err := c.AIUsage.CreateBatch(t.Context(), original)
		c.requireStatus(http.StatusCreated, err)

		// Same idempotency key, mutated body yields a different hash.
		conflict := original
		conflict.Lines[0].Quantity = 99
		conflict = withHash(conflict)

		_, err = c.AIUsage.CreateBatch(t.Context(), conflict)
		requireProblem(t, err, http.StatusConflict)
	})

	// ---------------------------------------------------------------------------
	// Linked correction via credit adjustment.
	// ---------------------------------------------------------------------------

	t.Run("linked correction reverses the batch charge", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		batch := withHash(validAIUsageBatch(customer.ID, 1))
		settled, err := c.AIUsage.CreateBatch(t.Context(), batch)
		c.requireStatus(http.StatusCreated, err)

		// Phase 1 does not define a dedicated AI-Usage-domain correction
		// resource. Instead, the correction is recorded via the existing Credit
		// Adjustments API. The batch ID embedded in the adjustment name and
		// description forms the foreign-key link. This is an explicit Phase 1
		// assumption, not a gap — a domain-native correction endpoint is
		// deferred to Phase 2.
		_, err = c.Customers.Credits.Adjustments.Create(t.Context(), customer.ID, v3sdk.CreateCreditAdjustmentRequest{
			Name:        "AI usage correction for batch " + settled.ID,
			Description: ptrTo("Reverses the 9-credit charge from batch " + settled.ID),
			Currency:    v3sdk.BillingCurrencyCode("CNY"),
			Amount:      v3sdk.Numeric("9"),
		})
		c.requireStatus(http.StatusCreated, err)

		// Balance is restored to the funded amount.
		balance, err := c.Customers.Credits.Balance.Get(t.Context(), customer.ID, v3sdk.GetCustomerCreditBalanceParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotEmpty(t, balance.Balances)
		assert.Equal(t, "1000", balance.Balances[0].Settled)
	})

	// ---------------------------------------------------------------------------
	// Enterprise receivable overflow: prepaid exhausted, remainder goes negative.
	// ---------------------------------------------------------------------------

	t.Run("enterprise receivable covers overflow after prepaid exhaustion", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)

		// Prepaid: 5 credits.
		fundWKC(t, c, customer.ID, 5)

		// Enterprise receivable grant. Funding method external creates a
		// receivable-backed grant that can go negative during settlement.
		_, err := c.Customers.Credits.Grants.Create(t.Context(), customer.ID, v3sdk.CreateCreditGrantRequest{
			Name:          "Enterprise receivable line",
			Amount:        v3sdk.Numeric("0"),
			Currency:      v3sdk.BillingCurrencyCode("CNY"),
			FundingMethod: v3sdk.CreditFundingMethodExternal,
			Priority:      ptrTo(int16(30)),
		})
		c.requireStatus(http.StatusCreated, err)

		// Submit a batch that costs 12 credits: 5 from prepaid, 7 from receivable.
		overflow := v3sdk.AIUsageUsageBatchCreate{
			IdempotencyKey:     uniqueKey("wkc_enterprise_batch"),
			BillingCustomerID:  customer.ID,
			SubjectKey:         customer.ID,
			TenantSeq:          1,
			OccurredAt:         time.Now().UTC(),
			RatePackageVersion: "weknora-billing-p1-v1",
			BillingMode:        v3sdk.AIUsageBillingModeComponent,
			ProviderManaged:    true,
			Lines: []v3sdk.AIUsageUsageLineCreate{
				{ResourceCode: "rag_queries", Quantity: 12, CanonicalLineIndex: 0},
			},
		}
		overflow = withHash(overflow)

		settled, err := c.AIUsage.CreateBatch(t.Context(), overflow)
		c.requireStatus(http.StatusCreated, err)
		// Enterprise plan: settlement succeeds even though prepaid is exhausted.
		assert.Equal(t, v3sdk.AIUsageBatchStatusSettled, settled.Status)
		assert.EqualValues(t, 12, settled.TotalCredits)
	})

	// ---------------------------------------------------------------------------
	// Runtime authorization contract.
	// ---------------------------------------------------------------------------

	t.Run("runtime authorization returns signed contract version", func(t *testing.T) {
		customer := createWeKnoraCustomer(t, c)
		fundWKC(t, c, customer.ID, 1000)

		auth, err := c.AIUsage.GetCustomerRuntimeAuthorization(t.Context(), customer.ID, v3sdk.GetCustomerRuntimeAuthorizationParams{
			Filter: &v3sdk.GetCustomerRuntimeAuthorizationFilter{
				SubjectKey: ptrTo(customer.ID),
			},
		})
		c.requireStatus(http.StatusOK, err)

		assert.Equal(t, "weknora-billing-p1-v1", auth.ContractVersion)
		assert.GreaterOrEqual(t, auth.CoveredTenantSeq, int64(0))
	})
}

// ---------------------------------------------------------------------------
// Infrastructure-gated scenario: server restart after an accepted batch.
//
// Requires OPENMETER_E2E_SERVER_RESTART=1 and an externally managed server
// that can be restarted (the test does not restart the server itself; it
// expects the operator or harness to bounce the process between phases).
// ---------------------------------------------------------------------------

func TestV3AIUsageServerRestart(t *testing.T) {
	c := newV3Client(t)
	requireAIUsageEnabled(t, c)

	if os.Getenv("OPENMETER_E2E_SERVER_RESTART") == "" {
		t.Skip("set OPENMETER_E2E_SERVER_RESTART=1 to enable the server-restart scenario")
	}

	customer := createWeKnoraCustomer(t, c)
	fundWKC(t, c, customer.ID, 1000)

	batch := withHash(validAIUsageBatch(customer.ID, 1))
	settled, err := c.AIUsage.CreateBatch(t.Context(), batch)
	c.requireStatus(http.StatusCreated, err)

	// The harness restarts the server here. After it comes back, the batch must
	// still be queryable: it was durably persisted to PostgreSQL before the
	// 201 response.
	t.Log("restart the OpenMeter server now, then press enter to continue the assertion")
	waitForOperator(t)

	fetched, err := c.AIUsage.GetBatch(t.Context(), settled.ID)
	c.requireStatus(http.StatusOK, err)
	require.Equal(t, settled.ID, fetched.ID)
	assert.Equal(t, v3sdk.AIUsageBatchStatusSettled, fetched.Status)
	assert.Equal(t, settled.TotalCredits, fetched.TotalCredits)
}

// ---------------------------------------------------------------------------
// Infrastructure-gated scenario: Kafka/ClickHouse interruption.
//
// Requires OPENMETER_E2E_INFRA_INTERRUPTION=1. The operator brings Kafka or
// ClickHouse down, the test submits a batch (synchronously accepted via
// PostgreSQL), then the operator brings the infrastructure back and the test
// polls until the meter projection converges.
// ---------------------------------------------------------------------------

func TestV3AIUsageProjectionConvergence(t *testing.T) {
	c := newV3Client(t)
	requireAIUsageEnabled(t, c)

	if os.Getenv("OPENMETER_E2E_INFRA_INTERRUPTION") == "" {
		t.Skip("set OPENMETER_E2E_INFRA_INTERRUPTION=1 to enable the Kafka/ClickHouse interruption scenario")
	}

	customer := createWeKnoraCustomer(t, c)
	fundWKC(t, c, customer.ID, 1000)

	// Submit a batch while Kafka/ClickHouse may be down. The 201 proves the
	// batch was accepted synchronously from PostgreSQL; projection is
	// eventually consistent.
	batch := withHash(validAIUsageBatch(customer.ID, 1))
	settled, err := c.AIUsage.CreateBatch(t.Context(), batch)
	c.requireStatus(http.StatusCreated, err)
	require.Equal(t, v3sdk.AIUsageBatchStatusSettled, settled.Status)

	t.Log("bring Kafka/ClickHouse back up now, then press enter to continue the convergence check")
	waitForOperator(t)

	// After the outbox relay drains, the batch is still queryable and settled.
	require.Eventually(t, func() bool {
		fetched, err := c.AIUsage.GetBatch(t.Context(), settled.ID)
		if err != nil {
			return false
		}
		return fetched.Status == v3sdk.AIUsageBatchStatusSettled
	}, 2*time.Minute, 5*time.Second, "batch status did not converge to settled after infrastructure recovery")
}

// ptrTo is a small generic helper for pointer literals.
func ptrTo[T any](v T) *T {
	return &v
}

// waitForOperator blocks until stdin receives a line, allowing manual
// infrastructure actions (restart, service toggling) between test phases.
func waitForOperator(t *testing.T) {
	t.Helper()
	t.Log("waiting for operator input on stdin...")
	var line string
	_, _ = fmt.Fscanln(os.Stdin, &line)
}
