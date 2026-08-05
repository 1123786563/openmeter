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

	// Settlement provides default billing profile parameters used by the
	// settlement service when resolving a customer's charge ID, currency,
	// feature key, and settlement mode. In production these are resolved per
	// customer from the billing/subscription stack; this config provides the
	// Phase 1 defaults.
	Settlement AIUsageSettlementConfiguration `yaml:"settlement"`

	// RateEntries are the default pricing table loaded into the pricing
	// service at startup. Each entry maps a resource to a credit rate.
	RateEntries []AIUsageRateEntryConfig `yaml:"rate_entries"`
}

// AIUsageSettlementConfiguration holds the default billing profile parameters.
type AIUsageSettlementConfiguration struct {
	DefaultChargeID   string `yaml:"default_charge_id"`
	DefaultFeatureKey string `yaml:"default_feature_key"`
	DefaultCurrency   string `yaml:"default_currency"`
}

// AIUsageRateEntryConfig configures a single pricing rate entry.
type AIUsageRateEntryConfig struct {
	ResourceCode   string `yaml:"resource_code"`
	Provider       string `yaml:"provider"`
	Model          string `yaml:"model"`
	CreditsPerUnit int64  `yaml:"credits_per_unit"`
	UnitSize       int64  `yaml:"unit_size"`
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
	v.SetDefault("aiUsage.enabled", false)
	v.SetDefault("aiUsage.signing.currentKeyId", "")
	v.SetDefault("aiUsage.signing.currentSeed", "")
	v.SetDefault("aiUsage.authorizationTtl", "5m")
	v.SetDefault("aiUsage.worker.leaseDuration", "30s")
	v.SetDefault("aiUsage.worker.batchSize", 50)
	v.SetDefault("aiUsage.settlement.defaultChargeId", "")
	v.SetDefault("aiUsage.settlement.defaultFeatureKey", "ai_usage")
	v.SetDefault("aiUsage.settlement.defaultCurrency", "USD")
}
