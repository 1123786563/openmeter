package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type testGrantConnector struct{}

func (testGrantConnector) CreateGrant(context.Context, models.NamespacedID, credit.CreateGrantInput) (*grant.Grant, error) {
	return &grant.Grant{}, nil
}

func (testGrantConnector) VoidGrant(context.Context, models.NamespacedID, *time.Time) error {
	return nil
}

type testFenceClient struct{}

func (testFenceClient) EstablishFence(context.Context, string, string, string) (refund.FenceResult, error) {
	return refund.FenceResult{Sequence: "fence-1", Established: true}, nil
}

func (testFenceClient) ReleaseFence(context.Context, string, string, string, string) error {
	return nil
}

func (testFenceClient) ConfirmSnapshotApplied(context.Context, string, string, string) (bool, error) {
	return true, nil
}

type testCreditReverser struct{}

func (testCreditReverser) ReverseCredits(_ context.Context, in refund.ReverseCreditsInput) (refund.ReverseCreditsResult, error) {
	return refund.ReverseCreditsResult{LedgerEntryID: "reversal-1", Credits: in.Credits}, nil
}

type testSnapshotPublisher struct{}

func (testSnapshotPublisher) PublishSnapshot(context.Context, refund.PublishSnapshotInput) (string, error) {
	return "snapshot-1", nil
}

type testNoopFence struct{ testFenceClient }

func (testNoopFence) IsNoop() bool { return true }

type testRefundProcessorService struct{ err error }

func (s testRefundProcessorService) ProcessOne(context.Context, string, string) (*refund.RefundRequest, error) {
	return nil, s.err
}

func completeRuntimeDependencies() *commerceRuntimeDependencies {
	return &commerceRuntimeDependencies{
		RefundFence:             testFenceClient{},
		RefundCreditReverser:    testCreditReverser{},
		RefundSnapshotPublisher: testSnapshotPublisher{},
	}
}

func TestWireCommerceDisabledKeepsReadOnlyWiringWithoutWorkers(t *testing.T) {
	wiring, err := wireCommerce(
		entdb.NewClient(), "default", testGrantConnector{},
		config.CommerceConfiguration{Enabled: false}, nil, testutils.NewLogger(t),
	)
	require.NoError(t, err)
	require.NotNil(t, wiring.Handler)
	require.NotNil(t, wiring.Catalog)
	require.Nil(t, wiring.WorkerManager)
	require.Empty(t, wiring.paymentProviders)
	require.Empty(t, wiring.refundProviders)
}

func TestWireCommerceDisabledRejectsAllMutationHandlers(t *testing.T) {
	wiring, err := wireCommerce(
		entdb.NewClient(), "default", testGrantConnector{},
		config.CommerceConfiguration{Enabled: false}, nil, testutils.NewLogger(t),
	)
	require.NoError(t, err)

	mutations := []struct {
		name    string
		method  string
		handler http.HandlerFunc
	}{
		{name: "create product", method: http.MethodPost, handler: wiring.Handler.CreateProduct()},
		{name: "update product", method: http.MethodPut, handler: wiring.Handler.UpdateProduct()},
		{name: "create order", method: http.MethodPost, handler: wiring.Handler.CreateOrder()},
		{name: "create checkout", method: http.MethodPost, handler: wiring.Handler.CreateCheckoutSession()},
		{name: "alipay callback", method: http.MethodPost, handler: wiring.Handler.AlipayPaymentCallback()},
		{name: "wechat callback", method: http.MethodPost, handler: wiring.Handler.WechatPaymentCallback()},
		{name: "create refund", method: http.MethodPost, handler: wiring.Handler.CreateRefund()},
		{name: "create offline payment", method: http.MethodPost, handler: wiring.Handler.CreateOfflinePayment()},
		{name: "update external invoice", method: http.MethodPut, handler: wiring.Handler.UpdateExternalInvoice()},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(mutation.method, "/commerce-disabled", nil)
			mutation.handler.ServeHTTP(response, request)
			require.Equal(t, http.StatusNotImplemented, response.Code)
			require.Empty(t, response.Body.String())
		})
	}
}

