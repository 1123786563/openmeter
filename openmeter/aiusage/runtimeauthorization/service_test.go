package runtimeauthorization_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
)

// ---- mock dependencies ----

type mockBalanceReader struct {
	balance runtimeauthorization.CreditBalance
	err     error
}

func (m *mockBalanceReader) ReadCreditBalance(_ context.Context, _, _ string) (runtimeauthorization.CreditBalance, error) {
	return m.balance, m.err
}

type mockSubscriptionReader struct {
	info runtimeauthorization.SubscriptionInfo
	err  error
}

func (m *mockSubscriptionReader) ReadSubscription(_ context.Context, _, _ string) (runtimeauthorization.SubscriptionInfo, error) {
	return m.info, m.err
}

type mockRatePackageReader struct {
	snap runtimeauthorization.RatePackageSnapshot
	err  error
}

func (m *mockRatePackageReader) ReadRatePackage(_ context.Context, _, _ string) (runtimeauthorization.RatePackageSnapshot, error) {
	return m.snap, m.err
}

type mockCoveredSeqReader struct {
	seq int64
	err error
}

func (m *mockCoveredSeqReader) ReadCoveredSeq(_ context.Context, _, _ string) (int64, error) {
	return m.seq, m.err
}

type mockSnapshotVersion struct {
	next  int64
	calls int
}

func (m *mockSnapshotVersion) Next(_ context.Context) (int64, error) {
	m.calls++
	m.next++
	return m.next, nil
}

type mockSubjectKeyResolver struct {
	keys []string
	err  error
}

func (m *mockSubjectKeyResolver) ResolveSubjectKeys(_ context.Context, _, _ string) ([]string, error) {
	return m.keys, m.err
}

type mockClock struct{ t time.Time }

func (m *mockClock) Now() time.Time { return m.t }

type mockOutboxWriter struct {
	mu     sync.Mutex
	events []aiusage.OutboxEvent
}

func (m *mockOutboxWriter) Append(_ context.Context, events []aiusage.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, events...)
	return nil
}

func (m *mockOutboxWriter) Events() []aiusage.OutboxEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]aiusage.OutboxEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// deterministicSigner builds a reproducible signer for tests.
func deterministicSigner(t *testing.T, keyID string) signing.Signer {
	t.Helper()
	h := sha256.Sum256([]byte(keyID))
	kp := signing.KeyPair{KeyID: keyID, Seed: h[:32]}
	// Pin the signer's clock to the same instant the service mockClock uses
	// (12:00 UTC) so Verify's expiry check sees now < ExpiresAt (12:05)
	// regardless of the real wall clock. Without this, the service builds
	// ExpiresAt from the mock clock while Verify uses the real time.
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	s, err := signing.New(signing.Config{CurrentKey: kp, Now: func() time.Time { return now }})
	require.NoError(t, err)
	return s
}

