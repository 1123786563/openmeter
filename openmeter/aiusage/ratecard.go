package aiusage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
)

// CustomerRateCardEntry maps a resource to its customer-facing price.
type CustomerRateCardEntry struct {
	models.ManagedModel `json:",inline"`

	Namespace       string                `json:"namespace"`
	CustomerID      *string               `json:"customer_id,omitempty"` // nil = namespace default
	ResourceCode    ResourceCode          `json:"resource_code"`
	Provider        *string               `json:"provider,omitempty"` // nil = all providers
	Model           *string               `json:"model,omitempty"`     // nil = all models
	PricePerUnitCNY alpacadecimal.Decimal `json:"price_per_unit_cny"`
	CreditRate      int64                 `json:"credit_rate"` // 1 CNY = X Credits
	EffectiveFrom   time.Time             `json:"effective_from"`
	EffectiveTo     *time.Time            `json:"effective_to,omitempty"`
}

func (e CustomerRateCardEntry) Validate() error {
	var errs []error

	if e.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace must not be empty"))
	}
	if err := e.ResourceCode.Validate(); err != nil {
		errs = append(errs, err)
	}
	if e.CreditRate <= 0 {
		errs = append(errs, fmt.Errorf("credit_rate must be positive"))
	}
	if e.EffectiveFrom.IsZero() {
		errs = append(errs, fmt.Errorf("effective_from must not be zero"))
	}
	if e.PricePerUnitCNY.IsNegative() {
		errs = append(errs, fmt.Errorf("price_per_unit_cny must not be negative"))
	}
	if e.EffectiveTo != nil && e.EffectiveTo.Before(e.EffectiveFrom) {
		errs = append(errs, fmt.Errorf("effective_to must not be before effective_from"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// IsCustomerSpecific returns true when this entry is scoped to a specific customer.
func (e CustomerRateCardEntry) IsCustomerSpecific() bool {
	return e.CustomerID != nil
}

// MatchPriority returns a priority score for rate resolution. Higher wins.
//
// Customer + provider + model specific = 100 (most specific)
// Customer + provider                = 80
// Customer + resource only           = 60
// Namespace + provider + model       = 40
// Namespace + resource only          = 20 (least specific)
func (e CustomerRateCardEntry) MatchPriority() int {
	score := 0

	if e.IsCustomerSpecific() {
		score += 50
	}

	hasProvider := e.Provider != nil
	hasModel := e.Model != nil

	if hasProvider && hasModel {
		score += 50
	} else if hasProvider {
		score += 30
	}

	return score
}

// IsActiveAt returns true if the entry is effective at the given time.
func (e CustomerRateCardEntry) IsActiveAt(at time.Time) bool {
	if at.Before(e.EffectiveFrom) {
		return false
	}
	if e.EffectiveTo != nil && at.After(*e.EffectiveTo) {
		return false
	}
	return true
}

// RateCardResolver resolves the effective rate for a resource at a point in time.
type RateCardResolver interface {
	Resolve(ctx context.Context, namespace string, customerID string, resource ResourceCode,
		provider string, model string, at time.Time) (CustomerRateCardEntry, error)
}

// MatchEntry checks whether this entry applies to the given lookup criteria.
func (e CustomerRateCardEntry) Matches(namespace, customerID string, resource ResourceCode, provider, model string) bool {
	if e.Namespace != namespace {
		return false
	}
	if e.ResourceCode != resource {
		return false
	}

	// Customer scope: if entry has a customer, it must match; if nil, it's a namespace default
	if e.CustomerID != nil && *e.CustomerID != customerID {
		return false
	}

	// Provider scope: if entry has a provider, it must match
	if e.Provider != nil && *e.Provider != provider {
		return false
	}

	// Model scope: if entry has a model, it must match
	if e.Model != nil && *e.Model != model {
		return false
	}

	return true
}
