package creditreservations

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Request bodies intentionally use the public v3 snake_case representation.
// The domain model retains its Go-oriented field names at this transport edge.
type resourceLineRequest struct {
	FeatureKey   string            `json:"feature_key"`
	ResourceCode string            `json:"resource_code"`
	Quantity     int64             `json:"quantity"`
	Provider     *string           `json:"provider,omitempty"`
	Model        *string           `json:"model,omitempty"`
	Dimensions   map[string]string `json:"dimensions,omitempty"`
}

type reserveRequest struct {
	ID                     string                `json:"id"`
	CustomerID             string                `json:"customer_id"`
	SubjectID              string                `json:"subject_id"`
	ClientCallID           string                `json:"client_call_id"`
	Operation              string                `json:"operation"`
	IdempotencyKey         string                `json:"idempotency_key"`
	PayloadHash            string                `json:"payload_hash"`
	Lines                  []resourceLineRequest `json:"lines"`
	AuthorizationExpiresAt time.Time             `json:"authorization_expires_at"`
	Provider               *string               `json:"provider,omitempty"`
	Model                  *string               `json:"model,omitempty"`
	RequestID              *string               `json:"request_id,omitempty"`
}

type executeRequest struct {
	IdempotencyKey    string    `json:"idempotency_key"`
	PayloadHash       string    `json:"payload_hash"`
	ExecutionDeadline time.Time `json:"execution_deadline"`
}

type settleRequest struct {
	IdempotencyKey string    `json:"idempotency_key"`
	PayloadHash    string    `json:"payload_hash"`
	ActualCredits  int64     `json:"actual_credits"`
	SettledAt      time.Time `json:"settled_at"`
}

type releaseRequest struct {
	IdempotencyKey    string  `json:"idempotency_key"`
	PayloadHash       string  `json:"payload_hash"`
	EvidenceKind      string  `json:"evidence_kind"`
	EvidenceReference *string `json:"evidence_reference,omitempty"`
}

type unknownRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	PayloadHash    string `json:"payload_hash"`
}

type chargeRequest struct {
	ID             string                `json:"id"`
	CustomerID     string                `json:"customer_id"`
	SubjectID      string                `json:"subject_id"`
	Operation      string                `json:"operation"`
	IdempotencyKey string                `json:"idempotency_key"`
	PayloadHash    string                `json:"payload_hash"`
	Lines          []resourceLineRequest `json:"lines"`
	BookedAt       time.Time             `json:"booked_at"`
}

type reverseChargeRequest struct {
	IdempotencyKey string    `json:"idempotency_key"`
	PayloadHash    string    `json:"payload_hash"`
	ReversedAt     time.Time `json:"reversed_at"`
}

type currencyResponse struct {
	Code             string  `json:"code"`
	CustomCurrencyID *string `json:"custom_currency_id,omitempty"`
}

type ratedLineResponse struct {
	FeatureKey   string            `json:"feature_key"`
	ResourceCode string            `json:"resource_code"`
	Quantity     int64             `json:"quantity"`
	Provider     *string           `json:"provider,omitempty"`
	Model        *string           `json:"model,omitempty"`
	Dimensions   map[string]string `json:"dimensions,omitempty"`
	RateCardKey  string            `json:"rate_card_key"`
	RateVersion  string            `json:"rate_version"`
	Credits      int64             `json:"credits"`
}

type fundingResponse struct {
	PrepaidHold    int64 `json:"prepaid_hold"`
	EnterpriseHold int64 `json:"enterprise_hold"`
}

type reservationResponse struct {
	ID                string              `json:"id"`
	CustomerID        string              `json:"customer_id"`
	Currency          currencyResponse    `json:"currency"`
	State             string              `json:"state"`
	RateVersion       string              `json:"rate_version"`
	Lines             []ratedLineResponse `json:"lines"`
	CeilingCredits    int64               `json:"ceiling_credits"`
	SettledCredits    int64               `json:"settled_credits"`
	ExpiresAt         *time.Time          `json:"expires_at,omitempty"`
	ExecutionDeadline *time.Time          `json:"execution_deadline,omitempty"`
	Funding           fundingResponse     `json:"funding"`
}

type chargeResponse struct {
	ID            string              `json:"id"`
	CustomerID    string              `json:"customer_id"`
	ReservationID *string             `json:"reservation_id,omitempty"`
	Currency      currencyResponse    `json:"currency"`
	RateVersion   string              `json:"rate_version"`
	Lines         []ratedLineResponse `json:"lines"`
	TotalCredits  int64               `json:"total_credits"`
	State         string              `json:"state"`
}

