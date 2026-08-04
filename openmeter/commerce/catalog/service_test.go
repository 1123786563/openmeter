package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
)

// mockRepo is an in-memory ProductRepository for testing.
type mockRepo struct {
	products map[string]*commerce.Product
	bySKU    map[string]string // sku -> id
	nextErr  error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		products: make(map[string]*commerce.Product),
		bySKU:    make(map[string]string),
	}
}

func (m *mockRepo) CreateProduct(_ context.Context, p commerce.Product) (*commerce.Product, error) {
	if m.nextErr != nil {
		err := m.nextErr
		m.nextErr = nil
		return nil, err
	}
	if _, ok := m.bySKU[p.Namespace+"/"+p.SKU]; ok {
		return nil, commerce.ErrSKUNotUnique
	}
	saved := p
	m.products[p.ID] = &saved
	m.bySKU[p.Namespace+"/"+p.SKU] = p.ID
	return &saved, nil
}

func (m *mockRepo) GetProduct(_ context.Context, namespace, id string) (*commerce.Product, error) {
	p, ok := m.products[id]
	if !ok || p.Namespace != namespace {
		return nil, commerce.ErrProductNotFound
	}
	return p, nil
}

func (m *mockRepo) GetProductBySKU(_ context.Context, namespace, sku string) (*commerce.Product, error) {
	id, ok := m.bySKU[namespace+"/"+sku]
	if !ok {
		return nil, commerce.ErrProductNotFound
	}
	return m.GetProduct(context.Background(), namespace, id)
}

