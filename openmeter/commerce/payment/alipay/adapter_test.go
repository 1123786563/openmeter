package alipay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/url"
	"strings"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

// --- Test key fixtures ---

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

func signMessage(t *testing.T, key *rsa.PrivateKey, message []byte) string {
	t.Helper()
	hashed := sha256.Sum256(message)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

// buildAlipayCallback creates a valid Alipay callback body with signature.
func buildAlipayCallback(t *testing.T, key *rsa.PrivateKey, overrides map[string]string) (url.Values, string, string) {
	t.Helper()
	values := url.Values{}
	values.Set("app_id", "2021000122634567")
	values.Set("charset", "utf-8")
	values.Set("out_trade_no", "ORDER-456")
	values.Set("trade_no", "2025010122001400001234567890")
	values.Set("trade_status", "TRADE_SUCCESS")
	values.Set("total_amount", "0.01")
	values.Set("notify_id", "NOTIFY-001")
	values.Set("notify_time", "2025-01-01 12:00:00")
	values.Set("seller_id", "2088000209967890")
	values.Set("buyer_id", "2088102119123456")
	values.Set("receipt_amount", "0.01")
	values.Set("point_amount", "0.00")
	values.Set("sign_type", "RSA2")

	for k, v := range overrides {
		values.Set(k, v)
	}

	// Build the canonical sign content.
	message := buildSignContent(values)
	sig := signMessage(t, key, []byte(message))
	values.Set("sign", sig)

	return values, values.Encode(), sig
}

// --- Tests ---

func TestVerifyCallbackValidSignature(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{SecretKeyAlipayPublicKey: pubPEM},
	}
	adapter, _ := New(Config{Secrets: secrets})

	values, body, _ := buildAlipayCallback(t, key, nil)

	fact, err := adapter.VerifyCallback(context.Background(), nil, []byte(body))
	if err != nil {
		t.Fatalf("valid callback should succeed: %v", err)
	}

	if !fact.Success {
		t.Error("fact should be success")
	}
	if fact.ProviderOrderID != "ORDER-456" {
		t.Errorf("provider_order_id = %s, want ORDER-456", fact.ProviderOrderID)
	}
	if fact.ProviderPaymentID != "2025010122001400001234567890" {
		t.Errorf("trade_no = %s", fact.ProviderPaymentID)
	}
	if fact.AmountMinor != 1 {
		t.Errorf("amount_minor = %d, want 1", fact.AmountMinor)
	}
	if fact.Currency != "CNY" {
		t.Errorf("currency = %s", fact.Currency)
	}
	if fact.ApplicationID != "2021000122634567" {
		t.Errorf("app_id = %s", fact.ApplicationID)
	}
	if fact.RawHash == "" {
		t.Error("raw_hash should be set")
	}

	_ = values
}

func TestVerifyCallbackInvalidSignature(t *testing.T) {
	_, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{SecretKeyAlipayPublicKey: pubPEM},
	}
	adapter, _ := New(Config{Secrets: secrets})

	// Use a different key to sign.
	otherKey, _ := generateTestKey(t)
	_, body, _ := buildAlipayCallback(t, otherKey, nil)

	_, err := adapter.VerifyCallback(context.Background(), nil, []byte(body))
	if err == nil {
		t.Fatal("invalid signature should fail")
	}
}

func TestVerifyCallbackTamperedAmount(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{SecretKeyAlipayPublicKey: pubPEM},
	}
	adapter, _ := New(Config{Secrets: secrets})

	values, _, _ := buildAlipayCallback(t, key, nil)

	// Tamper with amount after signing — regenerate body with wrong amount
	// but keep the original signature.
	values.Set("total_amount", "99.99")
	tamperedBody := values.Encode()

	_, err := adapter.VerifyCallback(context.Background(), nil, []byte(tamperedBody))
	if err == nil {
		t.Fatal("tampered amount should fail signature verification")
	}
}

func TestVerifyCallbackMissingSign(t *testing.T) {
	_, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{SecretKeyAlipayPublicKey: pubPEM},
	}
	adapter, _ := New(Config{Secrets: secrets})

	body := "app_id=2021000122634567&out_trade_no=ORDER-456&trade_status=TRADE_SUCCESS"
	_, err := adapter.VerifyCallback(context.Background(), nil, []byte(body))
	if err == nil {
		t.Fatal("missing sign should fail")
	}
}

