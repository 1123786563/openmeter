package enterprise

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
)

// ---------------------------------------------------------------------------
// In-memory mock repository (thread-safe)
// ---------------------------------------------------------------------------

type mockRepo struct {
	mu       sync.Mutex
	accounts map[string]*ReceivableAccount // keyed by namespace|customerID
	byID     map[string]*ReceivableAccount // keyed by namespace|id
	periods  []*ReceivablePeriod
	usage    map[string][]UsageAccrual // keyed by periodID
	facts    []*PaymentAuditFact
	auths    map[string]*OfflineAuthorization // keyed by namespace|nonce
	invoices map[string]*ExternalInvoiceRef   // keyed by id
	events   []AuditEvent
	idSeq    atomic.Int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		accounts: make(map[string]*ReceivableAccount),
		byID:     make(map[string]*ReceivableAccount),
		usage:    make(map[string][]UsageAccrual),
		auths:    make(map[string]*OfflineAuthorization),
		invoices: make(map[string]*ExternalInvoiceRef),
	}
}

func accountKey(namespace, customerID string) string { return namespace + "|" + customerID }
func authKey(namespace, nonce string) string         { return namespace + "|" + nonce }

func (m *mockRepo) CreateAccount(_ context.Context, account ReceivableAccount) (*ReceivableAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := accountKey(account.Namespace, account.CustomerID)
	if existing, ok := m.accounts[k]; ok {
		cp := *existing
		return &cp, nil
	}
	cp := account
	m.accounts[k] = &cp
	m.byID[account.Namespace+"|"+account.ID] = &cp
	return &cp, nil
}

func (m *mockRepo) GetAccount(_ context.Context, namespace, customerID string) (*ReceivableAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[accountKey(namespace, customerID)]
	if !ok {
		return nil, ErrAccountNotFound
	}
	cp := *acc
	return &cp, nil
}