func newTestService(t *testing.T) (runtimeauthorization.Service, *mockSnapshotVersion) {
	t.Helper()
	clk := &mockClock{t: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)}
	snapVer := &mockSnapshotVersion{}

	svc, err := runtimeauthorization.New(runtimeauthorization.Config{
		BalanceReader: &mockBalanceReader{
			balance: runtimeauthorization.CreditBalance{
				SpendableCredits:           5000,
				EnterpriseAvailableCredits: 10000,
			},
		},
		Subscription: &mockSubscriptionReader{
			info: runtimeauthorization.SubscriptionInfo{
				PlanCode:           "enterprise",
				SubscriptionCode:   "sub-001",
				SubscriptionStatus: "active",
				EntitlementCodes:   []string{"ent-x", "ent-y"},
				CurrentPeriodStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				CurrentPeriodEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		RatePackage: &mockRatePackageReader{
			snap: runtimeauthorization.RatePackageSnapshot{
				Version: "rp-v1",
				Entries: []signing.SignedRateEntry{
					{ResourceCode: "llm.input_tokens", Provider: "openai", Model: "gpt-4", CreditsPerUnit: 7, UnitSize: 1000},
				},
			},
		},
		CoveredSeq:      &mockCoveredSeqReader{seq: 99},
		SnapshotVersion: snapVer,
		Signer:          deterministicSigner(t, "key-ra-test"),
		Clock:           clk,
		Namespace:       "ns-test",
	})
	require.NoError(t, err)
	return svc, snapVer
}

func TestAuthorizationCapacityCredits(t *testing.T) {
	svc, _ := newTestService(t)

	pkg, err := svc.Get(t.Context(), "cust-001", []string{"subj-1"})
	require.NoError(t, err)

	// authorization_capacity_credits = spendable (5000) + enterprise (10000)
	require.Equal(t, int64(15000), pkg.AuthorizationCapacityCredits)
}

func TestEnterprisePackageCoversEverySubject(t *testing.T) {
	svc, _ := newTestService(t)

	subjects := []string{"subj-c", "subj-a", "subj-b", "subj-d"}

	pkg, err := svc.Get(t.Context(), "cust-001", subjects)
	require.NoError(t, err)

	// Every provided subject must appear in the package (canonical sorting
	// happens inside Sign but the logical set must match).
	require.ElementsMatch(t, subjects, pkg.SubjectKeys)

	// The same capacity applies to every subject (it's a customer-level capacity).
	require.True(t, pkg.AuthorizationCapacityCredits > 0)

	// Signature must verify.
	require.NoError(t, deterministicSigner(t, "key-ra-test").Verify(pkg))
}

func TestSnapshotVersionStrictlyIncreases(t *testing.T) {
	svc, snapVer := newTestService(t)

	pkg1, err := svc.Get(t.Context(), "cust-001", []string{"subj-1"})
	require.NoError(t, err)

	pkg2, err := svc.Get(t.Context(), "cust-001", []string{"subj-1"})
	require.NoError(t, err)

	require.Greater(t, pkg2.SnapshotVersion, pkg1.SnapshotVersion,
		"SnapshotVersion must strictly increase")
	require.Equal(t, 2, snapVer.calls, "version provider must be called once per Get")
}

func TestExpiresAtUsesTTL(t *testing.T) {
	svc, _ := newTestService(t)

	pkg, err := svc.Get(t.Context(), "cust-001", []string{"subj-1"})
	require.NoError(t, err)

	// Clock is 12:00:00 UTC, default TTL is 5 minutes.
	require.Equal(t, time.Date(2026, 8, 4, 12, 5, 0, 0, time.UTC), pkg.ExpiresAt)
}

func TestRatePackageIncluded(t *testing.T) {
	svc, _ := newTestService(t)

	pkg, err := svc.Get(t.Context(), "cust-001", []string{"subj-1"})
	require.NoError(t, err)

	require.Equal(t, "rp-v1", pkg.RatePackageVersion)
	require.Len(t, pkg.RatePackage, 1)
	require.Equal(t, "llm.input_tokens", pkg.RatePackage[0].ResourceCode)
}

func TestCoveredTenantSeqIncluded(t *testing.T) {
	svc, _ := newTestService(t)

	pkg, err := svc.Get(t.Context(), "cust-001", []string{"subj-1"})
	require.NoError(t, err)

	require.Equal(t, int64(99), pkg.CoveredTenantSeq)
}

func TestGetPropagatesErrors(t *testing.T) {
	clk := &mockClock{t: time.Now().UTC()}
	injectedErr := errors.New("reader down")

	svc, err := runtimeauthorization.New(runtimeauthorization.Config{
		BalanceReader:   &mockBalanceReader{err: injectedErr},
		Subscription:    &mockSubscriptionReader{},
		RatePackage:     &mockRatePackageReader{},
		CoveredSeq:      &mockCoveredSeqReader{},
		SnapshotVersion: &mockSnapshotVersion{},
		Signer:          deterministicSigner(t, "key-err"),
		Clock:           clk,
		Namespace:       "ns",
	})
	require.NoError(t, err)

	_, err = svc.Get(t.Context(), "cust-001", []string{"subj-1"})
	require.ErrorIs(t, err, injectedErr)
}

func TestEmptyCustomerIDRejected(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Get(t.Context(), "", []string{"subj-1"})
	require.Error(t, err)
}

func TestGetEmitsRuntimeAuthorizationUpdated(t *testing.T) {
	clk := &mockClock{t: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)}
	outbox := &mockOutboxWriter{}
	snapVer := &mockSnapshotVersion{}

	svc, err := runtimeauthorization.New(runtimeauthorization.Config{
		BalanceReader: &mockBalanceReader{
			balance: runtimeauthorization.CreditBalance{
				SpendableCredits:           5000,
				EnterpriseAvailableCredits: 10000,
			},
		},
		Subscription: &mockSubscriptionReader{
			info: runtimeauthorization.SubscriptionInfo{
				PlanCode:           "enterprise",
				SubscriptionCode:   "sub-001",
				SubscriptionStatus: "active",
				EntitlementCodes:   []string{"ent-x"},
			},
		},
		RatePackage:     &mockRatePackageReader{},
		CoveredSeq:      &mockCoveredSeqReader{seq: 1},
		SnapshotVersion: snapVer,
		Signer:          deterministicSigner(t, "key-outbox"),
		Outbox:          outbox,
		Clock:           clk,
		Namespace:       "ns",
	})
	require.NoError(t, err)

	pkg, err := svc.Get(t.Context(), "cust-outbox", []string{"subj-1"})
	require.NoError(t, err)

	events := outbox.Events()
	require.Len(t, events, 1)
	require.Equal(t, aiusage.EventRuntimeAuthorizationUpdated, events[0].EventType)
	require.Equal(t, "cust-outbox", events[0].Payload["billing_customer_id"])
	require.Equal(t, pkg.SnapshotVersion, events[0].Payload["snapshot_version"])
}

