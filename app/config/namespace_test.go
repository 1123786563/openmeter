package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamespaceConfigurationValidate(t *testing.T) {
	tests := []struct {
		name   string
		config NamespaceConfiguration
		valid  bool
	}{
		{
			name: "empty allowlist keeps the static behavior",
			config: NamespaceConfiguration{
				Default: "default",
			},
			valid: true,
		},
		{
			name: "non-empty allowlist enables request-level selection",
			config: NamespaceConfiguration{
				Default:   "default",
				Allowlist: []string{"tenant-a", "tenant-b"},
			},
			valid: true,
		},
		{
			name:   "default namespace is required",
			config: NamespaceConfiguration{},
		},
		{
			name: "allowlist must not contain empty namespaces",
			config: NamespaceConfiguration{
				Default:   "default",
				Allowlist: []string{"tenant-a", ""},
			},
		},
		{
			name: "allowlist may repeat the default and itself",
			config: NamespaceConfiguration{
				Default:   "default",
				Allowlist: []string{"default", "tenant-a", "tenant-a"},
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

func TestNamespaceConfigurationNamespaces(t *testing.T) {
	t.Run("empty allowlist lists only the default", func(t *testing.T) {
		config := NamespaceConfiguration{Default: "default"}

		require.Equal(t, []string{"default"}, config.Namespaces())
	})

	t.Run("default is always included", func(t *testing.T) {
		config := NamespaceConfiguration{
			Default:   "default",
			Allowlist: []string{"tenant-a", "tenant-b"},
		}

		require.Equal(t, []string{"default", "tenant-a", "tenant-b"}, config.Namespaces())
	})

	t.Run("duplicates with the default are deduplicated", func(t *testing.T) {
		config := NamespaceConfiguration{
			Default:   "default",
			Allowlist: []string{"default", "tenant-a", "tenant-a"},
		}

		require.Equal(t, []string{"default", "tenant-a"}, config.Namespaces())
	})

	t.Run("result is sorted for stable output", func(t *testing.T) {
		config := NamespaceConfiguration{
			Default:   "zeta",
			Allowlist: []string{"alpha", "mid"},
		}

		require.Equal(t, []string{"alpha", "mid", "zeta"}, config.Namespaces())
	})
}
