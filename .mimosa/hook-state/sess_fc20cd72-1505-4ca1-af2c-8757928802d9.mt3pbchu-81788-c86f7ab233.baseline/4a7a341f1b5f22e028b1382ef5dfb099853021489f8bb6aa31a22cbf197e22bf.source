package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestCreditReservationConfigurationDefaults(t *testing.T) {
	v := viper.New()
	ConfigureCreditReservation(v)

	require.False(t, v.GetBool("creditReservation.enabled"))
	require.Equal(t, "5m", v.GetString("creditReservation.authorizationTTL"))
	require.Equal(t, "10m", v.GetString("creditReservation.executionDeadline"))
	require.Equal(t, "1h", v.GetString("creditReservation.unknownManualReviewAfter"))
	require.Equal(t, "5s", v.GetString("creditReservation.worker.pollInterval"))
	require.Equal(t, "30s", v.GetString("creditReservation.worker.leaseDuration"))
	require.Equal(t, 50, v.GetInt("creditReservation.worker.batchSize"))
	require.Equal(t, 3, v.GetInt("creditReservation.worker.maxClaimCount"))
}

func TestCreditReservationConfigurationValidate(t *testing.T) {
	valid := func() CreditReservationConfiguration {
		return CreditReservationConfiguration{
			Enabled:                  true,
			AuthorizationTTL:         "5m",
			ExecutionDeadline:        "10m",
			UnknownManualReviewAfter: "1h",
			Worker: CreditReservationWorkerConfiguration{
				PollInterval:  "5s",
				LeaseDuration: "30s",
				BatchSize:     50,
				MaxClaimCount: 3,
			},
		}
	}

	t.Run("disabled accepts zero values", func(t *testing.T) {
		require.NoError(t, CreditReservationConfiguration{}.Validate())
	})

	t.Run("enabled accepts bounded worker settings", func(t *testing.T) {
		require.NoError(t, valid().Validate())
	})

	for _, test := range []struct {
		name string
		edit func(*CreditReservationConfiguration)
		want string
	}{
		{
			name: "authorization ttl is bounded",
			edit: func(c *CreditReservationConfiguration) { c.AuthorizationTTL = "25h" },
			want: "authorizationTTL must be between 1s and 24h",
		},
		{
			name: "execution deadline is bounded",
			edit: func(c *CreditReservationConfiguration) { c.ExecutionDeadline = "0s" },
			want: "executionDeadline must be between 1s and 24h",
		},
		{
			name: "unknown manual review follows execution deadline",
			edit: func(c *CreditReservationConfiguration) { c.UnknownManualReviewAfter = "5m" },
			want: "unknownManualReviewAfter must not be shorter than executionDeadline",
		},
		{
			name: "poll interval must be a positive duration",
			edit: func(c *CreditReservationConfiguration) { c.Worker.PollInterval = "0s" },
			want: "pollInterval must be a positive duration",
		},
		{
			name: "lease duration must be a positive duration",
			edit: func(c *CreditReservationConfiguration) { c.Worker.LeaseDuration = "not-a-duration" },
			want: "leaseDuration must be a positive duration",
		},
		{
			name: "batch size is bounded",
			edit: func(c *CreditReservationConfiguration) { c.Worker.BatchSize = 501 },
			want: "batchSize must be between 1 and 500",
		},
		{
			name: "claim count is bounded",
			edit: func(c *CreditReservationConfiguration) { c.Worker.MaxClaimCount = 11 },
			want: "maxClaimCount must be between 1 and 10",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid()
			test.edit(&cfg)

			require.ErrorContains(t, cfg.Validate(), test.want)
		})
	}
}