func decodeJSON(r io.Reader, destination any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return validationError(fmt.Errorf("decode request JSON: %w", err))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return validationError(fmt.Errorf("request body must contain one JSON value"))
	}
	return nil
}

func toReserveInput(namespace string, request reserveRequest) (creditreservation.ReserveInput, error) {
	if err := required("id", request.ID); err != nil {
		return creditreservation.ReserveInput{}, err
	}
	if err := required("customer_id", request.CustomerID); err != nil {
		return creditreservation.ReserveInput{}, err
	}
	if err := required("subject_id", request.SubjectID); err != nil {
		return creditreservation.ReserveInput{}, err
	}
	if err := required("client_call_id", request.ClientCallID); err != nil {
		return creditreservation.ReserveInput{}, err
	}
	if err := required("operation", request.Operation); err != nil {
		return creditreservation.ReserveInput{}, err
	}
	lines, err := toResourceLines(request.Lines)
	if err != nil {
		return creditreservation.ReserveInput{}, err
	}
	return creditreservation.ReserveInput{
		ID:                     models.NamespacedID{Namespace: namespace, ID: request.ID},
		CustomerID:             request.CustomerID,
		SubjectID:              request.SubjectID,
		ClientCallID:           request.ClientCallID,
		Operation:              request.Operation,
		CommandIdentity:        creditreservation.CommandIdentity{IdempotencyKey: request.IdempotencyKey, PayloadHash: request.PayloadHash},
		Lines:                  lines,
		AuthorizationExpiresAt: request.AuthorizationExpiresAt,
		Provider:               stringValue(request.Provider),
		Model:                  stringValue(request.Model),
		RequestID:              stringValue(request.RequestID),
	}, nil
}

func toExecuteInput(namespace, id string, request executeRequest) (creditreservation.ExecuteInput, error) {
	if err := required("reservation_id", id); err != nil {
		return creditreservation.ExecuteInput{}, err
	}
	return creditreservation.ExecuteInput{ID: models.NamespacedID{Namespace: namespace, ID: id}, IdempotencyKey: request.IdempotencyKey, PayloadHash: request.PayloadHash, ExecutionDeadline: request.ExecutionDeadline}, nil
}

func toSettleInput(namespace, id string, request settleRequest) (creditreservation.SettleInput, error) {
	if err := required("reservation_id", id); err != nil {
		return creditreservation.SettleInput{}, err
	}
	return creditreservation.SettleInput{ID: models.NamespacedID{Namespace: namespace, ID: id}, CommandIdentity: creditreservation.CommandIdentity{IdempotencyKey: request.IdempotencyKey, PayloadHash: request.PayloadHash}, ActualCredits: request.ActualCredits, SettledAt: request.SettledAt}, nil
}

func toReleaseInput(namespace, id string, request releaseRequest) (creditreservation.ReleaseInput, error) {
	if err := required("reservation_id", id); err != nil {
		return creditreservation.ReleaseInput{}, err
	}
	kind := creditreservation.EvidenceKind(request.EvidenceKind)
	if kind != creditreservation.EvidenceNotSent && kind != creditreservation.EvidenceProviderConfirmedNotExecuted {
		return creditreservation.ReleaseInput{}, validationError(fmt.Errorf("evidence_kind is invalid"))
	}
	return creditreservation.ReleaseInput{ID: models.NamespacedID{Namespace: namespace, ID: id}, IdempotencyKey: request.IdempotencyKey, PayloadHash: request.PayloadHash, Evidence: creditreservation.Evidence{Kind: kind, Reference: stringValue(request.EvidenceReference)}}, nil
}

func toUnknownInput(namespace, id string, request unknownRequest) (creditreservation.UnknownInput, error) {
	if err := required("reservation_id", id); err != nil {
		return creditreservation.UnknownInput{}, err
	}
	return creditreservation.UnknownInput{ID: models.NamespacedID{Namespace: namespace, ID: id}, IdempotencyKey: request.IdempotencyKey, PayloadHash: request.PayloadHash}, nil
}

