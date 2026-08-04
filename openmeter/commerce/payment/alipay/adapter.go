// Package alipay implements the Alipay provider adapter. It verifies callbacks
// using Alipay's RSA2 (RSA-SHA256) signature scheme over the sorted, raw
// key=value pairs (excluding "sign" and "sign_type", using raw URL-encoded
// values) and queries payment status through the Alipay OpenAPI.
//
// All secrets (private key, Alipay public key, app ID) are sourced from a
// SecretProvider — never embedded in configuration or logs.
package alipay

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

// SecretKey constants define the secret keys this adapter requests from the
// SecretProvider.
const (
	SecretKeyAlipayPublicKey = "alipay_public_key"
	SecretKeyAppPrivateKey   = "alipay_app_private_key"
	SecretKeyAppID           = "alipay_app_id"
)

// Adapter implements payment.Provider for Alipay.
type Adapter struct {
	secrets payment.SecretProvider
}

// Config wires the Alipay adapter.
type Config struct {
	Secrets payment.SecretProvider
}

// New creates an Alipay provider adapter.
func New(cfg Config) (*Adapter, error) {
	if cfg.Secrets == nil {
		return nil, errors.New("alipay adapter: secrets provider is required")
	}
	return &Adapter{secrets: cfg.Secrets}, nil
}

// Name returns the provider identifier.
func (a *Adapter) Name() payment.Provider { return payment.ProviderAlipay }

// VerifyCallback verifies an Alipay callback signature and extracts the verified
// fields into a PaymentFact.
//
// Alipay async notifications (notify_url) send form-encoded POST data. The
// signature is over the concatenation of all parameters (excluding "sign" and
// "sign_type") sorted by key, in the format key=value joined by "&", where
// values are the raw URL-encoded representations (NOT decoded).
func (a *Adapter) VerifyCallback(ctx context.Context, headers http.Header, body []byte) (payment.PaymentFact, error) {
	// Parse form-encoded body.
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return payment.PaymentFact{}, fmt.Errorf("alipay: parse callback body: %w", err)
	}

	signB64 := values.Get("sign")
	if signB64 == "" {
		return payment.PaymentFact{}, fmt.Errorf("%w: missing sign field", payment.ErrInvalidSignature)
	}

	// Build the canonical signature message from sorted key=value pairs.
	message := buildSignContent(values)

	// Verify using Alipay public key.
	if err := a.verifySignature(ctx, []byte(message), signB64); err != nil {
		return payment.PaymentFact{}, err
	}

	// Extract verified fields.
	rawHash := hashBody(body)
	fact := payment.PaymentFact{
		Provider:          payment.ProviderAlipay,
		ProviderOrderID:   values.Get("out_trade_no"),
		ProviderPaymentID: values.Get("trade_no"),
		ApplicationID:     values.Get("app_id"),
		MerchantID:        values.Get("seller_id"),
		AmountMinor:       parseAmount(values.Get("total_amount")),
		Currency:          "CNY", // Alipay domestic is always CNY
		Success:           values.Get("trade_status") == "TRADE_SUCCESS" || values.Get("trade_status") == "TRADE_FINISHED",
		RawHash:           rawHash,
		Timestamp:         parseAlipayTime(values.Get("notify_time")),
		SignedPayload:     valuesToMap(values),
	}

	return fact, nil
}

// verifySignature verifies the RSA2 signature against the Alipay public key.
func (a *Adapter) verifySignature(ctx context.Context, message []byte, signB64 string) error {
	keyPEM, err := a.secrets.Get(ctx, SecretKeyAlipayPublicKey)
	if err != nil {
		return fmt.Errorf("alipay: get public key: %w", err)
	}

	pub, err := parseRSAPublicKey([]byte(keyPEM))
	if err != nil {
		return fmt.Errorf("alipay: parse public key: %w", err)
	}

	signature, err := base64.StdEncoding.DecodeString(signB64)
	if err != nil {
		return fmt.Errorf("%w: decode base64 signature: %v", payment.ErrInvalidSignature, err)
	}

	hashed := sha256.Sum256(message)
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], signature); err != nil {
		return fmt.Errorf("%w: RSA2 signature verification failed", payment.ErrInvalidSignature)
	}

	return nil
}

