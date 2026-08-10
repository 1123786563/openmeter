package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const (
	creditReservationWorkerMaxBatchSize  = 500
	creditReservationWorkerMaxClaimCount = 10
	creditReservationMinimumDuration     = time.Second
	creditReservationMaximumTTL          = 24 * time.Hour
	creditReservationMinimumReviewAfter  = time.Minute
	creditReservationMaximumReviewAfter  = 7 * 24 * time.Hour
)

// CreditReservationConfiguration controls the opt-in reservation lifecycle.
// It is disabled by default so a deployment cannot expose the unstable API or
// start its delivery loop until its operators explicitly enable it.
type CreditReservationConfiguration struct {
	Enabled                  bool                                 `yaml:"enabled"`
	AuthorizationTTL         string                               `yaml:"authorizationTTL"`
	ExecutionDeadline        string                               `yaml:"executionDeadline"`
	UnknownManualReviewAfter string                               `yaml:"unknownManualReviewAfter"`
	Worker                   CreditReservationWorkerConfiguration `yaml:"worker"`
}

// CreditReservationWorkerConfiguration bounds the work claimed during one
// outbox poll. Bounded values make a misconfiguration unable to monopolize a
// database transaction or create unbounded retry churn.
type CreditReservationWorkerConfiguration struct {
	PollInterval  string `yaml:"pollInterval"`
	LeaseDuration string `yaml:"leaseDuration"`
	BatchSize     int    `yaml:"batchSize"`
	MaxClaimCount int    `yaml:"maxClaimCount"`
}

func (c CreditReservationConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error

	_, err := validateDurationInRange(c.AuthorizationTTL, creditReservationMinimumDuration, creditReservationMaximumTTL)
	if err != nil {
		errs = append(errs, fmt.Errorf("creditReservation.authorizationTTL must be between 1s and 24h: %w", err))
	}

	executionDeadline, err := validateDurationInRange(c.ExecutionDeadline, creditReservationMinimumDuration, creditReservationMaximumTTL)
	if err != nil {
		errs = append(errs, fmt.Errorf("creditReservation.executionDeadline must be between 1s and 24h: %w", err))
	}

	unknownManualReviewAfter, err := validateDurationInRange(c.UnknownManualReviewAfter, creditReservationMinimumReviewAfter, creditReservationMaximumReviewAfter)
	if err != nil {
		errs = append(errs, fmt.Errorf("creditReservation.unknownManualReviewAfter must be between 1m and 168h: %w", err))
	}
	if err == nil && executionDeadline > 0 && unknownManualReviewAfter < executionDeadline {
		errs = append(errs, errors.New("creditReservation.unknownManualReviewAfter must not be shorter than executionDeadline"))
	}

	if err := validatePositiveDuration(c.Worker.PollInterval); err != nil {
		errs = append(errs, fmt.Errorf("creditReservation.worker.pollInterval must be a positive duration: %w", err))
	}

	if err := validatePositiveDuration(c.Worker.LeaseDuration); err != nil {
		errs = append(errs, fmt.Errorf("creditReservation.worker.leaseDuration must be a positive duration: %w", err))
	}

	if c.Worker.BatchSize < 1 || c.Worker.BatchSize > creditReservationWorkerMaxBatchSize {
		errs = append(errs, fmt.Errorf("creditReservation.worker.batchSize must be between 1 and %d", creditReservationWorkerMaxBatchSize))
	}

	if c.Worker.MaxClaimCount < 1 || c.Worker.MaxClaimCount > creditReservationWorkerMaxClaimCount {
		errs = append(errs, fmt.Errorf("creditReservation.worker.maxClaimCount must be between 1 and %d", creditReservationWorkerMaxClaimCount))
	}

	return errors.Join(errs...)
}

func validatePositiveDuration(value string) error {
	_, err := validateDurationInRange(value, time.Nanosecond, time.Duration(1<<63-1))
	return err
}

func validateDurationInRange(value string, minimum, maximum time.Duration) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}
	if duration < minimum || duration > maximum {
		return 0, fmt.Errorf("must be between %s and %s", minimum, maximum)
	}
	return duration, nil
}

// ConfigureCreditReservation sets safe, bounded defaults for the optional
// credit reservation lifecycle.
func ConfigureCreditReservation(v *viper.Viper) {
	v.SetDefault("creditReservation.enabled", false)
	v.SetDefault("creditReservation.authorizationTTL", "5m")
	v.SetDefault("creditReservation.executionDeadline", "10m")
	v.SetDefault("creditReservation.unknownManualReviewAfter", "1h")
	v.SetDefault("creditReservation.worker.pollInterval", "5s")
	v.SetDefault("creditReservation.worker.leaseDuration", "30s")
	v.SetDefault("creditReservation.worker.batchSize", 50)
	v.SetDefault("creditReservation.worker.maxClaimCount", 3)
}
