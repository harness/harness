// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"context"
	"os"

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
	envfile string
}

func (c *command) run(*kingpin.ParseContext) error {
	// load environment variables from file.
	err := godotenv.Load(c.envfile)
	if err != nil {
		return err
	}

	// create the system configuration store by loading
	// data from the environment.
	config, err := load()
	if err != nil {
		log.Fatal().Err(err).
			Msg("cannot load configuration")
	}

	// configure the log level
	setupLogger(config)

	system, err := initSystem(config)
	if err != nil {
		log.Fatal().Err(err).
			Msg("cannot boot server")
	}

	var g errgroup.Group

	// starts the http server.
	g.Go(func() error {
		log.Info().
			Str("port", config.Server.Bind).
			Str("revision", version.GitCommit).
			Str("repository", version.GitRepository).
			Stringer("version", version.Version).
			Msg("server started")
		return system.server.ListenAndServe(context.Background())
	})

	// start the purge routine.
	g.Go(func() error {
		log.Debug().Msg("starting the nightly subroutine")
		system.nightly.Run(context.Background())
		return nil
	})

	return g.Wait()
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

	// if the terminal is a tty we should output the
	// logs in pretty format
	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.Logger = log.Output(
			zerolog.ConsoleWriter{
				Out:     os.Stderr,
				NoColor: false,
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
