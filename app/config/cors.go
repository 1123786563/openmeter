package config

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/models"
)

// CORSConfiguration configures CORS on the main API surface (the v1 and v3 API
// groups). An empty AllowedOrigins disables CORS entirely: no CORS headers are
// added and preflight requests are not answered. A "*" origin allows any
// origin and is served without credentials (the API authenticates with a
// bearer token instead of cookies, so wildcard responses never opt into
// credentialed requests).
type CORSConfiguration struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

func (c CORSConfiguration) Validate() error {
	var errs []error

	for _, origin := range c.AllowedOrigins {
		if origin == "" {
			errs = append(errs, errors.New("allowed origins must not contain empty values"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ConfigureCORS configures defaults in the Viper instance.
func ConfigureCORS(v *viper.Viper) {
	v.SetDefault("cors.allowed_origins", nil)
}
