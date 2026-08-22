// Package catalog implements product catalog CRUD and SKU management. Products
// are versioned; Version, SKU and Credits are immutable once published. Price
// and sale-interval are mutable but never retroactively affect existing orders.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Repository is the persistence interface for products.
type Repository interface {
	CreateProduct(ctx context.Context, product commerce.Product) (*commerce.Product, error)
	GetProduct(ctx context.Context, namespace, id string) (*commerce.Product, error)
	GetProductBySKU(ctx context.Context, namespace, sku string) (*commerce.Product, error)
	ListProducts(ctx context.Context, namespace string, kind *commerce.ProductKind, activeOnly bool) ([]commerce.Product, error)
	UpdateProduct(ctx context.Context, product commerce.Product) (*commerce.Product, error)
}

// Service is the catalog interface.
type Service interface {
	CreateProduct(ctx context.Context, in commerce.CreateProductInput) (*commerce.Product, error)
	GetProduct(ctx context.Context, namespace, id string) (*commerce.Product, error)
	GetProductBySKU(ctx context.Context, namespace, sku string) (*commerce.Product, error)
	ListProducts(ctx context.Context, namespace string, kind *commerce.ProductKind, activeOnly bool) ([]commerce.Product, error)
	UpdateProduct(ctx context.Context, in commerce.UpdateProductInput) (*commerce.Product, error)
}

// Config wires the catalog service.
type Config struct {
	Repo   Repository
	Logger *slog.Logger
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

// New creates a catalog Service from the given Config.
func New(cfg Config) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{repo: cfg.Repo, logger: logger}
}

// CreateProduct validates and persists a new product. Version starts at 1;
// SKU and Credits are immutable for the lifetime of the product.
func (s *service) CreateProduct(ctx context.Context, in commerce.CreateProductInput) (*commerce.Product, error) {
	if err := validateCreateInput(in); err != nil {
		return nil, err
	}

	now := clock.Now()
	product := commerce.Product{
		NamespacedID: models.NamespacedID{
			Namespace: in.Namespace,
			ID:        ulid.Make().String(),
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		Version:         1,
		SKU:             in.SKU,
		DisplayName:     in.DisplayName,
		Kind:            in.Kind,
		Credits:         in.Credits,
		AmountMinor:     in.AmountMinor,
		Currency:        in.Currency,
		Active:          true,
		DisplayOrder:    in.DisplayOrder,
		ApplicablePlans: in.ApplicablePlans,
		OnSaleAt:        in.OnSaleAt,
		OffSaleAt:       in.OffSaleAt,
		PurchaseLimit:   in.PurchaseLimit,
		RefundPolicy:    in.RefundPolicy,
		BonusCredits:    in.BonusCredits,
		Description:     in.Description,
	}

	return s.repo.CreateProduct(ctx, product)
}

// GetProduct retrieves a product by namespace and ID.
func (s *service) GetProduct(ctx context.Context, namespace, id string) (*commerce.Product, error) {
	if id == "" {
		return nil, fmt.Errorf("product id is required")
	}
	return s.repo.GetProduct(ctx, namespace, id)
}

// GetProductBySKU retrieves a product by namespace and SKU.
func (s *service) GetProductBySKU(ctx context.Context, namespace, sku string) (*commerce.Product, error) {
	if sku == "" {
		return nil, fmt.Errorf("product sku is required")
	}
	return s.repo.GetProductBySKU(ctx, namespace, sku)
}

// ListProducts lists products, optionally filtered by kind and active status.
func (s *service) ListProducts(ctx context.Context, namespace string, kind *commerce.ProductKind, activeOnly bool) ([]commerce.Product, error) {
	return s.repo.ListProducts(ctx, namespace, kind, activeOnly)
}

// UpdateProduct mutates mutable fields. Version, SKU and Credits are never changed.
func (s *service) UpdateProduct(ctx context.Context, in commerce.UpdateProductInput) (*commerce.Product, error) {
	existing, err := s.repo.GetProduct(ctx, in.Namespace, in.ID)
	if err != nil {
		return nil, err
	}

	if in.DisplayName != nil {
		existing.DisplayName = *in.DisplayName
	}
	if in.AmountMinor != nil {
		existing.AmountMinor = *in.AmountMinor
	}
	if in.Active != nil {
		existing.Active = *in.Active
	}
	if in.DisplayOrder != nil {
		existing.DisplayOrder = *in.DisplayOrder
	}
	if in.OnSaleAt != nil {
		existing.OnSaleAt = in.OnSaleAt
	}
	if in.OffSaleAt != nil {
		existing.OffSaleAt = in.OffSaleAt
	}
	existing.UpdatedAt = clock.Now()

	return s.repo.UpdateProduct(ctx, *existing)
}

// validateCreateInput checks the create-product input for required fields and
// valid values.
func validateCreateInput(in commerce.CreateProductInput) error {
	var errs []error

	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.SKU == "" {
		errs = append(errs, errors.New("sku is required"))
	}
	if in.DisplayName == "" {
		errs = append(errs, errors.New("display_name is required"))
	}
	if in.Credits < 0 {
		errs = append(errs, errors.New("credits must be non-negative"))
	}
	if in.AmountMinor < 0 {
		errs = append(errs, errors.New("amount_minor must be non-negative"))
	}
	if in.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}

	// Kind validation.
	switch in.Kind {
	case commerce.ProductKindPlanPurchase, commerce.ProductKindSubscriptionRenewal, commerce.ProductKindWalletTopUp:
	default:
		errs = append(errs, fmt.Errorf("invalid product kind: %s", in.Kind))
	}

	// Refund policy validation (defaults to none).
	if in.RefundPolicy == "" {
		// OK, will default downstream.
	} else {
		switch in.RefundPolicy {
		case commerce.RefundPolicyNone, commerce.RefundPolicyUnspent, commerce.RefundPolicyFullWindow:
		default:
			errs = append(errs, fmt.Errorf("invalid refund policy: %s", in.RefundPolicy))
		}
	}

	// BYOK / Enterprise plan_purchase with 0 credits is allowed.
	// WalletTopUp must have positive credits.
	if in.Kind == commerce.ProductKindWalletTopUp && in.Credits <= 0 {
		errs = append(errs, errors.New("wallet_top_up products must have positive credits"))
	}

	// Sale interval sanity: on_sale must precede off_sale when both are set.
	if in.OnSaleAt != nil && in.OffSaleAt != nil && in.OnSaleAt.After(*in.OffSaleAt) {
		errs = append(errs, errors.New("on_sale_at must not be after off_sale_at"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// IsOnSale returns true if the product is purchasable at the given time.
func IsOnSale(p commerce.Product, at time.Time) bool {
	if !p.Active {
		return false
	}
	if p.OnSaleAt != nil && at.Before(*p.OnSaleAt) {
		return false
	}
	if p.OffSaleAt != nil && at.After(*p.OffSaleAt) {
		return false
	}
	return true
}

var _ Service = (*service)(nil)
