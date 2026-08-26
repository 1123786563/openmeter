package config

import (
	"errors"
	"maps"
	"slices"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/models"
)

// NamespaceConfiguration configures the tenancy model. A non-empty Allowlist
// opts into request-level namespace selection: callers may select a namespace
// via the X-Namespace header, constrained to the allowlist (the default
// namespace is always allowed). An empty Allowlist keeps the static behavior
// where every request serves the default namespace.
type NamespaceConfiguration struct {
	Default           string
	DisableManagement bool
	Allowlist         []string `mapstructure:"allowlist"`
}

func (c NamespaceConfiguration) Validate() error {
	var errs []error

	if c.Default == "" {
		errs = append(errs, errors.New("default namespace is required"))
	}

	for _, ns := range c.Allowlist {
		if ns == "" {
			errs = append(errs, errors.New("allowlist must not contain empty namespaces"))
		}
	}

	// Duplicates (inside the allowlist, or with the default namespace) are
	// allowed: only set membership matters, and the namespaces listing
	// deduplicates before responding.

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Namespaces returns the selectable namespaces: the default namespace plus the
// allowlist, deduplicated and sorted. It is the data source of the
// GET /api/v3/openmeter/namespaces endpoint and stays stable for a given
// configuration.
func (c NamespaceConfiguration) Namespaces() []string {
	if len(c.Allowlist) == 0 {
		return []string{c.Default}
	}

	unique := make(map[string]struct{}, len(c.Allowlist)+1)
	unique[c.Default] = struct{}{}
	for _, ns := range c.Allowlist {
		unique[ns] = struct{}{}
	}

	return slices.Sorted(maps.Keys(unique))
}

// ConfigureNamespace configures some defaults in the Viper instance.
func ConfigureNamespace(v *viper.Viper) {
	v.SetDefault("namespace.default", "default")
	v.SetDefault("namespace.disableManagement", false)
	v.SetDefault("namespace.allowlist", nil)
}