func (m *mockRepo) ListProducts(_ context.Context, namespace string, kind *commerce.ProductKind, activeOnly bool) ([]commerce.Product, error) {
	var result []commerce.Product
	for _, p := range m.products {
		if p.Namespace != namespace {
			continue
		}
		if kind != nil && p.Kind != *kind {
			continue
		}
		if activeOnly && !p.Active {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockRepo) UpdateProduct(_ context.Context, p commerce.Product) (*commerce.Product, error) {
	existing, ok := m.products[p.ID]
	if !ok {
		return nil, commerce.ErrProductNotFound
	}
	existing.DisplayName = p.DisplayName
	existing.AmountMinor = p.AmountMinor
	existing.Active = p.Active
	existing.DisplayOrder = p.DisplayOrder
	existing.UpdatedAt = p.UpdatedAt
	return existing, nil
}

// --- Tests ---

// TestCreateFreeProduct creates a Free plan product and asserts all SKU attributes.
func TestCreateFreeProduct(t *testing.T) {
	repo := newMockRepo()
	svc := New(Config{Repo: repo})

	product, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:    "ns",
		SKU:          "PLAN-FREE",
		DisplayName:  "Free Plan",
		Kind:         commerce.ProductKindPlanPurchase,
		Credits:      0, // BYOK: token cost AND sales Credit are 0
		AmountMinor:  0,
		Currency:     "CNY",
		RefundPolicy: commerce.RefundPolicyNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertSKUAttributes(t, product, "PLAN-FREE", "Free Plan", 0, 0, "CNY", commerce.ProductKindPlanPurchase)
	if product.Version != 1 {
		t.Errorf("version = %d, want 1", product.Version)
	}
	if !product.Active {
		t.Error("new product should be active")
	}
}

// TestCreateProAndTeamProducts creates Pro and Team plan products.
func TestCreateProAndTeamProducts(t *testing.T) {
	repo := newMockRepo()
	svc := New(Config{Repo: repo})

	pro, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:    "ns",
		SKU:          "PLAN-PRO-MONTHLY",
		DisplayName:  "Pro Plan Monthly",
		Kind:         commerce.ProductKindPlanPurchase,
		Credits:      10000,
		AmountMinor:  9900,
		Currency:     "CNY",
		DisplayOrder: 10,
		RefundPolicy: commerce.RefundPolicyFullWindow,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertSKUAttributes(t, pro, "PLAN-PRO-MONTHLY", "Pro Plan Monthly", 10000, 9900, "CNY", commerce.ProductKindPlanPurchase)
	if pro.DisplayOrder != 10 {
		t.Errorf("display_order = %d, want 10", pro.DisplayOrder)
	}

	team, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:    "ns",
		SKU:          "PLAN-TEAM-MONTHLY",
		DisplayName:  "Team Plan Monthly",
		Kind:         commerce.ProductKindPlanPurchase,
		Credits:      50000,
		AmountMinor:  29900,
		Currency:     "CNY",
		DisplayOrder: 20,
		RefundPolicy: commerce.RefundPolicyFullWindow,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertSKUAttributes(t, team, "PLAN-TEAM-MONTHLY", "Team Plan Monthly", 50000, 29900, "CNY", commerce.ProductKindPlanPurchase)
}

// TestCreateRechargeProduct creates a wallet top-up product and verifies all
// immutable SKU attributes from the brief: version, public code, display name,
// integer Credit, amount_minor, currency, applicable plans, sale interval,
// purchase limit, and refund policy.
func TestCreateRechargeProduct(t *testing.T) {
	repo := newMockRepo()
	svc := New(Config{Repo: repo})

	now := time.Now()
	later := now.Add(24 * time.Hour)

	product, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:       "ns",
		SKU:             "RECHARGE-1000",
		DisplayName:     "1,000 Points Pack",
		Kind:            commerce.ProductKindWalletTopUp,
		Credits:         1000,
		AmountMinor:     9900,
		Currency:        "CNY",
		DisplayOrder:    1,
		ApplicablePlans: []string{"PLAN-PRO-MONTHLY", "PLAN-TEAM-MONTHLY"},
		OnSaleAt:        &now,
		OffSaleAt:       &later,
		PurchaseLimit:   5,
		RefundPolicy:    commerce.RefundPolicyUnspent,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Immutable version
	if product.Version != 1 {
		t.Errorf("version = %d, want 1 (immutable)", product.Version)
	}
	// Public code (SKU)
	if product.SKU != "RECHARGE-1000" {
		t.Errorf("sku = %s, want RECHARGE-1000", product.SKU)
	}
	// Display name
	if product.DisplayName != "1,000 Points Pack" {
		t.Errorf("display_name = %s", product.DisplayName)
	}
	// Integer Credit
	if product.Credits != 1000 {
		t.Errorf("credits = %d, want 1000", product.Credits)
	}
	// amount_minor
	if product.AmountMinor != 9900 {
		t.Errorf("amount_minor = %d, want 9900", product.AmountMinor)
	}
	// currency
	if product.Currency != "CNY" {
		t.Errorf("currency = %s, want CNY", product.Currency)
	}
	// applicable plans
	if len(product.ApplicablePlans) != 2 {
		t.Errorf("applicable_plans len = %d, want 2", len(product.ApplicablePlans))
	}
	// on/off-sale interval
	if product.OnSaleAt == nil || product.OffSaleAt == nil {
		t.Error("sale interval should be set")
	}
	// purchase limit
	if product.PurchaseLimit != 5 {
		t.Errorf("purchase_limit = %d, want 5", product.PurchaseLimit)
	}
	// refund policy
	if product.RefundPolicy != commerce.RefundPolicyUnspent {
		t.Errorf("refund_policy = %s, want unspent", product.RefundPolicy)
	}
}

// TestCreateEnterpriseQuoteProduct verifies enterprise quote-backed products
// can be created with enterprise_receivable-style attributes.
func TestCreateEnterpriseQuoteProduct(t *testing.T) {
	repo := newMockRepo()
	svc := New(Config{Repo: repo})

	product, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:       "ns",
		SKU:             "PLAN-ENTERPRISE-YEARLY",
		DisplayName:     "Enterprise Plan (Quote-Based)",
		Kind:            commerce.ProductKindSubscriptionRenewal,
		Credits:         0, // Enterprise: credit from quote, not SKU
		AmountMinor:     0, // Billed via receivable, not upfront
		Currency:        "CNY",
		ApplicablePlans: []string{"PLAN-ENTERPRISE-YEARLY"},
		RefundPolicy:    commerce.RefundPolicyNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if product.Kind != commerce.ProductKindSubscriptionRenewal {
		t.Errorf("kind = %s, want subscription_renewal", product.Kind)
	}
}

// TestSKUIsUniqueWithinNamespace verifies that duplicate SKUs in the same
// namespace are rejected.
func TestSKUIsUniqueWithinNamespace(t *testing.T) {
	repo := newMockRepo()
	svc := New(Config{Repo: repo})

	_, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:   "ns",
		SKU:         "DUP-SKU",
		DisplayName: "First",
		Kind:        commerce.ProductKindWalletTopUp,
		Credits:     100,
		AmountMinor: 1000,
		Currency:    "CNY",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:   "ns",
		SKU:         "DUP-SKU",
		DisplayName: "Second",
		Kind:        commerce.ProductKindWalletTopUp,
		Credits:     200,
		AmountMinor: 2000,
		Currency:    "CNY",
	})
	if err == nil {
		t.Fatal("duplicate SKU should fail")
	}
}

