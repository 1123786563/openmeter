package config

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/models"
)

// AuthConfiguration configures API authentication.
type AuthConfiguration struct {
	OIDC OIDCAuthConfiguration `mapstructure:"oidc"`
}

var _ models.Validator = (*OIDCAuthConfiguration)(nil)

// OIDCAuthConfiguration configures Casdoor OIDC bearer authentication in front
// of the v1 and v3 APIs. See docs/adr/0001-casdoor-oidc-auth-middleware.md.
//
// Enabled is fail-closed by default: non-admin environments (local development,
// quickstart, e2e) must opt out explicitly because they have no Casdoor to talk to.
type OIDCAuthConfiguration struct {
	Enabled bool `mapstructure:"enabled"`

	// Issuer is the expected JWT iss claim of the Casdoor instance.
	Issuer string `mapstructure:"issuer"`

	// JwksURL is the JWKS endpoint serving Casdoor's public signing keys,
	// typically {endpoint}/api/certs.
	JwksURL string `mapstructure:"jwks_url"`

	// Audience validates the JWT aud claim when non-empty; empty skips the check.
	Audience string `mapstructure:"audience"`

	// AllowedOrganizations restricts access to tokens whose organization claim
	// is listed here. Required when enabled: it is the authorization boundary
	// between "authenticated" and "trusted".
	AllowedOrganizations []string `mapstructure:"allowed_organizations"`

	// OrganizationClaim names the token claim carrying the organization.
	// Casdoor defaults to "owner".
	OrganizationClaim string `mapstructure:"organization_claim"`

	// ViewerRoles are token roles mapped to the read-only Viewer role.
	ViewerRoles []string `mapstructure:"viewer_roles"`

	// OperatorRoles are token roles mapped to the read-write Operator role.
	OperatorRoles []string `mapstructure:"operator_roles"`

	// RoleClaim names the token claim carrying role names. Casdoor defaults to "roles".
	RoleClaim string `mapstructure:"role_claim"`
}

// Validate validates the configuration.
func (c OIDCAuthConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error

	if c.Issuer == "" {
		errs = append(errs, errors.New("issuer is required"))
	}

	if c.JwksURL == "" {
		errs = append(errs, errors.New("jwks url is required"))
	}

	// The organization allowlist is the outermost authorization boundary:
	// without it any token the issuer signs maps to full Operator access, so a
	// deployment enabling authentication must name the organizations it trusts.
	if len(c.AllowedOrganizations) == 0 {
		errs = append(errs, errors.New("allowed_organizations is required (the organization allowlist is the authorization boundary)"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ConfigureAuth configures defaults in the Viper instance.
func ConfigureAuth(v *viper.Viper) {
	v.SetDefault("auth.oidc.enabled", true)
	v.SetDefault("auth.oidc.issuer", "")
	v.SetDefault("auth.oidc.jwks_url", "")
	v.SetDefault("auth.oidc.audience", "")
	v.SetDefault("auth.oidc.allowed_organizations", nil)
	v.SetDefault("auth.oidc.organization_claim", "owner")
	v.SetDefault("auth.oidc.viewer_roles", nil)
	v.SetDefault("auth.oidc.operator_roles", nil)
	v.SetDefault("auth.oidc.role_claim", "roles")
}
