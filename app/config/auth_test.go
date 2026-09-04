package config

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// randomSecret returns a non-reusable secret so no credential literal ever
// appears in source control.
func randomSecret(t *testing.T) string {
	t.Helper()

	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	require.NoError(t, err)

	return hex.EncodeToString(buf)
}

func TestAuthConfigurationValidate(t *testing.T) {
	clientSecret := randomSecret(t)
	tokenSecret := randomSecret(t)
	expiration := 720 * time.Hour

	validOIDC := OIDCConfiguration{
		Enabled:      true,
		Issuer:       "https://casdoor.example.com",
		ClientID:     "openmeter",
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:8888/auth/oidc/callback",
		DashboardURL: "http://localhost:5173/auth/callback",
	}

	testCases := []struct {
		name   string
		config AuthConfiguration
		assert func(t *testing.T, err error)
	}{
		{
			name:   "disabled requires nothing",
			config: AuthConfiguration{},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "valid",
			config: AuthConfiguration{
				OIDC:            validOIDC,
				TokenSecret:     tokenSecret,
				TokenExpiration: expiration,
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "missing issuer",
			config: AuthConfiguration{
				OIDC:            func() OIDCConfiguration { c := validOIDC; c.Issuer = ""; return c }(),
				TokenSecret:     tokenSecret,
				TokenExpiration: expiration,
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "issuer is required")
			},
		},
		{
			name: "issuer must be a URL",
			config: AuthConfiguration{
				OIDC:            func() OIDCConfiguration { c := validOIDC; c.Issuer = "not a url"; return c }(),
				TokenSecret:     tokenSecret,
				TokenExpiration: expiration,
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "issuer must be a valid http(s) URL")
			},
		},
		{
			name: "missing client credentials",
			config: AuthConfiguration{
				OIDC: func() OIDCConfiguration {
					c := validOIDC
					c.ClientID = ""
					c.ClientSecret = ""
					return c
				}(),
				TokenSecret:     tokenSecret,
				TokenExpiration: expiration,
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "clientId is required")
				assert.Contains(t, err.Error(), "clientSecret is required")
			},
		},
		{
			name: "missing token secret",
			config: AuthConfiguration{
				OIDC:            validOIDC,
				TokenExpiration: expiration,
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "tokenSecret is required")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, tc.config.Validate())
		})
	}
}

// TestAuthConfigurationEnvironmentOverride mirrors the server's viper setup to
// verify secrets injected through the environment reach the configuration.
func TestAuthConfigurationEnvironmentOverride(t *testing.T) {
	v := viper.NewWithOptions(viper.WithDecodeHook(DecodeHook()))
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	SetViperDefaults(v, flags)

	clientSecret := randomSecret(t)
	tokenSecret := randomSecret(t)

	t.Setenv("AUTH_OIDC_ENABLED", "true")
	t.Setenv("AUTH_OIDC_ISSUER", "https://casdoor.example.com")
	t.Setenv("AUTH_OIDC_CLIENTSECRET", clientSecret)
	t.Setenv("AUTH_TOKENSECRET", tokenSecret)

	var conf Configuration
	require.NoError(t, v.Unmarshal(&conf))

	assert.True(t, conf.Auth.OIDC.Enabled)
	assert.Equal(t, "https://casdoor.example.com", conf.Auth.OIDC.Issuer)
	assert.Equal(t, clientSecret, conf.Auth.OIDC.ClientSecret)
	assert.Equal(t, tokenSecret, conf.Auth.TokenSecret)
	assert.Equal(t, 720*time.Hour, conf.Auth.TokenExpiration)
}