func TestWireCommerceEnabledFailsClosedWithoutRealRefundDependencies(t *testing.T) {
	cfg := validCommerceConfiguration(t)

	for _, tt := range []struct {
		name string
		deps *commerceRuntimeDependencies
		want string
	}{
		{name: "missing bundle", want: "refund fence"},
		{
			name: "missing credit reverser",
			deps: &commerceRuntimeDependencies{RefundFence: testFenceClient{}, RefundSnapshotPublisher: testSnapshotPublisher{}},
			want: "credit reverser",
		},
		{
			name: "missing snapshot publisher",
			deps: &commerceRuntimeDependencies{RefundFence: testFenceClient{}, RefundCreditReverser: testCreditReverser{}},
			want: "snapshot publisher",
		},
		{
			name: "noop dependency",
			deps: &commerceRuntimeDependencies{RefundFence: testNoopFence{}, RefundCreditReverser: testCreditReverser{}, RefundSnapshotPublisher: testSnapshotPublisher{}},
			want: "refund fence",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wiring, err := wireCommerce(
				entdb.NewClient(), "default", testGrantConnector{}, cfg, tt.deps, testutils.NewLogger(t),
			)
			require.Nil(t, wiring)
			require.ErrorContains(t, err, "commerce automatic refund disabled")
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestWireCommerceEnabledWithoutChannelsStillRequiresRealRefundDependencies(t *testing.T) {
	cfg := config.CommerceConfiguration{
		Enabled: true,
		Payment: config.CommercePaymentConfiguration{
			HTTPTimeout: time.Second, MaxResponseBytes: 1024, PendingStaleAfter: 30 * time.Second,
		},
	}

	wiring, err := wireCommerce(
		entdb.NewClient(), "default", testGrantConnector{}, cfg, nil, testutils.NewLogger(t),
	)
	require.Nil(t, wiring)
	require.ErrorContains(t, err, "commerce automatic refund disabled")
}

func TestWireCommerceEnabledRegistersProviderRecoveryWorkersWithCompleteDependencies(t *testing.T) {
	wiring, err := wireCommerce(
		entdb.NewClient(), "default", testGrantConnector{},
		validCommerceConfiguration(t), completeRuntimeDependencies(), testutils.NewLogger(t),
	)
	require.NoError(t, err)
	require.Equal(t, []string{"fulfillment", "refund-query", "payment-query-recovery", "reconciliation"}, wiring.WorkerManager.RunnerNames())
	require.Contains(t, wiring.paymentProviders, payment.ProviderWeChat)
	require.Contains(t, wiring.refundProviders, payment.ProviderWeChat)
	require.Same(t, wiring.paymentProviders[payment.ProviderWeChat], wiring.refundProviders[payment.ProviderWeChat])
	require.Contains(t, wiring.paymentProviders, payment.ProviderAlipay)
	require.Contains(t, wiring.refundProviders, payment.ProviderAlipay)
	require.Same(t, wiring.paymentProviders[payment.ProviderAlipay], wiring.refundProviders[payment.ProviderAlipay])
}

func TestWireCommerceLoadsProviderSecretsAtStartup(t *testing.T) {
	cfg := validCommerceConfiguration(t)
	cfg.Payment.WeChat.MerchantPrivateKeyFile = filepath.Join(t.TempDir(), "missing.pem")

	wiring, err := wireCommerce(
		entdb.NewClient(), "default", testGrantConnector{}, cfg,
		completeRuntimeDependencies(), testutils.NewLogger(t),
	)
	require.Nil(t, wiring)
	require.Error(t, err)
	var pathErr *os.PathError
	require.True(t, errors.As(err, &pathErr))
}

func TestWirePaymentProvidersRejectsMalformedSecretFilesAtStartup(t *testing.T) {
	for _, tt := range []struct {
		name       string
		secretPath func(config.CommerceConfiguration) string
	}{
		{name: "WeChat merchant private key", secretPath: func(cfg config.CommerceConfiguration) string { return cfg.Payment.WeChat.MerchantPrivateKeyFile }},
		{name: "WeChat API v3 key", secretPath: func(cfg config.CommerceConfiguration) string { return cfg.Payment.WeChat.APIv3KeyFile }},
		{name: "WeChat platform public key", secretPath: func(cfg config.CommerceConfiguration) string {
			return cfg.Payment.WeChat.PlatformPublicKeyFiles["platform-serial"]
		}},
		{name: "Alipay application private key", secretPath: func(cfg config.CommerceConfiguration) string { return cfg.Payment.Alipay.AppPrivateKeyFile }},
		{name: "Alipay public key", secretPath: func(cfg config.CommerceConfiguration) string { return cfg.Payment.Alipay.AlipayPublicKeyFile }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validCommerceConfiguration(t)
			const secretMarker = "malformed-secret-file-content-marker"
			require.NoError(t, os.WriteFile(tt.secretPath(cfg), []byte(secretMarker), 0o600))

			paymentProviders, refundProviders, err := wirePaymentProviders(cfg, testutils.NewLogger(t))
			require.Error(t, err)
			require.Nil(t, paymentProviders)
			require.Nil(t, refundProviders)
			require.NotContains(t, err.Error(), secretMarker)
		})
	}
}

func TestValidateFlagRunsProductionProviderAssembly(t *testing.T) {
	if configPath := os.Getenv("OPENMETER_VALIDATE_TEST_CONFIG"); configPath != "" {
		os.Args = []string{"openmeter", "--validate", "--config", configPath}
		main()
		return
	}

	for _, tt := range []struct {
		name   string
		mutate func(*config.CommerceConfiguration)
	}{
		{name: "valid configuration"},
		{name: "missing key file", mutate: func(cfg *config.CommerceConfiguration) {
			cfg.Payment.WeChat.MerchantPrivateKeyFile = filepath.Join(t.TempDir(), "missing.pem")
		}},
		{name: "malformed key file", mutate: func(cfg *config.CommerceConfiguration) {
			const secretMarker = "validate-malformed-secret-content-marker"
			require.NoError(t, os.WriteFile(cfg.Payment.Alipay.AlipayPublicKeyFile, []byte(secretMarker), 0o600))
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validCommerceConfiguration(t)
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			configPath := writeCommerceValidationConfig(t, cfg)
			output, err := runValidateSubprocess(t, configPath)
			if tt.mutate == nil {
				require.NoError(t, err, output)
				return
			}
			require.Error(t, err, output)
			require.NotContains(t, output, "validate-malformed-secret-content-marker")
		})
	}
}

func TestValidateFlagAllowsDisabledCommerceWithoutProviderFiles(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "openmeter.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("commerce:\n  enabled: false\n"), 0o600))
	output, err := runValidateSubprocess(t, configPath)
	require.NoError(t, err, output)
}

func runValidateSubprocess(t *testing.T, configPath string) (string, error) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestValidateFlagRunsProductionProviderAssembly$")
	command.Env = append(os.Environ(), "OPENMETER_VALIDATE_TEST_CONFIG="+configPath)
	output, err := command.CombinedOutput()
	return string(output), err
}

func writeCommerceValidationConfig(t *testing.T, cfg config.CommerceConfiguration) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "openmeter.yaml")
	contents := fmt.Sprintf(`commerce:
  enabled: true
  payment:
    httpTimeout: %q
    maxResponseBytes: %d
    pendingStaleAfter: %q
    wechat:
      enabled: true
      baseURL: %q
      appID: %q
      merchantID: %q
      merchantSerial: %q
      merchantPrivateKeyFile: %q
      apiV3KeyFile: %q
      platformPublicKeyFiles:
        platform-serial: %q
      notifyURL: %q
      refundNotifyURL: %q
      callbackMaxAge: %q
    alipay:
      enabled: true
      gatewayURL: %q
      appID: %q
      sellerID: %q
      appPrivateKeyFile: %q
      alipayPublicKeyFile: %q
      notifyURL: %q
`,
		cfg.Payment.HTTPTimeout.String(), cfg.Payment.MaxResponseBytes, cfg.Payment.PendingStaleAfter.String(),
		cfg.Payment.WeChat.BaseURL, cfg.Payment.WeChat.AppID, cfg.Payment.WeChat.MerchantID,
		cfg.Payment.WeChat.MerchantSerial, cfg.Payment.WeChat.MerchantPrivateKeyFile,
		cfg.Payment.WeChat.APIv3KeyFile, cfg.Payment.WeChat.PlatformPublicKeyFiles["platform-serial"],
		cfg.Payment.WeChat.NotifyURL, cfg.Payment.WeChat.RefundNotifyURL, cfg.Payment.WeChat.CallbackMaxAge.String(),
		cfg.Payment.Alipay.GatewayURL, cfg.Payment.Alipay.AppID, cfg.Payment.Alipay.SellerID,
		cfg.Payment.Alipay.AppPrivateKeyFile, cfg.Payment.Alipay.AlipayPublicKeyFile, cfg.Payment.Alipay.NotifyURL,
	)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestRefundWorkerAdapterPassesThroughProcessError(t *testing.T) {
	wantErr := errors.New("provider timeout")
	adapter := refundWorkerAdapter{svc: testRefundProcessorService{err: wantErr}}
	require.ErrorIs(t, adapter.ProcessOne(t.Context(), "default", "refund-1"), wantErr)
}

