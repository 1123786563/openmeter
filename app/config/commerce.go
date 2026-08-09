package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CommerceConfiguration struct {
	Enabled bool                         `yaml:"enabled" mapstructure:"enabled"`
	Payment CommercePaymentConfiguration `yaml:"payment" mapstructure:"payment"`
}

type CommercePaymentConfiguration struct {
	HTTPTimeout       time.Duration              `yaml:"httpTimeout" mapstructure:"httpTimeout"`
	MaxResponseBytes  int64                      `yaml:"maxResponseBytes" mapstructure:"maxResponseBytes"`
	PendingStaleAfter time.Duration              `yaml:"pendingStaleAfter" mapstructure:"pendingStaleAfter"`
	WeChat            WeChatPaymentConfiguration `yaml:"wechat" mapstructure:"wechat"`
	Alipay            AlipayPaymentConfiguration `yaml:"alipay" mapstructure:"alipay"`
}

type WeChatPaymentConfiguration struct {
	Enabled                bool              `yaml:"enabled" mapstructure:"enabled"`
	BaseURL                string            `yaml:"baseURL" mapstructure:"baseURL"`
	AppID                  string            `yaml:"appID" mapstructure:"appID"`
	MerchantID             string            `yaml:"merchantID" mapstructure:"merchantID"`
	MerchantSerial         string            `yaml:"merchantSerial" mapstructure:"merchantSerial"`
	MerchantPrivateKeyFile string            `yaml:"merchantPrivateKeyFile" mapstructure:"merchantPrivateKeyFile"`
	APIv3KeyFile           string            `yaml:"apiV3KeyFile" mapstructure:"apiV3KeyFile"`
	PlatformPublicKeyFiles map[string]string `yaml:"platformPublicKeyFiles" mapstructure:"platformPublicKeyFiles"`
	NotifyURL              string            `yaml:"notifyURL" mapstructure:"notifyURL"`
	RefundNotifyURL        string            `yaml:"refundNotifyURL" mapstructure:"refundNotifyURL"`
	CallbackMaxAge         time.Duration     `yaml:"callbackMaxAge" mapstructure:"callbackMaxAge"`
}

type AlipayPaymentConfiguration struct {
	Enabled             bool   `yaml:"enabled" mapstructure:"enabled"`
	GatewayURL          string `yaml:"gatewayURL" mapstructure:"gatewayURL"`
	AppID               string `yaml:"appID" mapstructure:"appID"`
	SellerID            string `yaml:"sellerID" mapstructure:"sellerID"`
	AppPrivateKeyFile   string `yaml:"appPrivateKeyFile" mapstructure:"appPrivateKeyFile"`
	AlipayPublicKeyFile string `yaml:"alipayPublicKeyFile" mapstructure:"alipayPublicKeyFile"`
	NotifyURL           string `yaml:"notifyURL" mapstructure:"notifyURL"`
}

func (c CommerceConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error

	if c.Payment.HTTPTimeout <= 0 {
		errs = append(errs, errors.New("commerce.payment.http_timeout must be positive"))
	}

	if c.Payment.MaxResponseBytes <= 0 {
		errs = append(errs, errors.New("commerce.payment.max_response_bytes must be positive"))
	}

	if c.Payment.PendingStaleAfter <= 0 {
		errs = append(errs, errors.New("commerce.payment.pending_stale_after must be positive"))
	}

	errs = append(errs, c.Payment.WeChat.validate()...)
	errs = append(errs, c.Payment.Alipay.validate()...)

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c WeChatPaymentConfiguration) validate() []error {
	if !c.Enabled {
		return nil
	}

	const prefix = "commerce.payment.wechat"
	errs := requiredCommerceFields(prefix, map[string]string{
		"app_id":                    c.AppID,
		"merchant_id":               c.MerchantID,
		"merchant_serial":           c.MerchantSerial,
		"merchant_private_key_file": c.MerchantPrivateKeyFile,
		"api_v3_key_file":           c.APIv3KeyFile,
		"notify_url":                c.NotifyURL,
		"refund_notify_url":         c.RefundNotifyURL,
	})

	if strings.TrimSpace(c.BaseURL) == "" {
		errs = append(errs, fmt.Errorf("%s.base_url is required", prefix))
	} else if err := validateCommerceURL(c.BaseURL, true); err != nil {
		errs = append(errs, fmt.Errorf("%s.base_url must be a valid HTTPS URL: %w", prefix, err))
	}

	if len(c.PlatformPublicKeyFiles) == 0 {
		errs = append(errs, fmt.Errorf("%s.platform_public_key_files is required", prefix))
	}

	for serial, path := range c.PlatformPublicKeyFiles {
		if strings.TrimSpace(serial) == "" {
			errs = append(errs, fmt.Errorf("%s.platform_public_key_files contains an empty serial", prefix))
		}
		if strings.TrimSpace(path) == "" {
			errs = append(errs, fmt.Errorf("%s.platform_public_key_files[%q] is required", prefix, serial))
		}
	}

	errs = append(errs, validateRequiredCommerceURL(prefix+".notify_url", c.NotifyURL, true))
	errs = append(errs, validateRequiredCommerceURL(prefix+".refund_notify_url", c.RefundNotifyURL, true))

	if c.CallbackMaxAge <= 0 {
		errs = append(errs, fmt.Errorf("%s.callback_max_age must be positive", prefix))
	}

	return errs
}

