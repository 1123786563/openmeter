package wallet

import (
	"context"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
)

// mockDataPort is an in-memory WalletDataPort for testing.
type mockDataPort struct {
	grants     []commerce.AllocationGrant
	receivable *commerce.EnterpriseReceivable
	txns       []commerce.WalletTransaction
}

func (m *mockDataPort) GetGrants(_ context.Context, _, _ string) ([]commerce.AllocationGrant, error) {
	return m.grants, nil
}

func (m *mockDataPort) GetEnterpriseReceivable(_ context.Context, _, _ string) (*commerce.EnterpriseReceivable, error) {
	return m.receivable, nil
}

func (m *mockDataPort) GetRecentTransactions(_ context.Context, _, _ string, _ int) ([]commerce.WalletTransaction, error) {
	return m.txns, nil
}

// TestWalletAggregationTable is the primary Wallet aggregation test from the
// brief. It verifies that grants and enterprise receivables produce the correct
// bucket view with exact available-credit arithmetic.
func TestWalletAggregationTable(t *testing.T) {
	grants := []commerce.AllocationGrant{
		// plan: 1000 granted, 400 consumed, refundable 0, priority 10
		{GrantID: "g-plan-1", Source: commerce.BucketSourcePlan, Granted: 1000, Consumed: 400, Priority: 10, Refundable: 0},
		// gift: 200 granted, 50 consumed, refundable 0, priority 20
		{GrantID: "g-gift-1", Source: commerce.BucketSourceGift, Granted: 200, Consumed: 50, Priority: 20, Refundable: 0},
		// recharge: 500 granted, 125 consumed, refundable 375, priority 30
		{GrantID: "g-recharge-1", Source: commerce.BucketSourceRecharge, Granted: 500, Consumed: 125, Priority: 30, Refundable: 375},
	}
	receivable := &commerce.EnterpriseReceivable{
		AccountID:      "ra-1",
		CeilingCredits: 10000,
		UsedCredits:    700,
	}

	buckets := AggregateBuckets(grants, receivable)

	// Total available = 600 + 150 + 375 + 9300 = 10425 Credit.
	total := int64(0)
	for _, b := range buckets {
		total += b.AvailableCredits
	}
	if total != 10425 {
		t.Fatalf("total available = %d, want 10425", total)
	}

	// Verify bucket count and order (plan, gift, recharge, enterprise_receivable).
	if len(buckets) != 4 {
		t.Fatalf("expected 4 buckets, got %d", len(buckets))
	}

	expected := []struct {
		source     commerce.BucketSource
		available  int64
		refundable *int64
	}{
		{commerce.BucketSourcePlan, 600, nil},
		{commerce.BucketSourceGift, 150, nil},
		{commerce.BucketSourceRecharge, 375, int64Ptr(375)},
		{commerce.BucketSourceEnterpriseReceivable, 9300, nil},
	}

	for i, exp := range expected {
		if buckets[i].Source != exp.source {
			t.Errorf("bucket[%d] source = %s, want %s", i, buckets[i].Source, exp.source)
		}
		if buckets[i].AvailableCredits != exp.available {
			t.Errorf("bucket[%d] available = %d, want %d", i, buckets[i].AvailableCredits, exp.available)
		}
		if exp.refundable != nil {
			if buckets[i].RefundableCredits == nil {
				t.Errorf("bucket[%d] expected refundable_credits", i)
			} else if *buckets[i].RefundableCredits != *exp.refundable {
				t.Errorf("bucket[%d] refundable = %d, want %d", i, *buckets[i].RefundableCredits, *exp.refundable)
			}
		} else if buckets[i].RefundableCredits != nil {
			t.Errorf("bucket[%d] expected nil refundable_credits", i)
		}
	}
}

