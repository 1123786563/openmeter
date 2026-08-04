package wechat

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

// --- Test key fixtures ---

// generateTestKey creates an RSA key pair for testing. Never committed as
// static material — generated fresh on every test run.
func generateTestKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return key, string(pemBytes)
}

// signMessage signs a message with RSA-SHA256/PKCS1v15, returning base64.
func signMessage(t *testing.T, key *rsa.PrivateKey, message []byte) string {
	t.Helper()
	hashed := sha256.Sum256(message)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

// makeCallbackHeaders builds the WeChat Pay callback headers.
func makeCallbackHeaders(timestamp, nonce, signature, serial string) http.Header {
	h := http.Header{}
	h.Set("Wechatpay-Timestamp", timestamp)
	h.Set("Wechatpay-Nonce", nonce)
	h.Set("Wechatpay-Signature", signature)
	h.Set("Wechatpay-Serial", serial)
	return h
}

// validCallbackBody is a well-formed WeChat Pay v3 callback body for a
// successful payment of 1.00 CNY (100 fen) for out_trade_no "ORDER-123".
const validCallbackBody = `{
	"appid": "wx_test_app",
	"mchid": "1230000109",
	"out_trade_no": "ORDER-123",
	"transaction_id": "4200001234202501010001234567",
	"trade_type": "NATIVE",
	"trade_state": "SUCCESS",
	"bank_type": "OTHERS",
	"attach": "",
	"success_time": "2025-01-01T12:00:00+08:00",
	"payer": {"openid": "oUpF8_test"},
	"amount": {"total": 100, "payer_currency": "CNY", "currency": "CNY"}
}`

// --- Tests ---

// TestVerifyCallbackValidSignature verifies that a callback with a valid
// signature and matching fields returns a successful PaymentFact.
func TestVerifyCallbackValidSignature(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{
			SecretKeyPlatformPublicKey: pubPEM,
		},
	}
	adapter, err := New(Config{Secrets: secrets})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(validCallbackBody)
	message := "1234567890\nnonce-abc\n" + string(body) + "\n"
	sig := signMessage(t, key, []byte(message))

	headers := makeCallbackHeaders("1234567890", "nonce-abc", sig, "serial-001")

	fact, err := adapter.VerifyCallback(context.Background(), headers, body)
	if err != nil {
		t.Fatalf("valid callback should succeed: %v", err)
	}

	if !fact.Success {
		t.Error("fact should be success")
	}
	if fact.ProviderOrderID != "ORDER-123" {
		t.Errorf("provider_order_id = %s, want ORDER-123", fact.ProviderOrderID)
	}
	if fact.AmountMinor != 100 {
		t.Errorf("amount_minor = %d, want 100", fact.AmountMinor)
	}
	if fact.Currency != "CNY" {
		t.Errorf("currency = %s, want CNY", fact.Currency)
	}
	if fact.ApplicationID != "wx_test_app" {
		t.Errorf("app_id = %s, want wx_test_app", fact.ApplicationID)
	}
	if fact.MerchantID != "1230000109" {
		t.Errorf("mch_id = %s, want 1230000109", fact.MerchantID)
	}
	if fact.ProviderPaymentID != "4200001234202501010001234567" {
		t.Errorf("transaction_id = %s", fact.ProviderPaymentID)
	}
	if fact.RawHash == "" {
		t.Error("raw_hash should be set")
	}
}

// TestVerifyCallbackInvalidSignature verifies that an invalid signature is rejected.
func TestVerifyCallbackInvalidSignature(t *testing.T) {
	_, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{
			SecretKeyPlatformPublicKey: pubPEM,
		},
	}
	adapter, _ := New(Config{Secrets: secrets})

	body := []byte(validCallbackBody)

	// Sign with a different message so the signature doesn't match.
	otherKey, _ := generateTestKey(t)
	wrongSig := signMessage(t, otherKey, []byte("different message"))

	headers := makeCallbackHeaders("1234567890", "nonce-abc", wrongSig, "serial-001")

	_, err := adapter.VerifyCallback(context.Background(), headers, body)
	if err == nil {
		t.Fatal("invalid signature should fail")
	}
}