func validCommerceConfiguration(t *testing.T) config.CommerceConfiguration {
	t.Helper()
	dir := t.TempDir()
	writeSecret := func(name, value string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(value), 0o600))
		return path
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyDER})

	return config.CommerceConfiguration{
		Enabled: true,
		Payment: config.CommercePaymentConfiguration{
			HTTPTimeout: 2 * time.Second, MaxResponseBytes: 1 << 20, PendingStaleAfter: 30 * time.Second,
			WeChat: config.WeChatPaymentConfiguration{
				Enabled: true, BaseURL: "https://api.mch.weixin.qq.com", AppID: "wx-app", MerchantID: "wx-mch",
				MerchantSerial: "merchant-serial", MerchantPrivateKeyFile: writeSecret("merchant.pem", string(privateKeyPEM)),
				APIv3KeyFile:           writeSecret("api-v3-key", "0123456789abcdef0123456789abcdef"),
				PlatformPublicKeyFiles: map[string]string{"platform-serial": writeSecret("platform.pem", string(publicKeyPEM))},
				NotifyURL:              "https://merchant.example/wechat/notify", RefundNotifyURL: "https://merchant.example/wechat/refund-notify",
				CallbackMaxAge: 5 * time.Minute,
			},
			Alipay: config.AlipayPaymentConfiguration{
				Enabled: true, GatewayURL: "https://openapi.alipay.com/gateway.do",
				AppID: "ali-app", SellerID: "ali-seller",
				AppPrivateKeyFile:   writeSecret("alipay-app-private.pem", string(privateKeyPEM)),
				AlipayPublicKeyFile: writeSecret("alipay-public.pem", string(publicKeyPEM)),
				NotifyURL:           "https://merchant.example/alipay/notify",
			},
		},
	}
}
