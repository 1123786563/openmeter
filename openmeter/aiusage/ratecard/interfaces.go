package ratecard

import "context"

// Service exposes CRUD operations for rate card entries.
type Service interface {
	Create(ctx context.Context, input RateCardEntryInput) (*RateCardEntry, error)
	Get(ctx context.Context, namespace, id string) (*RateCardEntry, error)
	List(ctx context.Context, params ListParams) ([]RateCardEntry, error)
	Update(ctx context.Context, namespace, id string, input RateCardEntryInput) (*RateCardEntry, error)
	Delete(ctx context.Context, namespace, id string) error
	BootstrapSeed(ctx context.Context, namespace string, entries []RateCardEntryInput) error
}