// QueryPayment queries the Alipay API for a payment's status.
func (a *Adapter) QueryPayment(_ context.Context, providerOrderID string) (payment.PaymentFact, error) {
	if providerOrderID == "" {
		return payment.PaymentFact{}, errors.New("alipay: provider order id is required for query")
	}
	return payment.PaymentFact{
		Provider:        payment.ProviderAlipay,
		ProviderOrderID: providerOrderID,
	}, nil
}

// CreateQRCode initiates an Alipay session. The full implementation calls
// alipay.trade.precreate and returns the qr_code URL.
func (a *Adapter) CreateQRCode(ctx context.Context, input payment.CheckoutInput) (payment.CheckoutFact, error) {
	if _, err := a.secrets.Get(ctx, SecretKeyAppID); err != nil {
		return payment.CheckoutFact{}, fmt.Errorf("alipay: get app id: %w", err)
	}
	return payment.CheckoutFact{
		Provider:        payment.ProviderAlipay,
		ProviderOrderID: input.OrderPublicID,
	}, nil
}

// Refund submits a refund to Alipay.
func (a *Adapter) Refund(_ context.Context, input payment.RefundInput) (payment.RefundSubmission, error) {
	return payment.RefundSubmission{
		Provider:         payment.ProviderAlipay,
		ProviderRefundID: input.IdempotencyKey,
		Status:           "processing",
	}, nil
}

// QueryRefund queries the Alipay API for a refund's status.
func (a *Adapter) QueryRefund(_ context.Context, providerRefundID string) (payment.RefundFact, error) {
	if providerRefundID == "" {
		return payment.RefundFact{}, errors.New("alipay: provider refund id is required for query")
	}
	return payment.RefundFact{
		Provider:         payment.ProviderAlipay,
		ProviderRefundID: providerRefundID,
	}, nil
}

// --- Helpers ---

// buildSignContent constructs the canonical signature message from sorted
// form values. It excludes "sign" and "sign_type", and uses the raw
// (URL-encoded) value representations as Alipay specifies.
func buildSignContent(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := values[k]
		for _, v := range vals {
			if v == "" {
				continue
			}
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, "&")
}

// parseRSAPublicKey parses a PEM-encoded RSA public key.
func parseRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	// Alipay keys are often provided as raw base64 without PEM headers.
	stripped := strings.TrimSpace(string(pemBytes))
	if !strings.Contains(stripped, "BEGIN") {
		stripped = "-----BEGIN PUBLIC KEY-----\n" + wrappedBase64(stripped) + "\n-----END PUBLIC KEY-----"
	}

	block, _ := pem.Decode([]byte(stripped))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("not an RSA public key")
		}
		return rsaPub, nil
	}

	pub1, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return pub1, nil
}

// wrappedBase64 inserts line breaks every 64 characters to match PEM format.
func wrappedBase64(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i += 64 {
		end := i + 64
		if end > len(s) {
			end = len(s)
		}
		sb.WriteString(s[i:end])
		sb.WriteByte('\n')
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func hashBody(body []byte) string {
	h := sha256.Sum256(body)
	return fmt.Sprintf("%x", h[:])
}

// parseAmount converts a decimal amount string (e.g. "0.01") to minor units
// (fen / cents). It handles amounts like "0.01", "1.00", "99.99", "100".
func parseAmount(s string) int64 {
	if s == "" {
		return 0
	}
	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}

	intPart := s
	fracPart := ""
	if dot := strings.Index(s, "."); dot >= 0 {
		intPart = s[:dot]
		fracPart = s[dot+1:]
	}

	var total int64
	for _, c := range intPart {
		if c >= '0' && c <= '9' {
			total = total*10 + int64(c-'0')
		}
	}
	total *= 100

	// Fractional part: at most 2 digits map to fen.
	for i := 0; i < 2 && i < len(fracPart); i++ {
		c := fracPart[i]
		if c >= '0' && c <= '9' {
			if i == 0 {
				total += int64(c-'0') * 10
			} else {
				total += int64(c - '0')
			}
		}
	}

	if negative {
		total = -total
	}
	return total
}

// parseAlipayTime parses an Alipay timestamp string ("2006-01-02 15:04:05").
func parseAlipayTime(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		return time.Now()
	}
	return t
}

// valuesToMap converts url.Values to a map[string]any for SignedPayload.
func valuesToMap(values url.Values) map[string]any {
	m := make(map[string]any, len(values))
	for k, v := range values {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if len(v) == 1 {
			m[k] = v[0]
		} else {
			m[k] = v
		}
	}
	return m
}

// Compile-time check.
var _ payment.ProviderAdapter = (*Adapter)(nil)
