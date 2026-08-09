package payment

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderConstants verifies the wire string values.
func TestProviderConstants(t *testing.T) {
	assert.Equal(t, "wechat", string(ProviderWeChat))
	assert.Equal(t, "alipay", string(ProviderAlipay))
	assert.Equal(t, "offline", string(ProviderOffline))
}

// TestStaticSecretProviderGet verifies that the in-memory secret provider
// returns the stored value and errors for missing keys.
func TestStaticSecretProviderGet(t *testing.T) {
	sp := &StaticSecretProvider{Secrets: map[string]string{"k1": "v1"}}

	val, err := sp.Get(context.Background(), "k1")
	require.NoError(t, err)
	assert.Equal(t, "v1", val)
}

// TestStaticSecretProviderMissingKey verifies that a missing key returns
// SecretNotFoundError.
func TestStaticSecretProviderMissingKey(t *testing.T) {
	sp := &StaticSecretProvider{Secrets: map[string]string{}}

	_, err := sp.Get(context.Background(), "nonexistent")
	require.Error(t, err)

	var snf *SecretNotFoundError
	require.ErrorAs(t, err, &snf)
	assert.Equal(t, "nonexistent", snf.Key)
}

// TestSecretNotFoundError verifies the error message format.
func TestSecretNotFoundError(t *testing.T) {
	e := &SecretNotFoundError{Key: "my-secret"}
	assert.Contains(t, e.Error(), "my-secret")
	assert.Contains(t, e.Error(), "secret not found")
}

// TestStaticSecretProviderNilMap verifies behavior with an uninitialized map.
func TestStaticSecretProviderNilMap(t *testing.T) {
	sp := &StaticSecretProvider{}
	_, err := sp.Get(context.Background(), "anything")
	require.Error(t, err)
}

// TestProviderAdapterInterfaceCompile verifies the interface is satisfied by
// a minimal stub — this is a compile-time contract guard.
func TestProviderAdapterInterfaceCompile(t *testing.T) {
	var _ ProviderAdapter = (*noopAdapter)(nil)
}

type noopAdapter struct{}

func (noopAdapter) CreateQRCode(context.Context, CheckoutInput) (CheckoutFact, error) {
	return CheckoutFact{}, nil
}
func (noopAdapter) VerifyCallback(context.Context, http.Header, []byte) (PaymentFact, error) {
	return PaymentFact{}, nil
}
func (noopAdapter) QueryPayment(context.Context, string) (PaymentFact, error) {
	return PaymentFact{}, nil
}
func (noopAdapter) Refund(context.Context, RefundInput) (RefundSubmission, error) {
	return RefundSubmission{}, nil
}
func (noopAdapter) QueryRefund(context.Context, RefundQueryInput) (RefundFact, error) {
	return RefundFact{}, nil
}
func (noopAdapter) Name() Provider { return ProviderOffline }