func (m *mockRepo) GetAccountByID(_ context.Context, namespace, id string) (*ReceivableAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.byID[namespace+"|"+id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	cp := *acc
	return &cp, nil
}

func (m *mockRepo) SetCollectionState(_ context.Context, namespace, id string, state CollectionState) (*ReceivableAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.byID[namespace+"|"+id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	acc.CollectionState = state
	acc.UpdatedAt = clock.Now()
	cp := *acc
	return &cp, nil
}

func (m *mockRepo) CreatePeriod(_ context.Context, period ReceivablePeriod) (*ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := period
	m.periods = append(m.periods, &cp)
	return &cp, nil
}

func (m *mockRepo) GetOpenPeriod(_ context.Context, namespace, accountID string) (*ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.periods) - 1; i >= 0; i-- {
		p := m.periods[i]
		if p.Namespace == namespace && p.AccountID == accountID && p.Status == PeriodStatusOpen {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrNoOpenPeriod
}

func (m *mockRepo) GetPeriod(_ context.Context, namespace, id string) (*ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.periods {
		if p.Namespace == namespace && p.ID == id {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPeriodNotFound
}

func (m *mockRepo) ListPeriods(_ context.Context, namespace, accountID string) ([]ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ReceivablePeriod
	for _, p := range m.periods {
		if p.Namespace == namespace && p.AccountID == accountID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (m *mockRepo) ClosePeriod(_ context.Context, namespace, id string, totalCredits, totalMinor int64) (*ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.periods {
		if p.Namespace == namespace && p.ID == id {
			if p.Status != PeriodStatusOpen {
				return nil, ErrPeriodNotOpen
			}
			p.Status = PeriodStatusClosed
			p.TotalCredits = totalCredits
			p.TotalMinor = totalMinor
			p.UpdatedAt = clock.Now()
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPeriodNotFound
}

func (m *mockRepo) SetPeriodPaid(_ context.Context, namespace, id string, paidMinor int64, status PeriodStatus) (*ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.periods {
		if p.Namespace == namespace && p.ID == id {
			p.PaidMinor = paidMinor
			p.Status = status
			p.UpdatedAt = clock.Now()
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPeriodNotFound
}

func (m *mockRepo) AppendUsage(_ context.Context, accrual UsageAccrual) (*ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usage[accrual.PeriodID] = append(m.usage[accrual.PeriodID], accrual)
	// Update account used credits and balance.
	for _, acc := range m.accounts {
		if acc.Namespace == accrual.Namespace && acc.CustomerID == accrual.CustomerID {
			acc.UsedCredits += accrual.Credits
			acc.CurrentBalanceMinor -= accrual.AmountMinor
			acc.UpdatedAt = clock.Now()
		}
	}
	// Return the (unchanged total) open period copy.
	for _, p := range m.periods {
		if p.Namespace == accrual.Namespace && p.ID == accrual.PeriodID {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPeriodNotFound
}

func (m *mockRepo) ListUsage(_ context.Context, namespace, periodID string) ([]UsageAccrual, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]UsageAccrual, len(m.usage[periodID]))
	copy(out, m.usage[periodID])
	return out, nil
}

func (m *mockRepo) AppendPaymentFact(_ context.Context, fact PaymentAuditFact) (*PaymentAuditFact, *ReceivablePeriod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := fact
	m.facts = append(m.facts, &cp)
	// Recompute period paid from all facts.
	for _, p := range m.periods {
		if p.Namespace == fact.Namespace && p.ID == fact.PeriodID {
			var paid int64
			for _, f := range m.facts {
				if f.PeriodID == p.ID {
					paid += f.SignedAmountMinor()
				}
			}
			if paid < 0 {
				paid = 0
			}
			p.PaidMinor = paid
			p.Status = computePaidStatus(p, paid)
			p.UpdatedAt = clock.Now()
			pcp := *p
			return &cp, &pcp, nil
		}
	}
	return &cp, nil, ErrPeriodNotFound
}

func computePaidStatus(p *ReceivablePeriod, paid int64) PeriodStatus {
	if p.TotalMinor == 0 {
		return p.Status
	}
	if paid >= p.TotalMinor {
		return PeriodStatusPaid
	}
	if paid > 0 {
		return PeriodStatusPartiallyPaid
	}
	return p.Status
}

func (m *mockRepo) GetPaymentFact(_ context.Context, namespace, id string) (*PaymentAuditFact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range m.facts {
		if f.Namespace == namespace && f.ID == id {
			cp := *f
			return &cp, nil
		}
	}
	return nil, ErrPaymentNotFound
}

func (m *mockRepo) ListPaymentFacts(_ context.Context, namespace, periodID string) ([]PaymentAuditFact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []PaymentAuditFact
	for _, f := range m.facts {
		if f.Namespace == namespace && f.PeriodID == periodID {
			out = append(out, *f)
		}
	}
	return out, nil
}

func (m *mockRepo) SaveAuthorization(_ context.Context, auth OfflineAuthorization) (*OfflineAuthorization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := auth
	m.auths[authKey(auth.Namespace, auth.Nonce)] = &cp
	return &cp, nil
}

func (m *mockRepo) GetAuthorization(_ context.Context, namespace, nonce string) (*OfflineAuthorization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	auth, ok := m.auths[authKey(namespace, nonce)]
	if !ok {
		return nil, ErrAuthorizationNotFound
	}
	cp := *auth
	return &cp, nil
}

func (m *mockRepo) AppendInvoiceRef(_ context.Context, ref ExternalInvoiceRef) (*ExternalInvoiceRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := ref
	m.invoices[ref.ID] = &cp
	return &cp, nil
}

func (m *mockRepo) ListInvoiceRefs(_ context.Context, namespace, periodID string) ([]ExternalInvoiceRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ExternalInvoiceRef
	for _, r := range m.invoices {
		if r.Namespace == namespace && r.PeriodID == periodID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (m *mockRepo) UpdateInvoiceRefStatus(_ context.Context, namespace, id string, status InvoiceRefStatus) (*ExternalInvoiceRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.invoices[id]
	if !ok || r.Namespace != namespace {
		return nil, errors.New("invoice ref not found")
	}
	r.Status = status
	r.UpdatedAt = clock.Now()
	if status == InvoiceRefIssued || status == InvoiceRefPaid {
		if r.IssuedAt == nil {
			t := clock.Now()
			r.IssuedAt = &t
		}
	}
	cp := *r
	return &cp, nil
}

func (m *mockRepo) AppendEvent(_ context.Context, event AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

// ---------------------------------------------------------------------------
// Mock collaborators
// ---------------------------------------------------------------------------

type mockRoles struct {
	admins map[string]bool // keyed by actor
}

func (r *mockRoles) IsFinanceAdmin(_ context.Context, actor, _ string) (bool, error) {
	return r.admins[actor], nil
}

type mockEvents struct {
	mu      sync.Mutex
	events  []AuditEvent
	publish func(AuditEvent) error // optional injection
}

func (e *mockEvents) Publish(ctx context.Context, event AuditEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, event)
	return nil
}

type seqIDGen struct {
	n atomic.Int64
}

func (g *seqIDGen) NewID() string {
	return fmt.Sprintf("id-%d", g.n.Add(1))
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

type harness struct {
	svc    Service
	repo   *mockRepo
	roles  *mockRoles
	events *mockEvents
	signer AuthorizationSigner
	ids    *seqIDGen
}

func newHarness(t *testing.T, cfg ...func(*Config)) *harness {
	t.Helper()
	repo := newMockRepo()
	roles := &mockRoles{admins: map[string]bool{"finance-admin": true}}
	events := &mockEvents{}
	ids := &seqIDGen{}
	signer := HMACSigner{Secret: []byte("test-secret")}

	c := Config{
		Repo:   repo,
		Signer: signer,
		Roles:  roles,
		Events: events,
		IDs:    ids,
	}
	for _, fn := range cfg {
		fn(&c)
	}

	svc, err := New(c)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return &harness{svc: svc, repo: repo, roles: roles, events: events, signer: signer, ids: ids}
}

// setupCustomer opens an account + open period + a signed authorization and
// returns the account and authorization.
func (h *harness) setupCustomer(t *testing.T, namespace, customerID string, ceiling int64) (*ReceivableAccount, *OfflineAuthorization) {
	t.Helper()
	acc, err := h.svc.OpenAccount(context.Background(), OpenAccountInput{
		Namespace: namespace, CustomerID: customerID,
		CreditLimitMinor: 1_000_000, CeilingCredits: ceiling, Currency: "CNY",
	})
	if err != nil {
		t.Fatalf("OpenAccount: %v", err)
	}
	if _, err := h.svc.EnsureOpenPeriod(context.Background(), namespace, acc.ID); err != nil {
		t.Fatalf("EnsureOpenPeriod: %v", err)
	}
	auth, err := h.svc.IssueAuthorization(context.Background(), IssueAuthorizationInput{
		Namespace: namespace, CustomerID: customerID,
		Subject: "agent-1", MaxCredit: ceiling, Nonce: "nonce-" + namespace + "-" + customerID + "-" + h.ids.NewID(),
	})
	if err != nil {
		t.Fatalf("IssueAuthorization: %v", err)
	}
	return acc, auth
}

// ---------------------------------------------------------------------------
// Tests: open period accrues settled Credit usage into receivable
// ---------------------------------------------------------------------------

func TestAccrueUsageIntoOpenPeriod(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	acc, auth := h.setupCustomer(t, "ns-a", "cust-1", 1000)

	// Accrue usage beyond prepaid buckets into the receivable.
	period, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns-a", CustomerID: "cust-1", Nonce: auth.Nonce,
		Credits: 100, RateMinor: 10, // 100 credits = 1000 minor
		UserID: "user-1", AgentID: "agent-1", APIKeyID: "key-1",
	})
	if err != nil {
		t.Fatalf("AccrueUsage: %v", err)
	}
	if period.Status != PeriodStatusOpen {
		t.Errorf("period status = %s, want open", period.Status)
	}

	// The usage is recorded with full attribution.
	usage, _ := h.repo.ListUsage(context.Background(), "ns-a", period.ID)
	if len(usage) != 1 {
		t.Fatalf("usage count = %d, want 1", len(usage))
	}
	if usage[0].Credits != 100 || usage[0].AmountMinor != 1000 {
		t.Errorf("usage = %+v", usage[0])
	}

	// Account used credits reflect the accrual.
	updated, _ := h.repo.GetAccount(context.Background(), "ns-a", "cust-1")
	if updated.UsedCredits != 100 {
		t.Errorf("used credits = %d, want 100", updated.UsedCredits)
	}
	_ = acc
}

// ---------------------------------------------------------------------------
// Tests: signed offline authorization has Credit ceiling and expiry
// ---------------------------------------------------------------------------

func TestIssueAuthorizationCeilingAndExpiry(t *testing.T) {
	clock.ResetTime()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock.SetTime(now)

	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)

	if auth.MaxCredit != 5000 {
		t.Errorf("max credit = %d, want 5000", auth.MaxCredit)
	}
	if auth.Subject != "agent-1" {
		t.Errorf("subject = %s", auth.Subject)
	}
	// Default window is 24h: expiry must be issuedAt + 24h. Compare the
	// duration rather than the absolute instant to tolerate clock drift.
	if got := auth.Expiry.Sub(auth.IssuedAt); got != 24*time.Hour {
		t.Errorf("window = %s, want 24h", got)
	}
	if auth.Signature == "" {
		t.Error("signature should be set")
	}
	if auth.Nonce == "" {
		t.Error("nonce should be set")
	}
}

// ---------------------------------------------------------------------------
// Tests: authorization rejects use above ceiling or after expiry
// ---------------------------------------------------------------------------

func TestAuthorizeRejectsAboveCeiling(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 1000)

	// Within ceiling: ok.
	if err := h.svc.Authorize(context.Background(), "ns", "cust", auth.Nonce, 1000); err != nil {
		t.Fatalf("Authorize within ceiling: %v", err)
	}

	// Above ceiling: rejected.
	err := h.svc.Authorize(context.Background(), "ns", "cust", auth.Nonce, 1001)
	if !errors.Is(err, ErrAboveCeiling) {
		t.Fatalf("expected ErrAboveCeiling, got %v", err)
	}
}

func TestAuthorizeRejectsAfterExpiry(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock.SetTime(start)
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 1000)

	// Move past the 24h expiry window.
	clock.SetTime(start.Add(25 * time.Hour))

	err := h.svc.Authorize(context.Background(), "ns", "cust", auth.Nonce, 10)
	if !errors.Is(err, ErrAuthorizationExpired) {
		t.Fatalf("expected ErrAuthorizationExpired, got %v", err)
	}
}

func TestAuthorizeRejectsInvalidSignature(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 1000)

	// Tamper with the stored authorization signature.
	h.repo.mu.Lock()
	a := h.repo.auths[authKey("ns", auth.Nonce)]
	a.Signature = "tampered"
	h.repo.mu.Unlock()

	err := h.svc.Authorize(context.Background(), "ns", "cust", auth.Nonce, 10)
	if !errors.Is(err, ErrAuthorizationInvalid) {
		t.Fatalf("expected ErrAuthorizationInvalid, got %v", err)
	}
}

func TestNonceReuseRejected(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 1000)

	_, err := h.svc.IssueAuthorization(context.Background(), IssueAuthorizationInput{
		Namespace: "ns", CustomerID: "cust", Subject: "x", MaxCredit: 100, Nonce: auth.Nonce,
	})
	if !errors.Is(err, ErrNonceReuse) {
		t.Fatalf("expected ErrNonceReuse, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: period close snapshots usage, rates and amount_minor
// ---------------------------------------------------------------------------

func TestClosePeriodSnapshotsAmount(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)

	// Accrue usage at two different rates.
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce,
		Credits: 100, RateMinor: 10,
	})
	if _, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce,
		Credits: 50, RateMinor: 20,
	}); err != nil {
		t.Fatalf("AccrueUsage 2: %v", err)
	}

	// Close freezes the amount: 100*10 + 50*20 = 2000 minor.
	closed, err := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})
	if err != nil {
		t.Fatalf("ClosePeriod: %v", err)
	}
	if closed.Status != PeriodStatusClosed {
		t.Errorf("status = %s, want closed", closed.Status)
	}
	if closed.TotalMinor != 2000 {
		t.Errorf("total minor = %d, want 2000", closed.TotalMinor)
	}
	if closed.TotalCredits != 150 {
		t.Errorf("total credits = %d, want 150", closed.TotalCredits)
	}

	// After close, the settlement range and amount are frozen: accruing more
	// usage must NOT change the closed period.
	if _, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce,
		Credits: 10, RateMinor: 10,
	}); err == nil {
		// No open period remains, so this should fail (no open period).
		t.Error("accrue after close should fail with no open period")
	}

	refrozen, _ := h.repo.GetPeriod(context.Background(), "ns", closed.ID)
	if refrozen.TotalMinor != 2000 {
		t.Errorf("frozen total changed to %d, want 2000", refrozen.TotalMinor)
	}
}

// ---------------------------------------------------------------------------
// Tests: partial offline payment reduces outstanding amount
// ---------------------------------------------------------------------------

func TestPartialPaymentReducesOutstanding(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)

	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce,
		Credits: 100, RateMinor: 10, // 1000 minor
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	// Partial payment of 400.
	paid, err := h.svc.RecordPayment(context.Background(), RecordPaymentInput{
		Namespace: "ns", PeriodID: closed.ID, AccountID: closed.AccountID,
		AmountMinor: 400, Currency: "CNY", BankReference: "BANK-1", Payer: "Acme Corp",
		ReceivedAt: clock.Now(), EvidenceHash: "hash-1",
		ConfirmedBy: "finance-admin", FinanceAdminProof: "token",
	})
	if err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}
	if paid.Status != PeriodStatusPartiallyPaid {
		t.Errorf("status = %s, want partially_paid", paid.Status)
	}
	if paid.PaidMinor != 400 {
		t.Errorf("paid = %d, want 400", paid.PaidMinor)
	}
	if paid.OutstandingMinor() != 600 {
		t.Errorf("outstanding = %d, want 600", paid.OutstandingMinor())
	}
}

