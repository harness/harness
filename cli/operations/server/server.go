// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harness/gitness/app/pipeline/logger"
	"github.com/harness/gitness/profiler"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/version"

	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"
)

type command struct {
	envfile     string
	enableCI    bool
	initializer func(context.Context, *types.Config) (*System, error)
}

func (c *command) run(*kingpin.ParseContext) error {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// load environment variables from file.
	// no error handling needed when file is not present
	_ = godotenv.Load(c.envfile)

	// create the system configuration store by loading
	// data from the environment.
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("encountered an error while loading configuration: %w", err)
	}

	// configure the log level
	SetupLogger(config)

	// configure profiler
	SetupProfiler(config)

	// add logger to context
	log := log.Logger.With().Logger()
	ctx = log.WithContext(ctx)

	// initialize system
	system, err := c.initializer(ctx, config)
	if err != nil {
		return fmt.Errorf("encountered an error while wiring the system: %w", err)
	}

	// bootstrap the system
	err = system.bootstrap(ctx)
	if err != nil {
		return fmt.Errorf("encountered an error while bootstrapping the system: %w", err)
	}

	// gCtx is canceled if any of the following occurs:
	// - any go routine launched with g encounters an error
	// - ctx is canceled
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		// initialize metric collector
		if system.services.MetricCollector != nil {
			if err := system.services.MetricCollector.Register(gCtx); err != nil {
				log.Error().Err(err).Msg("failed to register metric collector")
				return err
			}
		}

		if system.services.RepoSizeCalculator != nil {
			if err := system.services.RepoSizeCalculator.Register(gCtx); err != nil {
				log.Error().Err(err).Msg("failed to register repo size calculator")
				return err
			}
		}

		if err := system.services.Cleanup.Register(gCtx); err != nil {
			log.Error().Err(err).Msg("failed to register cleanup service")
			return err
		}

		return system.services.JobScheduler.Run(gCtx)
	})

	// start server
	gHTTP, shutdownHTTP := system.server.ListenAndServe()
	g.Go(gHTTP.Wait)
	if c.enableCI {
		// start populating plugins
		g.Go(func() error {
			err := system.resolverManager.Populate(ctx)
			if err != nil {
				log.Error().Err(err).Msg("could not populate plugins")
			}
			return nil
		})
		// start poller for CI build executions.
		g.Go(func() error {
			system.poller.Poll(
				logger.WithWrappedZerolog(ctx),
				config.CI.ParallelWorkers,
			)
			return nil
		})
	}

	if config.SSH.Enable {
		g.Go(func() error {
			log.Err(system.sshServer.ListenAndServe()).Send()
			return nil
		})
	}

	log.Info().
		Int("port", config.Server.HTTP.Port).
		Str("revision", version.GitCommit).
		Str("repository", version.GitRepository).
		Stringer("version", version.Version).
		Msg("server started")

	// wait until the error group context is done
	<-gCtx.Done()

	// restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Info().Msg("shutting down gracefully (press Ctrl+C again to force)")

	// shutdown servers gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdownTime)
	defer cancel()

	if sErr := shutdownHTTP(shutdownCtx); sErr != nil {
		log.Err(sErr).Msg("failed to shutdown http server gracefully")
	}

	if config.SSH.Enable {
		if err := system.sshServer.Shutdown(shutdownCtx); err != nil {
			log.Err(err).Msg("failed to shutdown ssh server gracefully")
		}
	}

	system.services.JobScheduler.WaitJobsDone(shutdownCtx)

	log.Info().Msg("wait for subroutines to complete")
	err = g.Wait()

	return err
}

// SetupLogger configures the global logger from the loaded configuration.
func SetupLogger(config *types.Config) {
	// configure the log level
	switch {
	case config.Trace:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case config.Debug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// configure time format (ignored if running in terminal)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// if the terminal is a tty we should output the
	// logs in pretty format
	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.Logger = log.Output(
			zerolog.ConsoleWriter{
				Out:        os.Stderr,
				NoColor:    false,
				TimeFormat: "15:04:05.999",
			},
		)
	}
}

func SetupProfiler(config *types.Config) {
	profilerType, parsed := profiler.ParseType(config.Profiler.Type)
	if !parsed {
		log.Info().Msgf("No valid profiler so skipping profiling ['%s']", config.Profiler.Type)
		return
	}

	gitnessProfiler, _ := profiler.New(profilerType)
	gitnessProfiler.StartProfiling(config.Profiler.ServiceName, version.Version.String())
}

// Register the server command.
func Register(app *kingpin.Application, initializer func(context.Context, *types.Config) (*System, error)) {
	c := new(command)
	c.initializer = initializer

	cmd := app.Command("server", "starts the server").
		Action(c.run)

	cmd.Arg("envfile", "load the environment variable file").
		Default("").
		StringVar(&c.envfile)

	cmd.Flag("enable-ci", "start ci runners for build executions").
		Default("true").
		Envar("ENABLE_CI").
		BoolVar(&c.enableCI)
}
