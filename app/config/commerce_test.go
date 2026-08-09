package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestCommerceConfigurationValidate(t *testing.T) {
	tests := []struct {
		name string
		cfg  CommerceConfiguration
		want string
	}{
		{
			name: "wechat requires every production input",
			cfg: CommerceConfiguration{Enabled: true, Payment: CommercePaymentConfiguration{
				WeChat: WeChatPaymentConfiguration{Enabled: true},
			}},
			want: "commerce.payment.wechat.app_id is required",
		},
		{
			name: "alipay requires gateway identity and keys",
			cfg: CommerceConfiguration{Enabled: true, Payment: CommercePaymentConfiguration{
				Alipay: AlipayPaymentConfiguration{Enabled: true},
			}},
			want: "commerce.payment.alipay.app_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestCommerceConfigurationValidateDisabled(t *testing.T) {
	require.NoError(t, CommerceConfiguration{}.Validate())
}

func TestCommerceConfigurationValidateAllowsTestWeChatBaseURL(t *testing.T) {
	for _, baseURL := range []string{"http://127.0.0.1:8080", "http://payment-provider"} {
		t.Run(baseURL, func(t *testing.T) {
			cfg := validCommerceConfiguration()
			cfg.Payment.WeChat.Enabled = true
			cfg.Payment.WeChat.BaseURL = baseURL

			require.NoError(t, cfg.Validate())
		})
	}
}

func TestCommerceConfigurationValidateAllowsPhase2LocalProviderURLs(t *testing.T) {
	cfg := validCommerceConfiguration()
	cfg.Payment.WeChat.Enabled = true
	cfg.Payment.WeChat.BaseURL = "http://payment-provider:8080"
	cfg.Payment.WeChat.NotifyURL = "http://openmeter:8888/api/v3/payment-providers/wechat/callback"
	cfg.Payment.WeChat.RefundNotifyURL = "http://openmeter:8888/api/v3/payment-providers/wechat/callback"
	cfg.Payment.Alipay.Enabled = true
	cfg.Payment.Alipay.GatewayURL = "http://payment-provider:8080/gateway.do"
	cfg.Payment.Alipay.NotifyURL = "http://openmeter:8888/api/v3/payment-providers/alipay/callback"

	require.NoError(t, cfg.Validate())
}

func TestCommerceConfigurationValidateRejectsExternalHTTPProviderURLs(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*CommerceConfiguration)
		want  string
	}{
		{
			name: "wechat base URL",
			setup: func(cfg *CommerceConfiguration) {
				cfg.Payment.WeChat.Enabled = true
				cfg.Payment.WeChat.BaseURL = "http://example.com:8080"
			},
			want: "commerce.payment.wechat.base_url must be a valid HTTPS URL",
		},
		{
			name: "wechat callback URL",
			setup: func(cfg *CommerceConfiguration) {
				cfg.Payment.WeChat.Enabled = true
				cfg.Payment.WeChat.NotifyURL = "http://example.com/wechat/callback"
			},
			want: "commerce.payment.wechat.notify_url must be a valid HTTPS URL",
		},
		{
			name: "wechat refund callback URL",
			setup: func(cfg *CommerceConfiguration) {
				cfg.Payment.WeChat.Enabled = true
				cfg.Payment.WeChat.RefundNotifyURL = "http://example.com/wechat/refund-callback"
			},
			want: "commerce.payment.wechat.refund_notify_url must be a valid HTTPS URL",
		},
		{
			name: "alipay gateway URL",
			setup: func(cfg *CommerceConfiguration) {
				cfg.Payment.Alipay.Enabled = true
				cfg.Payment.Alipay.GatewayURL = "http://example.com/gateway.do"
			},
			want: "commerce.payment.alipay.gateway_url must be a valid HTTPS URL",
		},
		{
			name: "alipay callback URL",
			setup: func(cfg *CommerceConfiguration) {
				cfg.Payment.Alipay.Enabled = true
				cfg.Payment.Alipay.NotifyURL = "http://example.com/alipay/callback"
			},
			want: "commerce.payment.alipay.notify_url must be a valid HTTPS URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validCommerceConfiguration()
			tt.setup(&cfg)

			require.ErrorContains(t, cfg.Validate(), tt.want)
		})
	}
}