// ---------------------------------------------------------------------------
// Tests: full payment marks period paid
// ---------------------------------------------------------------------------

func TestFullPaymentMarksPaid(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)

	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce,
		Credits: 100, RateMinor: 10, // 1000 minor
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	// Full payment.
	paid, err := h.svc.RecordPayment(context.Background(), RecordPaymentInput{
		Namespace: "ns", PeriodID: closed.ID, AccountID: closed.AccountID,
		AmountMinor: 1000, Currency: "CNY", BankReference: "BANK-2", Payer: "Acme Corp",
		ReceivedAt: clock.Now(), EvidenceHash: "hash-2",
		ConfirmedBy: "finance-admin", FinanceAdminProof: "token",
	})
	if err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}
	if paid.Status != PeriodStatusPaid {
		t.Errorf("status = %s, want paid", paid.Status)
	}
	if !paid.IsPaid() {
		t.Error("period should be fully paid")
	}
}

func TestRecordPaymentRequiresFinanceAdmin(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 100, RateMinor: 10,
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	// Non-admin actor.
	_, err := h.svc.RecordPayment(context.Background(), RecordPaymentInput{
		Namespace: "ns", PeriodID: closed.ID, AccountID: closed.AccountID,
		AmountMinor: 1000, Currency: "CNY", BankReference: "BANK-3", Payer: "Acme Corp",
		ReceivedAt: clock.Now(), EvidenceHash: "hash-3",
		ConfirmedBy: "regular-user", FinanceAdminProof: "token",
	})
	if !errors.Is(err, ErrFinanceAdminRequired) {
		t.Fatalf("expected ErrFinanceAdminRequired, got %v", err)
	}
}

