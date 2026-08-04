// Package payment implements payment fact verification and the
// paid-versus-fulfilled separation. A PaymentFact is an immutable record of a
// verified provider callback. The Provider boundary returns verified facts;
// only the domain service decides order transitions.
//
// Design rules (from the phase 2 brief):
//   - "paid" means a verified provider fact exists.
//   - "fulfilled" means the commercial invoice is paid and Credits granted
//     atomically.
//   - All provider callbacks must verify signatures, deduplicate, and
//     hash-retain the raw body.
//   - Keys come from Secret Manager only — never in API, logs, or Ent files.
//   - Credits are int64 — no floats.
package payment

import (
	"context"
	"net/http"
	"time"
)

// Provider identifies a payment channel. It is the type-level guarantee that a
// callback or query result came from a specific provider after verification.
type Provider string

const (
	ProviderWeChat  Provider = "wechat"
	ProviderAlipay  Provider = "alipay"
	ProviderOffline Provider = "offline"
)

// CheckoutInput carries the order context needed to create a provider payment
// session (QR code). The provider assigns its own order/payment IDs; the domain
// stores them on the PaymentAttempt.
type CheckoutInput struct {
	Namespace      string
	OrderID        string
	OrderPublicID  string
	CustomerID     string
	AmountMinor    int64
	Currency       string
	Description    string
	NotifyURL      string
	IdempotencyKey string
}

// CheckoutFact is the result of initiating a provider payment. The QRCodeURL
// is returned to the client; the provider order ID is persisted for later
// reconciliation.
type CheckoutFact struct {
	Provider          Provider
	ProviderOrderID   string
	ProviderPaymentID string
	QRCodeURL         string
	ExpiresAt         *time.Time
}

// PaymentFact is an immutable, verified record that a payment succeeded. It
// carries only fields that were cryptographically verified by the provider
// adapter — never the raw callback body. The RawHash is the SHA-256 of the raw
// body, retained for deduplication without persisting sensitive material.
type PaymentFact struct {
	Provider          Provider
	ProviderOrderID   string
	ProviderPaymentID string
	MerchantID        string
	ApplicationID     string
	ProviderEventID   string
	AmountMinor       int64
	Currency          string
	Success           bool
	RawHash           string
	Timestamp         time.Time
	// SignedPayload stores verified fields extracted from the callback. This is
	// the durable audit record — it never contains the raw body.
	SignedPayload map[string]any
}

// RefundInput carries the context for a provider refund.
type RefundInput struct {
	Namespace         string
	OrderID           string
	ProviderOrderID   string
	ProviderPaymentID string
	AmountMinor       int64
	Currency          string
	Reason            string
	IdempotencyKey    string
}

// RefundSubmission is the result of submitting a refund to a provider. The
// provider refund ID is used to query the refund status later.
type RefundSubmission struct {
	Provider         Provider
	ProviderRefundID string
	Status           string
}

// RefundFact is a verified refund result from the provider.
type RefundFact struct {
	Provider         Provider
	ProviderRefundID string
	ProviderOrderID  string
	AmountMinor      int64
	Currency         string
	Success          bool
	RawHash          string
	Timestamp        time.Time
	SignedPayload    map[string]any
}

// Provider is the boundary between the payment domain and external payment
// channels (WeChat Pay, Alipay, offline). Adapters return verified facts; only
// the domain service decides transitions. Implementations must:
//
//   - Verify signatures before returning any fact.
//   - Never log or persist raw API keys.
//   - Source all secrets from a SecretProvider.
type ProviderAdapter interface {
	// CreateQRCode initiates a payment session at the provider and returns a
	// checkout fact (QR code + provider IDs).
	CreateQRCode(ctx context.Context, input CheckoutInput) (CheckoutFact, error)

	// VerifyCallback verifies a provider callback's signature, extracts the
	// verified fields, and returns a PaymentFact. The raw body is never stored;
	// only its SHA-256 hash (RawHash) is retained for deduplication.
	VerifyCallback(ctx context.Context, headers http.Header, body []byte) (PaymentFact, error)

	// QueryPayment queries the provider directly for the payment status of a
	// provider order. This is used when the callback is lost or to confirm a
	// payment before fulfillment.
	QueryPayment(ctx context.Context, providerOrderID string) (PaymentFact, error)

	// Refund submits a refund to the provider.
	Refund(ctx context.Context, input RefundInput) (RefundSubmission, error)

	// QueryRefund queries the provider for a refund's status.
	QueryRefund(ctx context.Context, providerRefundID string) (RefundFact, error)

	// Name returns the provider identifier.
	Name() Provider
}

// SecretProvider supplies secrets from a secret manager. Keys are never
// embedded in configuration, logs, or Ent files. Implementations read from
// the platform secret store (e.g. Vault, cloud secret manager).
type SecretProvider interface {
	// Get returns the secret value for the given key, or an error if not found.
	Get(ctx context.Context, key string) (string, error)
}

// StaticSecretProvider is a test-only secret provider that returns secrets from
// an in-memory map. Production code must use a real secret manager.
type StaticSecretProvider struct {
	Secrets map[string]string
}

// Get returns the secret for the key.
func (s *StaticSecretProvider) Get(_ context.Context, key string) (string, error) {
	if v, ok := s.Secrets[key]; ok {
		return v, nil
	}
	return "", &SecretNotFoundError{Key: key}
}

// SecretNotFoundError is returned when a secret is not found in the secret provider.
type SecretNotFoundError struct {
	Key string
}

func (e *SecretNotFoundError) Error() string {
	return "secret not found: " + e.Key
}
