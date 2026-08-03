package aiusage

import (
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// fromAPIBatchCreate converts the API request body into the domain IngestBatchInput.
func fromAPIBatchCreate(namespace string, body api.AIUsageUsageBatchCreate) aiusage.IngestBatchInput {
	ceiling := body.ReservationCeilingCredits
	reservationID := body.ReservationId

	input := aiusage.IngestBatchInput{
		Namespace:       namespace,
		CustomerID:      body.BillingCustomerId,
		SubjectID:       body.SubjectKey,
		UsageBatchID:    body.IdempotencyKey,
		TenantSeq:       body.TenantSeq,
		OccurredAt:      time.Time(body.OccurredAt),
		ReservationID:   &reservationID,
		CeilingCredits:  &ceiling,
		RateVersion:     body.RatePackageVersion,
		BillingMode:     aiusage.BillingMode(body.BillingMode),
		PayloadHash:     body.PayloadHash,
		ProviderManaged: body.ProviderManaged,
	}

	for _, line := range body.Lines {
		item := aiusage.UsageLineItem{
			ResourceCode:    aiusage.ResourceCode(line.ResourceCode),
			Quantity:        line.Quantity,
			Provider:        ptrStringVal(line.Provider),
			Model:           ptrStringVal(line.Model),
			ProviderManaged: body.ProviderManaged,
			Dimensions:      ptrMapVal(line.PricingDimensions),
		}
		input.LineItems = append(input.LineItems, item)
	}

	return input
}

// toAPIBatch builds the API response from the domain batch metadata and the
// settlement result.
func toAPIBatch(
	input aiusage.IngestBatchInput,
	result *aiusage.BatchSettlementResult,
	createdAt time.Time,
) api.AIUsageUsageBatch {
	resp := api.AIUsageUsageBatch{
		Id:                        result.BatchID,
		IdempotencyKey:            input.UsageBatchID,
		PayloadHash:               input.PayloadHash,
		BillingCustomerId:         input.CustomerID,
		SubjectKey:                input.SubjectID,
		TenantSeq:                 input.TenantSeq,
		OccurredAt:                api.DateTime(input.OccurredAt),
		ReservationId:             ptrStringVal(input.ReservationID),
		RatePackageVersion:        input.RateVersion,
		BillingMode:               api.AIUsageBillingMode(input.BillingMode),
		ProviderManaged:           input.ProviderManaged,
		Status:                    api.AIUsageBatchStatus(result.Status),
		TotalCredits:              result.TotalCredits,
		CoveredTenantSeq:          result.CoveredTenantSeq,
		CreatedAt:                 api.DateTime(createdAt),
		ReservationCeilingCredits: ptrInt64Val(input.CeilingCredits),
	}

	for i, item := range input.LineItems {
		line := api.AIUsageUsageLine{
			ResourceCode:       string(item.ResourceCode),
			Quantity:           item.Quantity,
			Provider:           stringPtrOrNil(item.Provider),
			Model:              stringPtrOrNil(item.Model),
			PricingDimensions:  mapPtrOrNil(item.Dimensions),
			CanonicalLineIndex: int32(i),
		}
		if i < len(result.RatingSnapshots) {
			snap := result.RatingSnapshots[i]
			line.Credits = snap.Credits
			line.CostSnapshot = costSnapshotToAPI(snap.CostSnapshot)
			line.SalesSnapshot = salesSnapshotToAPI(snap.SalesSnapshot)
		}
		resp.Lines = append(resp.Lines, line)
	}

	return resp
}

// toAPIBatchFromDomain converts a stored domain batch to the API response.
func toAPIBatchFromDomain(batch *aiusage.AIUsageBatch) api.AIUsageUsageBatch {
	resp := api.AIUsageUsageBatch{
		Id:                        batch.UsageBatchID,
		IdempotencyKey:            batch.UsageBatchID,
		PayloadHash:               batch.PayloadHash,
		BillingCustomerId:         batch.CustomerID,
		SubjectKey:                batch.SubjectID,
		TenantSeq:                 batch.TenantSeq,
		OccurredAt:                api.DateTime(batch.OccurredAt),
		ReservationId:             ptrStringVal(batch.ReservationID),
		RatePackageVersion:        batch.RateVersion,
		BillingMode:               api.AIUsageBillingMode(batch.BillingMode),
		Status:                    api.AIUsageBatchStatus(batch.Status),
		CreatedAt:                 api.DateTime(batch.CreatedAt),
		ReservationCeilingCredits: ptrInt64Val(batch.CeilingCredits),
	}

	for i, item := range batch.LineItems {
		line := api.AIUsageUsageLine{
			ResourceCode:       string(item.ResourceCode),
			Quantity:           item.Quantity,
			Provider:           stringPtrOrNil(item.Provider),
			Model:              stringPtrOrNil(item.Model),
			PricingDimensions:  mapPtrOrNil(item.Dimensions),
			CanonicalLineIndex: int32(i),
		}
		resp.Lines = append(resp.Lines, line)
	}

	return resp
}

// toAPIRuntimeAuthorization maps the handler view to the API response.
func toAPIRuntimeAuthorization(view authorizationPackageView, now time.Time) api.AIUsageRuntimeAuthorization {
	resp := api.AIUsageRuntimeAuthorization{
		ContractVersion:  view.ContractVersion,
		RetrievedAt:      api.DateTime(now),
		Authorized:       view.Authorized,
		AvailableCredits: view.AvailableCredits,
		CoveredTenantSeq: view.CoveredTenantSeq,
	}
	if view.ReservationCeilingCredits > 0 {
		v := view.ReservationCeilingCredits
		resp.ReservationCeilingCredits = &v
	}
	if view.DenialReason != "" {
		resp.DenialReason = &view.DenialReason
	}
	return resp
}

func costSnapshotToAPI(snap aiusage.CostSnapshot) *api.AIUsageCostSnapshot {
	return &api.AIUsageCostSnapshot{
		Currency: api.BillingCurrencyCode(snap.Currency),
		Amount:   snap.Amount.String(),
		Source:   snap.Source,
	}
}

func salesSnapshotToAPI(snap aiusage.SalesSnapshot) *api.AIUsageSalesSnapshot {
	return &api.AIUsageSalesSnapshot{
		Currency:        api.BillingCurrencyCode(snap.Currency),
		Amount:          snap.Amount.String(),
		RateCardVersion: snap.RateCardVersion,
	}
}

// --- helpers ---

func ptrStringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrMapVal(m *map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	return *m
}

func mapPtrOrNil(m map[string]string) *map[string]string {
	if m == nil {
		return nil
	}
	return &m
}

func ptrInt64Val(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