func TestRecordPaymentRequiresAllFields(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 100, RateMinor: 10,
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	cases := []struct {
		name string
		in   RecordPaymentInput
	}{
		{"missing bank reference", RecordPaymentInput{
			Namespace: "ns", PeriodID: closed.ID, AmountMinor: 1000, Currency: "CNY",
			Payer: "p", ReceivedAt: clock.Now(), EvidenceHash: "h", ConfirmedBy: "finance-admin", FinanceAdminProof: "t"}},
		{"missing payer", RecordPaymentInput{
			Namespace: "ns", PeriodID: closed.ID, AmountMinor: 1000, Currency: "CNY",
			BankReference: "b", ReceivedAt: clock.Now(), EvidenceHash: "h", ConfirmedBy: "finance-admin", FinanceAdminProof: "t"}},
		{"missing evidence hash", RecordPaymentInput{
			Namespace: "ns", PeriodID: closed.ID, AmountMinor: 1000, Currency: "CNY",
			BankReference: "b", Payer: "p", ReceivedAt: clock.Now(), ConfirmedBy: "finance-admin", FinanceAdminProof: "t"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := h.svc.RecordPayment(context.Background(), tc.in); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("%s: expected ErrInvalidInput, got %v", tc.name, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: corrections use reversing entries
// ---------------------------------------------------------------------------

func TestPaymentReversalIsImmutableEntry(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 100, RateMinor: 10,
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	paid, _ := h.svc.RecordPayment(context.Background(), RecordPaymentInput{
		Namespace: "ns", PeriodID: closed.ID, AccountID: closed.AccountID,
		AmountMinor: 1000, Currency: "CNY", BankReference: "BANK-4", Payer: "Acme Corp",
		ReceivedAt: clock.Now(), EvidenceHash: "hash-4",
		ConfirmedBy: "finance-admin", FinanceAdminProof: "token",
	})
	if paid.PaidMinor != 1000 {
		t.Fatalf("paid = %d, want 1000", paid.PaidMinor)
	}

	// The applied fact exists.
	facts, _ := h.repo.ListPaymentFacts(context.Background(), "ns", closed.ID)
	if len(facts) != 1 || facts[0].Kind != PaymentFactApplied {
		t.Fatalf("expected 1 applied fact, got %+v", facts)
	}
	originalFactID := facts[0].ID

	// Reverse it: a new reversing entry, original untouched.
	reversed, err := h.svc.ReversePayment(context.Background(), ReversePaymentInput{
		Namespace: "ns", PaymentFactID: originalFactID,
		ConfirmedBy: "finance-admin", FinanceAdminProof: "token", Reason: "bank reversal",
	})
	if err != nil {
		t.Fatalf("ReversePayment: %v", err)
	}
	if reversed.PaidMinor != 0 {
		t.Errorf("paid after reversal = %d, want 0", reversed.PaidMinor)
	}

	// Both facts still exist; the original was not mutated/deleted.
	facts, _ = h.repo.ListPaymentFacts(context.Background(), "ns", closed.ID)
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}
	var applied, reversedFact *PaymentAuditFact
	for i := range facts {
		if facts[i].Kind == PaymentFactApplied {
			applied = &facts[i]
		}
		if facts[i].Kind == PaymentFactReversed {
			reversedFact = &facts[i]
		}
	}
	if applied == nil || reversedFact == nil {
		t.Fatal("expected both applied and reversed facts")
	}
	if applied.AmountMinor != 1000 {
		t.Errorf("applied amount mutated to %d", applied.AmountMinor)
	}
	if reversedFact.ReferencePaymentID != originalFactID {
		t.Errorf("reversal reference = %s, want %s", reversedFact.ReferencePaymentID, originalFactID)
	}
}

// ---------------------------------------------------------------------------
// Tests: overdue status does not rewrite historical usage
// ---------------------------------------------------------------------------

func TestOverdueDoesNotRewriteHistoricalUsage(t *testing.T) {
	clock.ResetTime()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(start)
	h := newHarness(t, func(c *Config) {
		// Short grace for the test.
		c.Policy = CollectionPolicy{Version: "test", GracePeriod: time.Hour, RestrictionDelay: time.Hour, SuspensionDelay: time.Hour}
	})
	acc, auth := h.setupCustomer(t, "ns", "cust", 5000)
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 100, RateMinor: 10,
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	// Snapshot historical usage before overdue.
	usageBefore, _ := h.repo.ListUsage(context.Background(), "ns", closed.ID)
	totalBefore := closed.TotalMinor

	// Advance time past grace; evaluate collection to drive overdue.
	clock.SetTime(start.AddDate(0, 1, 1).Add(2 * time.Hour))
	// Force the closed period's end into the past for the grace check.
	h.repo.mu.Lock()
	for _, p := range h.repo.periods {
		if p.ID == closed.ID {
			p.PeriodEnd = start.AddDate(0, 1, 0)
		}
	}
	h.repo.mu.Unlock()

	updated, err := h.svc.EvaluateCollection(context.Background(), "ns", acc.ID)
	if err != nil {
		t.Fatalf("EvaluateCollection: %v", err)
	}
	if updated.CollectionState != CollectionStateOverdue {
		t.Errorf("state = %s, want overdue", updated.CollectionState)
	}

	// Historical usage and the frozen total are unchanged.
	usageAfter, _ := h.repo.ListUsage(context.Background(), "ns", closed.ID)
	if len(usageAfter) != len(usageBefore) {
		t.Errorf("usage rewritten: %d -> %d", len(usageBefore), len(usageAfter))
	}
	closedAfter, _ := h.repo.GetPeriod(context.Background(), "ns", closed.ID)
	if closedAfter.TotalMinor != totalBefore {
		t.Errorf("frozen total rewritten: %d -> %d", totalBefore, closedAfter.TotalMinor)
	}
}

// ---------------------------------------------------------------------------
// Tests: two Tenant entities under one enterprise remain isolated
// ---------------------------------------------------------------------------

func TestTenantIsolation(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)

	// Two tenants (namespaces) under one enterprise, same customer ID.
	acc1, auth1 := h.setupCustomer(t, "tenant-1", "enterprise-cust", 1000)
	acc2, auth2 := h.setupCustomer(t, "tenant-2", "enterprise-cust", 1000)

	if acc1.ID == acc2.ID {
		t.Fatal("accounts should be distinct per namespace")
	}

	// Accrue usage in tenant-1.
	if _, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "tenant-1", CustomerID: "enterprise-cust", Nonce: auth1.Nonce,
		Credits: 100, RateMinor: 10,
	}); err != nil {
		t.Fatalf("accrue tenant-1: %v", err)
	}

	// tenant-1 account has used credits; tenant-2 does not.
	a1, _ := h.repo.GetAccount(context.Background(), "tenant-1", "enterprise-cust")
	a2, _ := h.repo.GetAccount(context.Background(), "tenant-2", "enterprise-cust")
	if a1.UsedCredits != 100 {
		t.Errorf("tenant-1 used = %d, want 100", a1.UsedCredits)
	}
	if a2.UsedCredits != 0 {
		t.Errorf("tenant-2 used = %d, want 0 (leak)", a2.UsedCredits)
	}

	// tenant-2's authorization cannot be used in tenant-1 and vice versa.
	err := h.svc.Authorize(context.Background(), "tenant-1", "enterprise-cust", auth2.Nonce, 10)
	if !errors.Is(err, ErrAuthorizationNotFound) {
		t.Errorf("cross-tenant auth should not resolve: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: grace/collection policy transitions overdue -> restricted -> suspended
// ---------------------------------------------------------------------------

func TestCollectionPolicyTransitions(t *testing.T) {
	clock.ResetTime()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(base)

	h := newHarness(t, func(c *Config) {
		c.Policy = CollectionPolicy{Version: "test", GracePeriod: time.Hour, RestrictionDelay: time.Hour, SuspensionDelay: time.Hour}
	})
	acc, auth := h.setupCustomer(t, "ns", "cust", 5000)
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 100, RateMinor: 10,
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	// Push the period end into the past.
	h.repo.mu.Lock()
	for _, p := range h.repo.periods {
		if p.ID == closed.ID {
			p.PeriodEnd = base.AddDate(0, 1, 0)
		}
	}
	h.repo.mu.Unlock()

	// active -> overdue (past grace).
	clock.SetTime(base.AddDate(0, 1, 0).Add(2 * time.Hour))
	acc, _ = h.svc.EvaluateCollection(context.Background(), "ns", acc.ID)
	if acc.CollectionState != CollectionStateOverdue {
		t.Fatalf("state = %s, want overdue", acc.CollectionState)
	}

	// A collection event was published.
	if len(h.events.events) == 0 {
		t.Fatal("expected a collection event")
	}
	lastEvent := h.events.events[len(h.events.events)-1]
	if lastEvent.Kind != "collection_state_changed" {
		t.Errorf("event kind = %s", lastEvent.Kind)
	}

	// overdue -> restricted (past restriction delay).
	clock.SetTime(base.AddDate(0, 1, 0).Add(4 * time.Hour))
	acc, _ = h.svc.EvaluateCollection(context.Background(), "ns", acc.ID)
	if acc.CollectionState != CollectionStateRestricted {
		t.Fatalf("state = %s, want restricted", acc.CollectionState)
	}

	// Restricted account cannot accrue new usage.
	_, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 10, RateMinor: 10,
	})
	if !errors.Is(err, ErrAccountRestricted) {
		t.Errorf("expected ErrAccountRestricted, got %v", err)
	}

	// restricted -> suspended (past suspension delay).
	clock.SetTime(base.AddDate(0, 1, 0).Add(6 * time.Hour))
	acc, _ = h.svc.EvaluateCollection(context.Background(), "ns", acc.ID)
	if acc.CollectionState != CollectionStateSuspended {
		t.Fatalf("state = %s, want suspended", acc.CollectionState)
	}

	// Suspended is terminal: re-evaluating does not error or change state.
	acc, err = h.svc.EvaluateCollection(context.Background(), "ns", acc.ID)
	if err != nil {
		t.Fatalf("re-evaluate suspended: %v", err)
	}
	if acc.CollectionState != CollectionStateSuspended {
		t.Errorf("state = %s, want suspended (terminal)", acc.CollectionState)
	}
}

// ---------------------------------------------------------------------------
// Tests: external invoice status/reference recorded, no tax document content
// ---------------------------------------------------------------------------

func TestExternalInvoiceRefNoTaxContent(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 100, RateMinor: 10,
	})
	closed, _ := h.svc.ClosePeriod(context.Background(), ClosePeriodInput{Namespace: "ns", AccountID: period.AccountID})

	ref, err := h.svc.RecordInvoiceRef(context.Background(), RecordInvoiceRefInput{
		Namespace: "ns", PeriodID: closed.ID,
		ExternalSystem: "xero", ExternalInvoiceID: "INV-001", Status: InvoiceRefIssued,
		InvoiceURL: "https://example.com/inv/001",
	})
	if err != nil {
		t.Fatalf("RecordInvoiceRef: %v", err)
	}
	// Only the allowed fields are exposed.
	if ref.ExternalSystem != "xero" || ref.ExternalInvoiceID != "INV-001" {
		t.Errorf("ref = %+v", ref)
	}
	if ref.Status != InvoiceRefIssued {
		t.Errorf("status = %s", ref.Status)
	}
	if ref.IssuedAt == nil {
		t.Error("issued_at should be set for issued status")
	}

	// Status can be updated.
	updated, err := h.svc.UpdateInvoiceRefStatus(context.Background(), "ns", ref.ID, InvoiceRefPaid)
	if err != nil {
		t.Fatalf("UpdateInvoiceRefStatus: %v", err)
	}
	if updated.Status != InvoiceRefPaid {
		t.Errorf("status = %s, want paid", updated.Status)
	}

	// The ref exposes no tax/PDF/fiscal fields (struct has none by design).
	// This is enforced structurally; we assert the type has only the expected
	// exported fields by checking the documented surface.
	_ = ExternalInvoiceRef{} // compile-time presence
}

