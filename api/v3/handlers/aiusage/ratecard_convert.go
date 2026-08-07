package aiusage

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/ratecard"
)

// --- API request/response types ---

type rateCardEntryCreate struct {
	ResourceCode   string     `json:"resource_code"`
	Provider       string     `json:"provider,omitempty"`
	Model          string     `json:"model,omitempty"`
	CreditsPerUnit int64      `json:"credits_per_unit"`
	UnitSize       int64      `json:"unit_size"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveTo    *time.Time `json:"effective_to,omitempty"`
}

type rateCardEntryUpdate = rateCardEntryCreate

type rateCardEntryResponse struct {
	ID             string     `json:"id"`
	Namespace      string     `json:"namespace"`
	ResourceCode   string     `json:"resource_code"`
	Provider       string     `json:"provider,omitempty"`
	Model          string     `json:"model,omitempty"`
	CreditsPerUnit int64      `json:"credits_per_unit"`
	UnitSize       int64      `json:"unit_size"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveTo    *time.Time `json:"effective_to,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type rateCardEntryListResponse struct {
	Entries []rateCardEntryResponse `json:"entries"`
}

// --- conversion functions ---

func rateCardEntryToResponse(e *ratecard.RateCardEntry) rateCardEntryResponse {
	return rateCardEntryResponse{
		ID:             e.ID,
		Namespace:      e.Namespace,
		ResourceCode:   string(e.ResourceCode),
		Provider:       e.Provider,
		Model:          e.Model,
		CreditsPerUnit: e.CreditsPerUnit,
		UnitSize:       e.UnitSize,
		EffectiveFrom:  e.EffectiveFrom,
		EffectiveTo:    e.EffectiveTo,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func rateCardCreateToInput(namespace string, body rateCardEntryCreate) ratecard.RateCardEntryInput {
	input := ratecard.RateCardEntryInput{
		Namespace:      namespace,
		ResourceCode:   aiusage.ResourceCode(body.ResourceCode),
		Provider:       body.Provider,
		Model:          body.Model,
		CreditsPerUnit: body.CreditsPerUnit,
		UnitSize:       body.UnitSize,
		EffectiveFrom:  body.EffectiveFrom,
		EffectiveTo:    body.EffectiveTo,
	}
	if input.UnitSize <= 0 {
		input.UnitSize = 1
	}
	return input
}
