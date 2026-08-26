package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCORSConfigurationValidate(t *testing.T) {
	tests := []struct {
		name   string
		config CORSConfiguration
		valid  bool
	}{
		{
			name:   "empty origins disables CORS",
			config: CORSConfiguration{},
			valid:  true,
		},
		{
			name: "specific origins",
			config: CORSConfiguration{
				AllowedOrigins: []string{"https://admin.example.com"},
			},
			valid: true,
		},
		{
			name: "wildcard origin",
			config: CORSConfiguration{
				AllowedOrigins: []string{"*"},
			},
			valid: true,
		},
		{
			name: "empty origin is rejected",
			config: CORSConfiguration{
				AllowedOrigins: []string{"https://admin.example.com", ""},
			},
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