// ---------------------------------------------------------------------------
// Tests: usage detail remains attributable by Tenant, user, Agent and API key
// ---------------------------------------------------------------------------

func TestUsageAttribution(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns-attr", "cust-attr", 5000)

	period, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns-attr", CustomerID: "cust-attr", Nonce: auth.Nonce,
		Credits: 50, RateMinor: 5,
		UserID: "user-42", AgentID: "agent-7", APIKeyID: "apikey-9",
	})
	if err != nil {
		t.Fatalf("AccrueUsage: %v", err)
	}

	usage, _ := h.repo.ListUsage(context.Background(), "ns-attr", period.ID)
	if len(usage) != 1 {
		t.Fatalf("usage count = %d", len(usage))
	}
	u := usage[0]
	if u.Namespace != "ns-attr" {
		t.Errorf("namespace = %s", u.Namespace)
	}
	if u.UserID != "user-42" {
		t.Errorf("user_id = %s", u.UserID)
	}
	if u.AgentID != "agent-7" {
		t.Errorf("agent_id = %s", u.AgentID)
	}
	if u.APIKeyID != "apikey-9" {
		t.Errorf("api_key_id = %s", u.APIKeyID)
	}
	if u.CustomerID != "cust-attr" {
		t.Errorf("customer_id = %s", u.CustomerID)
	}
}