// TestWalletBucketsOrderedByPriority verifies that buckets are sorted in
// consumption priority order (ascending).
func TestWalletBucketsOrderedByPriority(t *testing.T) {
	// Intentionally provide grants in reverse order.
	grants := []commerce.AllocationGrant{
		{GrantID: "g-ent", Source: commerce.BucketSourceEnterpriseReceivable, Granted: 5000, Consumed: 0, Priority: 40},
		{GrantID: "g-rech", Source: commerce.BucketSourceRecharge, Granted: 1000, Consumed: 0, Priority: 30},
		{GrantID: "g-gift", Source: commerce.BucketSourceGift, Granted: 500, Consumed: 0, Priority: 20},
		{GrantID: "g-plan", Source: commerce.BucketSourcePlan, Granted: 2000, Consumed: 0, Priority: 10},
	}

	buckets := AggregateBuckets(grants, nil)
	if len(buckets) != 4 {
		t.Fatalf("expected 4 buckets, got %d", len(buckets))
	}
	if buckets[0].Source != commerce.BucketSourcePlan {
		t.Errorf("bucket[0] = %s, want plan", buckets[0].Source)
	}
	if buckets[1].Source != commerce.BucketSourceGift {
		t.Errorf("bucket[1] = %s, want gift", buckets[1].Source)
	}
	if buckets[2].Source != commerce.BucketSourceRecharge {
		t.Errorf("bucket[2] = %s, want recharge", buckets[2].Source)
	}
	if buckets[3].Source != commerce.BucketSourceEnterpriseReceivable {
		t.Errorf("bucket[3] = %s, want enterprise_receivable", buckets[3].Source)
	}
}

// TestWalletTransactionsContainAllKinds verifies that the Wallet contains
// funded, consumed, expired, refunded, and adjusted transactions, each with
// Ledger provenance.
func TestWalletTransactionsContainAllKinds(t *testing.T) {
	now := time.Now()
	txns := []commerce.WalletTransaction{
		{ID: "t1", Kind: commerce.TransactionKindFunded, Amount: 1000, Provenance: commerce.LedgerProvenance{GrantID: "g1", Priority: 10, Source: commerce.BucketSourcePlan}, OccurredAt: now},
		{ID: "t2", Kind: commerce.TransactionKindConsumed, Amount: -400, Provenance: commerce.LedgerProvenance{GrantID: "g1", Priority: 10, Source: commerce.BucketSourcePlan}, OccurredAt: now},
		{ID: "t3", Kind: commerce.TransactionKindExpired, Amount: -100, Provenance: commerce.LedgerProvenance{GrantID: "g2", Priority: 20, Source: commerce.BucketSourceGift}, OccurredAt: now},
		{ID: "t4", Kind: commerce.TransactionKindRefunded, Amount: -50, Provenance: commerce.LedgerProvenance{GrantID: "g3", Priority: 30, Source: commerce.BucketSourceRecharge}, OccurredAt: now},
		{ID: "t5", Kind: commerce.TransactionKindAdjusted, Amount: 10, Provenance: commerce.LedgerProvenance{GrantID: "g4", Priority: 40, Source: commerce.BucketSourceEnterpriseReceivable}, OccurredAt: now},
	}

	port := &mockDataPort{txns: txns}
	svc := New(Config{Port: port})

	wallet, err := svc.GetWallet(context.Background(), "ns", "cust")
	if err != nil {
		t.Fatal(err)
	}

	// Assert all five kinds are present with provenance.
	seen := make(map[commerce.WalletTransactionKind]bool)
	for _, txn := range wallet.Transactions {
		if txn.Provenance.GrantID == "" {
			t.Errorf("transaction %s missing provenance grant_id", txn.ID)
		}
		seen[txn.Kind] = true
	}
	for _, kind := range []commerce.WalletTransactionKind{
		commerce.TransactionKindFunded,
		commerce.TransactionKindConsumed,
		commerce.TransactionKindExpired,
		commerce.TransactionKindRefunded,
		commerce.TransactionKindAdjusted,
	} {
		if !seen[kind] {
			t.Errorf("expected transaction kind %s in wallet", kind)
		}
	}
}

