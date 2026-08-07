package ratecard

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// RateCardEntry is the domain representation of a single customer-facing rate.
type RateCardEntry struct {
	ID             string
	Namespace      string
	CustomerID     string
	ResourceCode   aiusage.ResourceCode
	Provider       string
	Model          string
	CreditsPerUnit int64
	UnitSize       int64
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RateCardEntryInput carries the writable fields for create/update.
type RateCardEntryInput struct {
	Namespace      string
	CustomerID     string
	ResourceCode   aiusage.ResourceCode
	Provider       string
	Model          string
	CreditsPerUnit int64
	UnitSize       int64
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
}

// ListParams filters the rate card list query.
type ListParams struct {
	Namespace    string
	ResourceCode string
	Provider     string
	Model        string
	ActiveOnly   bool
}
