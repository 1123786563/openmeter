package commerce

import (
	"context"
	"time"
)

// Service is the top-level commerce service. It composes the wallet, catalog,
// and order sub-services. Each sub-service is independently testable through its
// own repository interfaces.
type Service interface {
	Wallet() WalletService
	Catalog() CatalogService
	Orders() OrderService
}

// WalletService is the read-only Wallet aggregation interface.
type WalletService interface {
	GetWallet(ctx context.Context, namespace, customerID string) (*Wallet, error)
}

// CatalogService is the product catalog CRUD interface.
type CatalogService interface {
	CreateProduct(ctx context.Context, in CreateProductInput) (*Product, error)
	GetProduct(ctx context.Context, namespace, id string) (*Product, error)
	ListProducts(ctx context.Context, namespace string, kind *ProductKind, activeOnly bool) ([]Product, error)
	UpdateProduct(ctx context.Context, in UpdateProductInput) (*Product, error)
}

// OrderService is the order lifecycle interface.
type OrderService interface {
	CreateOrder(ctx context.Context, in CreateOrderInput) (*Order, bool, error)
	GetOrder(ctx context.Context, namespace, id string) (*Order, error)
	TransitionStatus(ctx context.Context, namespace, id string, to OrderStatus) (*Order, error)
}

// CreateProductInput is the request for creating a catalog product.
type CreateProductInput struct {
	Namespace       string
	SKU             string
	DisplayName     string
	Kind            ProductKind
	Credits         int64
	AmountMinor     int64
	Currency        string
	DisplayOrder    int
	ApplicablePlans []string
	PurchaseLimit   int
	RefundPolicy    RefundPolicy
	BonusCredits    int64
	Description     string
	OnSaleAt        *time.Time
	OffSaleAt       *time.Time
}

// UpdateProductInput mutates a product's mutable fields.
type UpdateProductInput struct {
	Namespace    string
	ID           string
	DisplayName  *string
	AmountMinor  *int64
	Active       *bool
	DisplayOrder *int
	OnSaleAt     *time.Time
	OffSaleAt    *time.Time
}

// CreateOrderInput is the request for creating a new order.
type CreateOrderInput struct {
	Namespace      string
	CustomerID     string
	Kind           OrderKind
	IdempotencyKey string
	Currency       string
	ProductIDs     []string // resolved to line snapshots
	Description    string
}

// service composes the three sub-services.
type service struct {
	wallet  WalletService
	catalog CatalogService
	orders  OrderService
}

func (s *service) Wallet() WalletService  { return s.wallet }
func (s *service) Catalog() CatalogService { return s.catalog }
func (s *service) Orders() OrderService    { return s.orders }

// NewService composes the three sub-services into a top-level commerce Service.
func NewService(wallet WalletService, catalog CatalogService, orders OrderService) Service {
	return &service{wallet: wallet, catalog: catalog, orders: orders}
}
