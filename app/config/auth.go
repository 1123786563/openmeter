package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/viper"
)

// OIDCConfiguration configures the management-plane OIDC client against an
// external identity provider such as Casdoor.
type OIDCConfiguration struct {
	Enabled bool `mapstructure:"enabled"`

	// Issuer is the OIDC issuer URL of the provider. For Casdoor this is the
	// deployment's base URL; discovery happens at {issuer}/.well-known/openid-configuration.
	Issuer string `mapstructure:"issuer"`

	// ClientID is the OAuth2 client identifier registered in the provider.
	ClientID string `mapstructure:"clientId"`

	// ClientSecret is the OAuth2 client secret. It must be supplied through the
	// AUTH_OIDC_CLIENTSECRET environment variable (or a secret service), never
	// committed to configuration files.
	ClientSecret string `mapstructure:"clientSecret"`

	// RedirectURL is this server's OAuth2 callback endpoint as registered in
	// the provider, for example http://localhost:8888/auth/oidc/callback.
	RedirectURL string `mapstructure:"redirectURL"`

	// DashboardURL is the front-end URL the browser is redirected to after a
	// successful login. The session token is appended in the URL fragment,
	// for example http://localhost:5173/auth/callback#token=...
	DashboardURL string `mapstructure:"dashboardURL"`
}

// AuthConfiguration configures management-plane authentication.
type AuthConfiguration struct {
	OIDC OIDCConfiguration

	// TokenSecret signs the session JWTs OpenMeter issues after a successful
	// OIDC login. It must be supplied through the AUTH_TOKENSECRET environment
	// variable (or a secret service), never committed to configuration files.
	TokenSecret string `mapstructure:"tokenSecret"`

	// TokenExpiration is how long an issued session token stays valid.
	TokenExpiration time.Duration `mapstructure:"tokenExpiration"`
}

// Validate validates the configuration.
func (c AuthConfiguration) Validate() error {
	if !c.OIDC.Enabled {
		return nil
	}

	var errs []error

	if err := validateHTTPURL("issuer", c.OIDC.Issuer); err != nil {
		errs = append(errs, err)
	}

	if c.OIDC.ClientID == "" {
		errs = append(errs, errors.New("oidc clientId is required"))
	}

	if c.OIDC.ClientSecret == "" {
		errs = append(errs, errors.New("oidc clientSecret is required"))
	}

	if err := validateHTTPURL("oidc redirectURL", c.OIDC.RedirectURL); err != nil {
		errs = append(errs, err)
	}

	if err := validateHTTPURL("oidc dashboardURL", c.OIDC.DashboardURL); err != nil {
		errs = append(errs, err)
	}

	if c.TokenSecret == "" {
		errs = append(errs, errors.New("tokenSecret is required"))
	}

	if c.TokenExpiration.Seconds() == 0 {
		errs = append(errs, errors.New("token duration is required"))
	}

	return errors.Join(errs...)
}

func validateHTTPURL(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be a valid http(s) URL", field)
	}

	return nil
}

// ConfigureAuth configures some defaults in the Viper instance.
func ConfigureAuth(v *viper.Viper) {
	v.SetDefault("auth.oidc.enabled", false)
	v.SetDefault("auth.oidc.issuer", "")
	v.SetDefault("auth.oidc.clientId", "")
	v.SetDefault("auth.oidc.clientSecret", "")
	v.SetDefault("auth.oidc.redirectURL", "")
	v.SetDefault("auth.oidc.dashboardURL", "")
	v.SetDefault("auth.tokenSecret", "")
	v.SetDefault("auth.tokenExpiration", "720h")
}