func newHandlerTestService(t *testing.T, snapVer *mockSnapshotVersion) (runtimeauthorization.Service, *mockOutboxWriter) {
	t.Helper()
	outbox := &mockOutboxWriter{}
	svc, err := runtimeauthorization.New(runtimeauthorization.Config{
		BalanceReader: &mockBalanceReader{
			balance: runtimeauthorization.CreditBalance{
				SpendableCredits:           5000,
				EnterpriseAvailableCredits: 10000,
			},
		},
		Subscription: &mockSubscriptionReader{
			info: runtimeauthorization.SubscriptionInfo{
				PlanCode:         "enterprise",
				SubscriptionCode: "sub-001",
				EntitlementCodes: []string{"ent-x"},
			},
		},
		RatePackage:     &mockRatePackageReader{},
		CoveredSeq:      &mockCoveredSeqReader{seq: 1},
		SnapshotVersion: snapVer,
		Signer:          deterministicSigner(t, "key-handler"),
		Outbox:          outbox,
		Clock:           &mockClock{t: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)},
		Namespace:       "ns",
	})
	require.NoError(t, err)
	return svc, outbox
}

func TestEventHandlerCreditGrantExpired(t *testing.T) {
	snapVer := &mockSnapshotVersion{}
	svc, outbox := newHandlerTestService(t, snapVer)

	handler, err := runtimeauthorization.NewEventHandler(runtimeauthorization.EventHandlerConfig{
		Service:     svc,
		SubjectKeys: &mockSubjectKeyResolver{keys: []string{"subj-1", "subj-2"}},
		Namespace:   "ns",
	})
	require.NoError(t, err)

	require.NoError(t, handler.Handle(t.Context(), aiusage.EventConsumedCreditGrantExpired, map[string]any{
		"customer_id": "cust-handler",
	}))

	// A new signed package was generated and an outbox event emitted.
	require.Equal(t, 1, snapVer.calls, "snapshot version provider called once")
	events := outbox.Events()
	require.Len(t, events, 1)
	require.Equal(t, aiusage.EventRuntimeAuthorizationUpdated, events[0].EventType)
}

func TestEventHandlerSubscriptionUpdated(t *testing.T) {
	snapVer := &mockSnapshotVersion{}
	svc, outbox := newHandlerTestService(t, snapVer)

	handler, err := runtimeauthorization.NewEventHandler(runtimeauthorization.EventHandlerConfig{
		Service:     svc,
		SubjectKeys: &mockSubjectKeyResolver{keys: []string{"subj-3"}},
		Namespace:   "ns",
	})
	require.NoError(t, err)

	require.NoError(t, handler.Handle(t.Context(), aiusage.EventConsumedSubscriptionUpdated, map[string]any{
		"customer_id": "cust-sub-update",
	}))

	events := outbox.Events()
	require.Len(t, events, 1)
	require.Equal(t, aiusage.EventRuntimeAuthorizationUpdated, events[0].EventType)
	require.Equal(t, "cust-sub-update", events[0].Payload["billing_customer_id"])
}

func TestEventHandlerIncrementsSnapshotVersion(t *testing.T) {
	snapVer := &mockSnapshotVersion{}
	svc, _ := newHandlerTestService(t, snapVer)

	handler, err := runtimeauthorization.NewEventHandler(runtimeauthorization.EventHandlerConfig{
		Service:     svc,
		SubjectKeys: &mockSubjectKeyResolver{keys: []string{"subj-1"}},
		Namespace:   "ns",
	})
	require.NoError(t, err)

	// First event generates version 1.
	require.NoError(t, handler.Handle(t.Context(), aiusage.EventConsumedCreditGrantExpired, map[string]any{
		"customer_id": "cust-inc",
	}))

	// Second event generates version 2 (strictly increasing).
	require.NoError(t, handler.Handle(t.Context(), aiusage.EventConsumedSubscriptionUpdated, map[string]any{
		"customer_id": "cust-inc",
	}))

	require.Equal(t, 2, snapVer.calls)
}

func TestEventHandlerIgnoresUnknownEventTypes(t *testing.T) {
	snapVer := &mockSnapshotVersion{}
	svc, outbox := newHandlerTestService(t, snapVer)

	handler, err := runtimeauthorization.NewEventHandler(runtimeauthorization.EventHandlerConfig{
		Service:     svc,
		SubjectKeys: &mockSubjectKeyResolver{keys: []string{"subj-1"}},
		Namespace:   "ns",
	})
	require.NoError(t, err)

	require.NoError(t, handler.Handle(t.Context(), "some.other.event", map[string]any{
		"customer_id": "cust-ignore",
	}))

	require.Zero(t, snapVer.calls, "unknown events must not trigger regeneration")
	require.Empty(t, outbox.Events())
}

func TestEventHandlerMissingCustomerID(t *testing.T) {
	svc, _ := newHandlerTestService(t, &mockSnapshotVersion{})

	handler, err := runtimeauthorization.NewEventHandler(runtimeauthorization.EventHandlerConfig{
		Service:     svc,
		SubjectKeys: &mockSubjectKeyResolver{keys: []string{"subj-1"}},
		Namespace:   "ns",
	})
	require.NoError(t, err)

	err = handler.Handle(t.Context(), aiusage.EventConsumedCreditGrantExpired, map[string]any{})
	require.Error(t, err)
}