func toChargeInput(namespace string, request chargeRequest) (creditreservation.ChargeInput, error) {
	if err := required("id", request.ID); err != nil {
		return creditreservation.ChargeInput{}, err
	}
	if err := required("customer_id", request.CustomerID); err != nil {
		return creditreservation.ChargeInput{}, err
	}
	if err := required("subject_id", request.SubjectID); err != nil {
		return creditreservation.ChargeInput{}, err
	}
	if err := required("operation", request.Operation); err != nil {
		return creditreservation.ChargeInput{}, err
	}
	lines, err := toResourceLines(request.Lines)
	if err != nil {
		return creditreservation.ChargeInput{}, err
	}
	return creditreservation.ChargeInput{ID: models.NamespacedID{Namespace: namespace, ID: request.ID}, CustomerID: request.CustomerID, SubjectID: request.SubjectID, Operation: request.Operation, CommandIdentity: creditreservation.CommandIdentity{IdempotencyKey: request.IdempotencyKey, PayloadHash: request.PayloadHash}, Lines: lines, BookedAt: request.BookedAt}, nil
}

func toReverseChargeInput(namespace, id string, request reverseChargeRequest) (creditreservation.ReverseChargeInput, error) {
	if err := required("charge_id", id); err != nil {
		return creditreservation.ReverseChargeInput{}, err
	}
	return creditreservation.ReverseChargeInput{ID: models.NamespacedID{Namespace: namespace, ID: id}, CommandIdentity: creditreservation.CommandIdentity{IdempotencyKey: request.IdempotencyKey, PayloadHash: request.PayloadHash}, ReversedAt: request.ReversedAt}, nil
}

func toResourceLines(lines []resourceLineRequest) ([]creditreservation.ResourceLine, error) {
	if len(lines) == 0 {
		return nil, validationError(fmt.Errorf("lines must contain at least one item"))
	}
	result := make([]creditreservation.ResourceLine, 0, len(lines))
	for index, line := range lines {
		if strings.TrimSpace(line.FeatureKey) == "" || strings.TrimSpace(line.ResourceCode) == "" {
			return nil, validationError(fmt.Errorf("lines[%d] feature_key and resource_code are required", index))
		}
		if line.Quantity < 0 {
			return nil, validationError(fmt.Errorf("lines[%d].quantity must be non-negative", index))
		}
		result = append(result, creditreservation.ResourceLine{FeatureKey: line.FeatureKey, ResourceCode: line.ResourceCode, Quantity: line.Quantity, Provider: stringValue(line.Provider), Model: stringValue(line.Model), Dimensions: line.Dimensions})
	}
	return result, nil
}

func toReservationResponse(reservation creditreservation.Reservation) reservationResponse {
	return reservationResponse{ID: reservation.ID, CustomerID: reservation.CustomerID, Currency: toCurrencyResponse(reservation.Currency.Code.String(), reservation.Currency.CustomCurrencyID), State: strings.ToLower(string(reservation.State)), RateVersion: reservation.RateVersion, Lines: toRatedLineResponses(reservation.Lines), CeilingCredits: reservation.TotalCredits, SettledCredits: reservation.SettledCredits, ExpiresAt: reservation.ExpiresAt, ExecutionDeadline: reservation.ExecutionDeadline, Funding: fundingResponse{PrepaidHold: reservation.PrepaidHold, EnterpriseHold: reservation.EnterpriseHold}}
}

func toChargeResponse(charge creditreservation.Charge) chargeResponse {
	return chargeResponse{ID: charge.ID, CustomerID: charge.CustomerID, ReservationID: optionalString(charge.ReservationID), Currency: toCurrencyResponse(charge.Currency.Code.String(), charge.Currency.CustomCurrencyID), RateVersion: charge.RateVersion, Lines: toRatedLineResponses(charge.Lines), TotalCredits: charge.TotalCredits, State: strings.ToLower(charge.State)}
}

func toCurrencyResponse(code string, customCurrencyID *string) currencyResponse {
	return currencyResponse{Code: code, CustomCurrencyID: customCurrencyID}
}

func toRatedLineResponses(lines []creditreservation.RatedLine) []ratedLineResponse {
	result := make([]ratedLineResponse, 0, len(lines))
	for _, line := range lines {
		result = append(result, ratedLineResponse{FeatureKey: line.FeatureKey, ResourceCode: line.ResourceCode, Quantity: line.Quantity, Provider: optionalString(line.Provider), Model: optionalString(line.Model), Dimensions: line.Dimensions, RateCardKey: line.RateCardKey, RateVersion: line.RateVersion, Credits: line.Credits})
	}
	return result
}

func required(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return validationError(fmt.Errorf("%s is required", name))
	}
	return nil
}

func validationError(err error) error {
	return models.NewGenericValidationError(err)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