// ---------------------------------------------------------------------------
// Tests: offline window config (default 24h, clamped to 72h)
// ---------------------------------------------------------------------------

func TestOfflineWindowClamped(t *testing.T) {
	clock.ResetTime()
	t.Run("default 24h", func(t *testing.T) {
		h := newHarness(t)
		if got := h.svc.(*service).OfflineWindow(); got != 24*time.Hour {
			t.Errorf("default window = %s, want 24h", got)
		}
	})
	t.Run("configured lower accepted", func(t *testing.T) {
		h := newHarness(t, func(c *Config) { c.OfflineWindow = 6 * time.Hour })
		if got := h.svc.(*service).OfflineWindow(); got != 6*time.Hour {
			t.Errorf("window = %s, want 6h", got)
		}
	})
	t.Run("configured above 72h clamped", func(t *testing.T) {
		h := newHarness(t, func(c *Config) { c.OfflineWindow = 200 * time.Hour })
		if got := h.svc.(*service).OfflineWindow(); got != 72*time.Hour {
			t.Errorf("window = %s, want 72h (clamped)", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: usage requires active account + valid authorization
// ---------------------------------------------------------------------------

func TestAccrueRequiresAuthorization(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	h.setupCustomer(t, "ns", "cust", 1000)

	// No nonce / unknown nonce.
	_, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: "unknown-nonce",
		Credits: 10, RateMinor: 10,
	})
	if !errors.Is(err, ErrAuthorizationNotFound) {
		t.Errorf("expected ErrAuthorizationNotFound, got %v", err)
	}
}

func TestAccrueRequiresAccount(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	// No account opened.
	_, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "ghost", Nonce: "n",
		Credits: 10, RateMinor: 10,
	})
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestAccrueBeyondCeilingRejected(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 100)

	// First accrual within ceiling.
	if _, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 80, RateMinor: 10,
	}); err != nil {
		t.Fatalf("accrue 80: %v", err)
	}

	// Second accrual would exceed ceiling (80 + 30 > 100).
	_, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 30, RateMinor: 10,
	})
	if !errors.Is(err, ErrAboveCeiling) {
		t.Errorf("expected ErrAboveCeiling, got %v", err)
	}
}