// TestUpdateProductDoesNotChangeImmutableFields verifies that updating a product
// never changes Version, SKU, or Credits.
func TestUpdateProductDoesNotChangeImmutableFields(t *testing.T) {
	repo := newMockRepo()
	svc := New(Config{Repo: repo})

	created, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:   "ns",
		SKU:         "RECHARGE-500",
		DisplayName: "500 Points",
		Kind:        commerce.ProductKindWalletTopUp,
		Credits:     500,
		AmountMinor: 4900,
		Currency:    "CNY",
	})
	if err != nil {
		t.Fatal(err)
	}

	active := false
	updated, err := svc.UpdateProduct(context.Background(), commerce.UpdateProductInput{
		Namespace: "ns",
		ID:        created.ID,
		Active:    &active,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != created.Version {
		t.Errorf("version changed: %d -> %d", created.Version, updated.Version)
	}
	if updated.SKU != created.SKU {
		t.Errorf("sku changed: %s -> %s", created.SKU, updated.SKU)
	}
	if updated.Credits != created.Credits {
		t.Errorf("credits changed: %d -> %d", created.Credits, updated.Credits)
	}
	if updated.Active {
		t.Error("active should be false after update")
	}
}

// TestIsOnSale checks the on-sale logic.
func TestIsOnSale(t *testing.T) {
	now := time.Now()
	before := now.Add(-1 * time.Hour)
	after := now.Add(1 * time.Hour)

	// Active, no interval.
	p1 := commerce.Product{Active: true}
	if !IsOnSale(p1, now) {
		t.Error("active product with no interval should be on sale")
	}

	// Inactive.
	p2 := commerce.Product{Active: false}
	if IsOnSale(p2, now) {
		t.Error("inactive product should not be on sale")
	}

	// Before on-sale.
	p3 := commerce.Product{Active: true, OnSaleAt: &after}
	if IsOnSale(p3, now) {
		t.Error("product before on_sale_at should not be on sale")
	}

	// After off-sale.
	p4 := commerce.Product{Active: true, OffSaleAt: &before}
	if IsOnSale(p4, now) {
		t.Error("product after off_sale_at should not be on sale")
	}

	// Within interval.
	p5 := commerce.Product{Active: true, OnSaleAt: &before, OffSaleAt: &after}
	if !IsOnSale(p5, now) {
		t.Error("product within interval should be on sale")
	}
}

// TestWalletTopUpRequiresPositiveCredits verifies that wallet_top_up products
// must have positive credits.
func TestWalletTopUpRequiresPositiveCredits(t *testing.T) {
	repo := newMockRepo()
	svc := New(Config{Repo: repo})

	_, err := svc.CreateProduct(context.Background(), commerce.CreateProductInput{
		Namespace:   "ns",
		SKU:         "BAD-RECHARGE",
		DisplayName: "Zero Credits",
		Kind:        commerce.ProductKindWalletTopUp,
		Credits:     0,
		AmountMinor: 1000,
		Currency:    "CNY",
	})
	if err == nil {
		t.Fatal("wallet_top_up with 0 credits should fail validation")
	}
}

func assertSKUAttributes(t *testing.T, p *commerce.Product, sku, name string, credits, amount int64, currency string, kind commerce.ProductKind) {
	t.Helper()
	if p.SKU != sku {
		t.Errorf("sku = %s, want %s", p.SKU, sku)
	}
	if p.DisplayName != name {
		t.Errorf("display_name = %s, want %s", p.DisplayName, name)
	}
	if p.Credits != credits {
		t.Errorf("credits = %d, want %d", p.Credits, credits)
	}
	if p.AmountMinor != amount {
		t.Errorf("amount_minor = %d, want %d", p.AmountMinor, amount)
	}
	if p.Currency != currency {
		t.Errorf("currency = %s, want %s", p.Currency, currency)
	}
	if p.Kind != kind {
		t.Errorf("kind = %s, want %s", p.Kind, kind)
	}
}
