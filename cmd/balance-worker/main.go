package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/log"
)

func main() {
	defer log.PanicLogger(log.WithExit)

	v, flags := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook())), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)
	ctx := context.Background()

	config.SetViperDefaults(v, flags)

	flags.String("config", "", "Configuration file")
	flags.Bool("version", false, "Show version information")
	flags.Bool("validate", false, "Validate configuration and exit")

	_ = flags.Parse(os.Args[1:])

	if v, _ := flags.GetBool("version"); v {
		fmt.Printf("%s version %s (%s) built on %s\n", "Open Meter", version, revision, revisionDate)

		os.Exit(0)
	}

	if c, _ := flags.GetString("config"); c != "" {
		v.SetConfigFile(c)
	}

	err := v.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		panic(err)
	}

	var conf config.Configuration
	err = v.Unmarshal(&conf)
	if err != nil {
		panic(err)
	}

	err = conf.Validate()
	if err != nil {
		println("configuration error:")
		println(err.Error())
		os.Exit(1)
	}

	if v, _ := flags.GetBool("validate"); v {
		os.Exit(0)
	}

	app, cleanup, err := initializeApplication(ctx, conf)
	if err != nil {
		slog.Error("failed to initialize application", "error", err)

		// Call cleanup function is may not set yet
		if cleanup != nil {
			cleanup()
		}

		os.Exit(1)
	}
	defer cleanup()

	app.SetGlobals()

	logger := app.Logger

	// Startup and serving funnel through a single error exit so deferred
	// cleanup still runs: os.Exit skips deferred calls, so it is called
	// explicitly before exiting.
	if err := func() error {
		// Validate service prerequisites
		if err := app.Migrate(ctx); err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		return app.Run()
	}(); err != nil {
		logger.Error("application stopped due to error", "error", err)
		cleanup()
		os.Exit(1)
	}
}