// TestVerifyCallbackTamperedBody verifies that a body tampered after signing
// fails verification.
func TestVerifyCallbackTamperedBody(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{
			SecretKeyPlatformPublicKey: pubPEM,
		},
	}
	adapter, _ := New(Config{Secrets: secrets})

	body := []byte(validCallbackBody)
	message := "1234567890\nnonce-abc\n" + string(body) + "\n"
	sig := signMessage(t, key, []byte(message))

	// Tamper with the body after signing.
	tamperedBody := []byte(`{"appid":"wx_test_app","mchid":"1230000109","out_trade_no":"ORDER-999"}`)

	headers := makeCallbackHeaders("1234567890", "nonce-abc", sig, "serial-001")
	_, err := adapter.VerifyCallback(context.Background(), headers, tamperedBody)
	if err == nil {
		t.Fatal("tampered body should fail signature verification")
	}
}

// TestVerifyCallbackMissingHeaders verifies that missing required headers fail.
func TestVerifyCallbackMissingHeaders(t *testing.T) {
	_, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{
			SecretKeyPlatformPublicKey: pubPEM,
		},
	}
	adapter, _ := New(Config{Secrets: secrets})

	// Missing signature header.
	headers := makeCallbackHeaders("1234567890", "nonce-abc", "", "serial-001")
	_, err := adapter.VerifyCallback(context.Background(), headers, []byte("{}"))
	if err == nil {
		t.Fatal("missing signature should fail")
	}
}

// TestVerifyCallbackWrongMerchant verifies that the merchant ID is extracted
// from the verified body (not matched against a config here — the domain
// service does that match).
func TestVerifyCallbackExtractsMerchantAndApp(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{
			SecretKeyPlatformPublicKey: pubPEM,
		},
	}
	adapter, _ := New(Config{Secrets: secrets})

	body := []byte(validCallbackBody)
	message := "1234567890\nnonce-abc\n" + string(body) + "\n"
	sig := signMessage(t, key, []byte(message))

	headers := makeCallbackHeaders("1234567890", "nonce-abc", sig, "serial-001")
	fact, err := adapter.VerifyCallback(context.Background(), headers, body)
	if err != nil {
		t.Fatal(err)
	}

	// The domain service will verify these match the configured merchant/app.
	// Here we just confirm they are extracted.
	if fact.MerchantID == "" {
		t.Error("merchant_id should be extracted")
	}
	if fact.ApplicationID == "" {
		t.Error("application_id should be extracted")
	}
}

// TestVerifyCallbackFailedPayment verifies that a callback for a failed payment
// (non-SUCCESS trade state) returns Success=false.
func TestVerifyCallbackFailedPayment(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{
			SecretKeyPlatformPublicKey: pubPEM,
		},
	}
	adapter, _ := New(Config{Secrets: secrets})

	body := []byte(`{
		"appid": "wx_test_app",
		"mchid": "1230000109",
		"out_trade_no": "ORDER-FAIL",
		"transaction_id": "4200001234202501010009999999",
		"trade_state": "CLOSED",
		"amount": {"total": 100, "currency": "CNY"}
	}`)
	message := "1234567890\nnonce-abc\n" + string(body) + "\n"
	sig := signMessage(t, key, []byte(message))

	headers := makeCallbackHeaders("1234567890", "nonce-abc", sig, "serial-001")
	fact, err := adapter.VerifyCallback(context.Background(), headers, body)
	if err != nil {
		t.Fatal(err)
	}
	if fact.Success {
		t.Error("CLOSED trade state should not be success")
	}
}

// TestQueryPayment verifies the query interface returns the correct provider.
func TestQueryPayment(t *testing.T) {
	secrets := &payment.StaticSecretProvider{Secrets: map[string]string{}}
	adapter, _ := New(Config{Secrets: secrets})

	fact, err := adapter.QueryPayment(context.Background(), "ORDER-123")
	if err != nil {
		t.Fatal(err)
	}
	if fact.Provider != payment.ProviderWeChat {
		t.Errorf("provider = %s, want wechat", fact.Provider)
	}
	if fact.ProviderOrderID != "ORDER-123" {
		t.Errorf("provider_order_id = %s, want ORDER-123", fact.ProviderOrderID)
	}
}

// TestName verifies the provider name.
func TestName(t *testing.T) {
	secrets := &payment.StaticSecretProvider{Secrets: map[string]string{}}
	adapter, _ := New(Config{Secrets: secrets})
	if adapter.Name() != payment.ProviderWeChat {
		t.Errorf("name = %s, want wechat", adapter.Name())
	}
}