func (c AlipayPaymentConfiguration) validate() []error {
	if !c.Enabled {
		return nil
	}

	const prefix = "commerce.payment.alipay"
	errs := requiredCommerceFields(prefix, map[string]string{
		"app_id":                 c.AppID,
		"seller_id":              c.SellerID,
		"app_private_key_file":   c.AppPrivateKeyFile,
		"alipay_public_key_file": c.AlipayPublicKeyFile,
		"notify_url":             c.NotifyURL,
	})

	if strings.TrimSpace(c.GatewayURL) == "" {
		errs = append(errs, fmt.Errorf("%s.gateway_url is required", prefix))
	} else if err := validateCommerceURL(c.GatewayURL, true); err != nil {
		errs = append(errs, fmt.Errorf("%s.gateway_url must be a valid HTTPS URL: %w", prefix, err))
	}

	errs = append(errs, validateRequiredCommerceURL(prefix+".notify_url", c.NotifyURL, true))

	return errs
}

func requiredCommerceFields(prefix string, fields map[string]string) []error {
	errs := make([]error, 0, len(fields))
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("%s.%s is required", prefix, name))
		}
	}

	return errs
}

func validateRequiredCommerceURL(field, value string, allowTestHTTP bool) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	if err := validateCommerceURL(value, allowTestHTTP); err != nil {
		return fmt.Errorf("%s must be a valid HTTPS URL: %w", field, err)
	}

	return nil
}

func validateCommerceURL(rawURL string, allowTestHTTP bool) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		return errors.New("invalid URL")
	}

	if parsed.Scheme == "https" {
		return nil
	}

	if allowTestHTTP && parsed.Scheme == "http" && isCommerceLocalHTTPHost(parsed.Hostname()) {
		return nil
	}

	return errors.New("HTTPS is required")
}

func isCommerceLocalHTTPHost(host string) bool {
	switch strings.ToLower(host) {
	case "127.0.0.1", "payment-provider", "openmeter":
		return true
	default:
		return false
	}
}

func ConfigureCommerce(v *viper.Viper) {
	v.SetDefault("commerce.enabled", false)
	v.SetDefault("commerce.payment.httpTimeout", 10*time.Second)
	v.SetDefault("commerce.payment.maxResponseBytes", int64(1024*1024))
	v.SetDefault("commerce.payment.pendingStaleAfter", 30*time.Second)
	v.SetDefault("commerce.payment.wechat.enabled", false)
	v.SetDefault("commerce.payment.wechat.baseURL", "https://api.mch.weixin.qq.com")
	v.SetDefault("commerce.payment.wechat.appID", "")
	v.SetDefault("commerce.payment.wechat.merchantID", "")
	v.SetDefault("commerce.payment.wechat.merchantSerial", "")
	v.SetDefault("commerce.payment.wechat.merchantPrivateKeyFile", "")
	v.SetDefault("commerce.payment.wechat.apiV3KeyFile", "")
	v.SetDefault("commerce.payment.wechat.platformPublicKeyFiles", map[string]string{})
	v.SetDefault("commerce.payment.wechat.notifyURL", "")
	v.SetDefault("commerce.payment.wechat.refundNotifyURL", "")
	v.SetDefault("commerce.payment.wechat.callbackMaxAge", 5*time.Minute)
	v.SetDefault("commerce.payment.alipay.enabled", false)
	v.SetDefault("commerce.payment.alipay.gatewayURL", "https://openapi.alipay.com/gateway.do")
	v.SetDefault("commerce.payment.alipay.appID", "")
	v.SetDefault("commerce.payment.alipay.sellerID", "")
	v.SetDefault("commerce.payment.alipay.appPrivateKeyFile", "")
	v.SetDefault("commerce.payment.alipay.alipayPublicKeyFile", "")
	v.SetDefault("commerce.payment.alipay.notifyURL", "")
}