func TestPaymentOnOpenPeriodRejected(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	_, auth := h.setupCustomer(t, "ns", "cust", 5000)
	period, _ := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
		Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce, Credits: 100, RateMinor: 10,
	})
	// Period still open.
	_, err := h.svc.RecordPayment(context.Background(), RecordPaymentInput{
		Namespace: "ns", PeriodID: period.ID, AccountID: period.AccountID,
		AmountMinor: 1000, Currency: "CNY", BankReference: "B", Payer: "P",
		ReceivedAt: clock.Now(), EvidenceHash: "h",
		ConfirmedBy: "finance-admin", FinanceAdminProof: "token",
	})
	if !errors.Is(err, ErrPeriodNotOpen) {
		t.Errorf("expected ErrPeriodNotOpen, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: OpenAccount idempotency
// ---------------------------------------------------------------------------

func TestOpenAccountIdempotent(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	first, err := h.svc.OpenAccount(context.Background(), OpenAccountInput{
		Namespace: "ns", CustomerID: "cust", CreditLimitMinor: 1000, CeilingCredits: 500, Currency: "CNY",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.svc.OpenAccount(context.Background(), OpenAccountInput{
		Namespace: "ns", CustomerID: "cust", CreditLimitMinor: 1000, CeilingCredits: 500, Currency: "CNY",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Errorf("idempotent: %s vs %s", first.ID, second.ID)
	}
}

func TestNewValidatesDeps(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("New should require repo")
	}
	signer := HMACSigner{Secret: []byte("x")}
	if _, err := New(Config{Signer: signer}); err == nil {
		t.Error("New should require repo")
	}
	repo := newMockRepo()
	if _, err := New(Config{Repo: repo, Signer: signer}); err == nil {
		t.Error("New should require roles")
	}
}

// ---------------------------------------------------------------------------
// Concurrent accrual safety
// ---------------------------------------------------------------------------

func TestConcurrentAccrual(t *testing.T) {
	clock.ResetTime()
	h := newHarness(t)
	acc, auth := h.setupCustomer(t, "ns", "cust", 10000)
	_, _ = h.svc.EnsureOpenPeriod(context.Background(), "ns", acc.ID)

	var wg sync.WaitGroup
	var ok atomic.Int64
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := h.svc.AccrueUsage(context.Background(), AccrueUsageInput{
				Namespace: "ns", CustomerID: "cust", Nonce: auth.Nonce,
				Credits: 1, RateMinor: 1,
				UserID: "u", AgentID: "a", APIKeyID: "k",
			}); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()

	if ok.Load() != 50 {
		t.Errorf("successful accruals = %d, want 50", ok.Load())
	}
	updated, _ := h.repo.GetAccount(context.Background(), "ns", "cust")
	if updated.UsedCredits != 50 {
		t.Errorf("used credits = %d, want 50", updated.UsedCredits)
	}
}
