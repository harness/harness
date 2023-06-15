// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/version"

	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// GraceFullShutdownTime defines the max time we wait when shutting down a server.
	// 5min should be enough for most git clones to complete.
	GraceFullShutdownTime = 300 * time.Second
)

type command struct {
	envfile      string
	enableGitRPC bool
	initializer  func(context.Context, *types.Config) (*System, error)
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

	// start server
	gHTTP, shutdownHTTP := system.server.ListenAndServe()
	g.Go(gHTTP.Wait)
	log.Info().
		Str("port", config.Server.HTTP.Bind).
		Str("revision", version.GitCommit).
		Str("repository", version.GitRepository).
		Stringer("version", version.Version).
		Msg("server started")

	if c.enableGitRPC {
		// start grpc server
		g.Go(system.gitRPCServer.Start)
		log.Info().Msg("gitrpc server started")

		// run the gitrpc cron jobs
		g.Go(func() error {
			return system.gitRPCCronMngr.Run(ctx)
		})
		log.Info().Msg("gitrpc cron manager subroutine started")
	}

	// wait until the error group context is done
	<-gCtx.Done()

	// restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Info().Msg("shutting down gracefully (press Ctrl+C again to force)")

	// shutdown servers gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), GraceFullShutdownTime)
	defer cancel()

	if sErr := shutdownHTTP(shutdownCtx); sErr != nil {
		log.Err(sErr).Msg("failed to shutdown http server gracefully")
	}

	if c.enableGitRPC {
		if rpcErr := system.gitRPCServer.Stop(); rpcErr != nil {
			log.Err(rpcErr).Msg("failed to shutdown grpc server gracefully")
		}
	}

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

// Register the server command.
func Register(app *kingpin.Application, initializer func(context.Context, *types.Config) (*System, error)) {
	c := new(command)
	c.initializer = initializer

	cmd := app.Command("server", "starts the server").
		Action(c.run)

	cmd.Arg("envfile", "load the environment variable file").
		Default("").
		StringVar(&c.envfile)

	cmd.Flag("enable-gitrpc", "start the gitrpc server").
		Default("true").
		Envar("ENABLE_GITRPC").
		BoolVar(&c.enableGitRPC)
}
