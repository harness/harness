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
	"golang.org/x/sync/errgroup"

	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// GraceFullShutdownTime defines the max time we wait when shutting down a server.
	// 5min should be enough for most git clones to complete.
	GraceFullShutdownTime = 300 * time.Second
)

type command struct {
	envfile string
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
	config, err := load()
	if err != nil {
		return fmt.Errorf("encountered an error while loading configuration: %w", err)
	}

	// configure the log level
	setupLogger(config)

	// add logger to context
	log := log.Logger.With().Logger()
	ctx = log.WithContext(ctx)

	// initialize system
	system, err := initSystem(ctx, config)
	if err != nil {
		return fmt.Errorf("encountered an error while wiring the system: %w", err)
	}

	// bootstrap the system
	err = system.bootstrap(ctx)
	if err != nil {
		return fmt.Errorf("encountered an error while bootstrapping the system: %w", err)
	}

	// collects all go routines - gCTX cancels if any go routine encounters an error
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

	// start the purge routine.
	g.Go(func() error {
		system.nightly.Run(gCtx)
		return nil
	})
	log.Info().Msg("nightly subroutine started")

	// start grpc server
	g.Go(system.gitRPCServer.Start)

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

	if rpcErr := system.gitRPCServer.Stop(); rpcErr != nil {
		log.Err(rpcErr).Msg("failed to shutdown grpc server gracefully")
	}

	log.Info().Msg("wait for subroutines to complete")
	err = g.Wait()

	return err
}

// helper function configures the global logger from
// the loaded configuration.
func setupLogger(config *types.Config) {
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
func Register(app *kingpin.Application) {
	c := new(command)

	cmd := app.Command("server", "starts the server").
		Action(c.run)

	cmd.Arg("envfile", "load the environment variable file").
		Default("").
		StringVar(&c.envfile)
}
