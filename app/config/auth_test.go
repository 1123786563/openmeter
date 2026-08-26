package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOIDCAuthConfigurationValidate(t *testing.T) {
	tests := []struct {
		name   string
		config OIDCAuthConfiguration
		valid  bool
	}{
		{
			name:   "disabled ignores empty issuer and jwks url",
			config: OIDCAuthConfiguration{},
			valid:  true,
		},
		{
			name: "enabled requires issuer and jwks url",
			config: OIDCAuthConfiguration{
				Enabled: true,
				JwksURL: "https://casdoor.example.com/api/certs",
			},
		},
		{
			name: "enabled requires the organization allowlist",
			config: OIDCAuthConfiguration{
				Enabled: true,
				Issuer:  "https://casdoor.example.com",
				JwksURL: "https://casdoor.example.com/api/certs",
			},
		},
		{
			name: "enabled with issuer, jwks url and organization allowlist",
			config: OIDCAuthConfiguration{
				Enabled:              true,
				Issuer:               "https://casdoor.example.com",
				JwksURL:              "https://casdoor.example.com/api/certs",
				AllowedOrganizations: []string{"built-in"},
			},
			valid: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()

			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