func TestCommerceConfigurationValidateRejectsInsecureProviderURL(t *testing.T) {
	cfg := validCommerceConfiguration()
	cfg.Payment.Alipay.Enabled = true
	cfg.Payment.Alipay.GatewayURL = "http://openapi.alipay.com/gateway.do"

	require.ErrorContains(t, cfg.Validate(), "commerce.payment.alipay.gateway_url must be a valid HTTPS URL")
}

func TestCommerceConfigurationValidateRequiresAlipayGatewayURL(t *testing.T) {
	cfg := validCommerceConfiguration()
	cfg.Payment.Alipay.Enabled = true
	cfg.Payment.Alipay.GatewayURL = ""

	require.ErrorContains(t, cfg.Validate(), "commerce.payment.alipay.gateway_url is required")
}

func TestConfigureCommerceDefaults(t *testing.T) {
	v := viper.New()
	ConfigureCommerce(v)

	require.False(t, v.GetBool("commerce.enabled"))
	require.Equal(t, 10*time.Second, v.GetDuration("commerce.payment.httpTimeout"))
	require.EqualValues(t, 1024*1024, v.GetInt64("commerce.payment.maxResponseBytes"))
	require.Equal(t, 30*time.Second, v.GetDuration("commerce.payment.pendingStaleAfter"))
	require.Equal(t, "https://api.mch.weixin.qq.com", v.GetString("commerce.payment.wechat.baseURL"))
	require.Equal(t, 5*time.Minute, v.GetDuration("commerce.payment.wechat.callbackMaxAge"))
	require.Equal(t, "https://openapi.alipay.com/gateway.do", v.GetString("commerce.payment.alipay.gatewayURL"))
}

func TestConfigurationValidatePrefixesCommerceErrors(t *testing.T) {
	err := (Configuration{
		Commerce: CommerceConfiguration{
			Enabled: true,
			Payment: CommercePaymentConfiguration{
				WeChat: WeChatPaymentConfiguration{Enabled: true},
			},
		},
	}).Validate()

	require.ErrorContains(t, err, "commerce: validation error:")
	require.ErrorContains(t, err, "commerce.payment.wechat.app_id is required")
}

func validCommerceConfiguration() CommerceConfiguration {
	return CommerceConfiguration{
		Enabled: true,
		Payment: CommercePaymentConfiguration{
			HTTPTimeout:       10 * time.Second,
			MaxResponseBytes:  1024 * 1024,
			PendingStaleAfter: 30 * time.Second,
			WeChat: WeChatPaymentConfiguration{
				BaseURL:                "https://api.mch.weixin.qq.com",
				AppID:                  "wx-app-id",
				MerchantID:             "merchant-id",
				MerchantSerial:         "merchant-serial",
				MerchantPrivateKeyFile: "/run/secrets/wechat-merchant-private-key.pem",
				APIv3KeyFile:           "/run/secrets/wechat-api-v3-key",
				PlatformPublicKeyFiles: map[string]string{"platform-serial": "/run/secrets/wechat-platform-public-key.pem"},
				NotifyURL:              "https://openmeter.example.com/commerce/wechat/notify",
				RefundNotifyURL:        "https://openmeter.example.com/commerce/wechat/refund-notify",
				CallbackMaxAge:         5 * time.Minute,
			},
			Alipay: AlipayPaymentConfiguration{
				GatewayURL:          "https://openapi.alipay.com/gateway.do",
				AppID:               "alipay-app-id",
				SellerID:            "seller-id",
				AppPrivateKeyFile:   "/run/secrets/alipay-app-private-key.pem",
				AlipayPublicKeyFile: "/run/secrets/alipay-public-key.pem",
				NotifyURL:           "https://openmeter.example.com/commerce/alipay/notify",
			},
		},
	}
}