func TestVerifyCallbackWrongTradeStatus(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{SecretKeyAlipayPublicKey: pubPEM},
	}
	adapter, _ := New(Config{Secrets: secrets})

	_, body, _ := buildAlipayCallback(t, key, map[string]string{
		"trade_status": "WAIT_BUYER_PAY",
	})

	fact, err := adapter.VerifyCallback(context.Background(), nil, []byte(body))
	if err != nil {
		t.Fatalf("callback should verify: %v", err)
	}
	if fact.Success {
		t.Error("WAIT_BUYER_PAY should not be success")
	}
}

func TestVerifyCallbackDuplicateEventId(t *testing.T) {
	// This test verifies that the same notify_id produces the same raw_hash
	// for deduplication. The actual dedup is handled by the payment service.
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{SecretKeyAlipayPublicKey: pubPEM},
	}
	adapter, _ := New(Config{Secrets: secrets})

	_, body1, _ := buildAlipayCallback(t, key, nil)

	// The exact same body should produce the same raw_hash.
	fact1, err := adapter.VerifyCallback(context.Background(), nil, []byte(body1))
	if err != nil {
		t.Fatal(err)
	}

	fact2, err := adapter.VerifyCallback(context.Background(), nil, []byte(body1))
	if err != nil {
		t.Fatal(err)
	}

	if fact1.RawHash != fact2.RawHash {
		t.Fatalf("same body should produce same raw_hash: %s vs %s", fact1.RawHash, fact2.RawHash)
	}
}

func TestQueryPayment(t *testing.T) {
	secrets := &payment.StaticSecretProvider{Secrets: map[string]string{}}
	adapter, _ := New(Config{Secrets: secrets})

	fact, err := adapter.QueryPayment(context.Background(), "ORDER-456")
	if err != nil {
		t.Fatal(err)
	}
	if fact.Provider != payment.ProviderAlipay {
		t.Errorf("provider = %s, want alipay", fact.Provider)
	}
}

func TestName(t *testing.T) {
	secrets := &payment.StaticSecretProvider{Secrets: map[string]string{}}
	adapter, _ := New(Config{Secrets: secrets})
	if adapter.Name() != payment.ProviderAlipay {
		t.Errorf("name = %s, want alipay", adapter.Name())
	}
}

func TestBuildSignContentExcludesSignFields(t *testing.T) {
	values := url.Values{}
	values.Set("app_id", "2021")
	values.Set("out_trade_no", "ORDER-1")
	values.Set("sign", "should-be-excluded")
	values.Set("sign_type", "RSA2")
	values.Set("trade_status", "SUCCESS")

	content := buildSignContent(values)

	if strings.Contains(content, "sign=") {
		t.Errorf("sign content should not contain sign field: %s", content)
	}
	if strings.Contains(content, "sign_type=") {
		t.Errorf("sign content should not contain sign_type field: %s", content)
	}
	if !strings.Contains(content, "app_id=2021") {
		t.Errorf("sign content should contain app_id: %s", content)
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"0.01", 1},
		{"1.00", 100},
		{"99.99", 9999},
		{"100", 10000},
		{"", 0},
		{"0.1", 10},
	}
	for _, tt := range tests {
		got := parseAmount(tt.input)
		if got != tt.want {
			t.Errorf("parseAmount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestVerifyCallbackDifferentAppId(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	secrets := &payment.StaticSecretProvider{
		Secrets: map[string]string{SecretKeyAlipayPublicKey: pubPEM},
	}
	adapter, _ := New(Config{Secrets: secrets})

	// A callback for a different app_id — the signature is valid, but the
	// application identity differs from what the merchant expects. The domain
	// service checks this; here we just confirm it's extracted.
	_, body, _ := buildAlipayCallback(t, key, map[string]string{
		"app_id": "DIFFERENT_APP_999",
	})

	fact, err := adapter.VerifyCallback(context.Background(), nil, []byte(body))
	if err != nil {
		t.Fatalf("signature should be valid: %v", err)
	}
	if fact.ApplicationID != "DIFFERENT_APP_999" {
		t.Errorf("app_id = %s, want DIFFERENT_APP_999", fact.ApplicationID)
	}
}
