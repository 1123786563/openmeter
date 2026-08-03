package config

import (
	"errors"

	"github.com/spf13/viper"
)

// AIUsageConfiguration configures the AI Usage billing subsystem.
type AIUsageConfiguration struct {
	Enabled bool `yaml:"enabled"`

	Signing AIUsageSigningConfiguration `yaml:"signing"`

	AuthorizationTTL string `yaml:"authorization_ttl"`

	Worker AIUsageWorkerConfiguration `yaml:"worker"`
}

// AIUsageSigningConfiguration configures the Ed25519 signing key used for
// runtime authorization packages.
type AIUsageSigningConfiguration struct {
	CurrentKeyID string `yaml:"current_key_id"`
	CurrentSeed  string `yaml:"current_seed"`
}

// AIUsageWorkerConfiguration configures the outbox relay worker.
type AIUsageWorkerConfiguration struct {
	LeaseDuration string `yaml:"lease_duration"`
	BatchSize     int    `yaml:"batch_size"`
}

func (c AIUsageConfiguration) Validate() error {
	var errs []error

	if !c.Enabled {
		return errors.Join(errs...)
	}

	if c.Signing.CurrentKeyID == "" {
		errs = append(errs, errors.New("ai_usage.signing.current_key_id is required when ai_usage is enabled"))
	}

	if c.Worker.BatchSize <= 0 {
		errs = append(errs, errors.New("ai_usage.worker.batch_size must be positive when ai_usage is enabled"))
	}

	return errors.Join(errs...)
}

// ConfigureAIUsage sets viper defaults for the ai_usage configuration keys.
func ConfigureAIUsage(v *viper.Viper) {
	v.SetDefault("ai_usage.enabled", false)
	v.SetDefault("ai_usage.signing.current_key_id", "")
	v.SetDefault("ai_usage.signing.current_seed", "")
	v.SetDefault("ai_usage.authorization_ttl", "5m")
	v.SetDefault("ai_usage.worker.lease_duration", "30s")
	v.SetDefault("ai_usage.worker.batch_size", 50)
}