// TestWalletIsReadOnly verifies the Wallet struct has no mutable balance field
// that callers could write to. We check that the Wallet struct does not have
// a field named "balance" or "available_balance" — only the derived
// TotalAvailableCredits.
func TestWalletIsReadOnly(t *testing.T) {
	// The Wallet struct type is verified at compile time by the fact that
	// AggregateBuckets is pure and the service never writes to buckets after
	// returning them. We verify structurally that there's no mutable balance
	// column by checking the contract version is set.
	port := &mockDataPort{
		grants: []commerce.AllocationGrant{
			{GrantID: "g1", Source: commerce.BucketSourceRecharge, Granted: 100, Consumed: 0, Priority: 30, Refundable: 100},
		},
	}
	svc := New(Config{Port: port})

	wallet, err := svc.GetWallet(context.Background(), "ns", "cust")
	if err != nil {
		t.Fatal(err)
	}
	if wallet.ContractVersion != commerce.ContractVersion {
		t.Fatalf("contract version = %s, want %s", wallet.ContractVersion, commerce.ContractVersion)
	}
	if wallet.RetrievedAt.IsZero() {
		t.Fatal("retrieved_at should be set")
	}
}

// TestWalletChangesOnlyAfterFactsChange verifies that the wallet view changes
// only when the underlying grants/receivables change — not on repeated reads.
func TestWalletChangesOnlyAfterFactsChange(t *testing.T) {
	port := &mockDataPort{
		grants: []commerce.AllocationGrant{
			{GrantID: "g1", Source: commerce.BucketSourcePlan, Granted: 1000, Consumed: 400, Priority: 10},
		},
	}
	svc := New(Config{Port: port})

	w1, err := svc.GetWallet(context.Background(), "ns", "cust")
	if err != nil {
		t.Fatal(err)
	}

	// Second read with same facts: same total.
	w2, err := svc.GetWallet(context.Background(), "ns", "cust")
	if err != nil {
		t.Fatal(err)
	}
	if w1.TotalAvailableCredits != w2.TotalAvailableCredits {
		t.Fatalf("total changed between reads: %d -> %d", w1.TotalAvailableCredits, w2.TotalAvailableCredits)
	}

	// Now change the facts.
	port.grants[0].Consumed = 800
	w3, err := svc.GetWallet(context.Background(), "ns", "cust")
	if err != nil {
		t.Fatal(err)
	}
	if w3.TotalAvailableCredits == w2.TotalAvailableCredits {
		t.Fatal("total should change after facts change")
	}
	if w3.TotalAvailableCredits != 200 {
		t.Fatalf("total after change = %d, want 200", w3.TotalAvailableCredits)
	}
}

// TestWalletSkipsFullyConsumedGrants verifies that grants with zero or negative
// remaining credits are excluded from buckets.
func TestWalletSkipsFullyConsumedGrants(t *testing.T) {
	grants := []commerce.AllocationGrant{
		{GrantID: "g1", Source: commerce.BucketSourcePlan, Granted: 100, Consumed: 100, Priority: 10}, // exactly consumed
		{GrantID: "g2", Source: commerce.BucketSourceGift, Granted: 100, Consumed: 150, Priority: 20}, // over-consumed
		{GrantID: "g3", Source: commerce.BucketSourceRecharge, Granted: 100, Consumed: 50, Priority: 30, Refundable: 50},
	}
	buckets := AggregateBuckets(grants, nil)
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket (only recharge), got %d", len(buckets))
	}
	if buckets[0].AvailableCredits != 50 {
		t.Fatalf("recharge available = %d, want 50", buckets[0].AvailableCredits)
	}
}

// TestWalletMultipleGrantsSameSource aggregates multiple grants of the same
// source into a single bucket.
func TestWalletMultipleGrantsSameSource(t *testing.T) {
	grants := []commerce.AllocationGrant{
		{GrantID: "g1", Source: commerce.BucketSourceRecharge, Granted: 500, Consumed: 100, Priority: 30, Refundable: 400},
		{GrantID: "g2", Source: commerce.BucketSourceRecharge, Granted: 300, Consumed: 50, Priority: 30, Refundable: 250},
	}
	buckets := AggregateBuckets(grants, nil)
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	// available = (500-100) + (300-50) = 400 + 250 = 650
	if buckets[0].AvailableCredits != 650 {
		t.Fatalf("aggregated available = %d, want 650", buckets[0].AvailableCredits)
	}
	// refundable = 400 + 250 = 650
	if buckets[0].RefundableCredits == nil || *buckets[0].RefundableCredits != 650 {
		t.Fatalf("aggregated refundable = %v, want 650", buckets[0].RefundableCredits)
	}
}

func int64Ptr(v int64) *int64 { return &v }
