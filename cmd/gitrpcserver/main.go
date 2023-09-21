// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harness/gitness/profiler"
	"github.com/harness/gitness/version"

	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	application = "gitrpcserver"
	description = "GitRPC is a GRPC server that exposes git via RPC."
)

func main() {
	// define new kingpin application with global entry point
	app := kingpin.New(application, description)
	app.Version(version.Version.String())

	var envFile string
	app.Action(func(*kingpin.ParseContext) error { return run(envFile) })
	app.Arg("envfile", "load the environment variable file").
		Default("").
		StringVar(&envFile)

	// trigger execution
	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func run(envFile string) error {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return fmt.Errorf("failed to load environment file: %w", err)
		}
	}

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// setup logger and inject into context
	setupLogger(config)
	log := log.Logger.With().Logger()
	ctx = log.WithContext(ctx)

	setupProfiler(config)

	system, err := initSystem()
	if err != nil {
		return fmt.Errorf("failed to init gitrpc server: %w", err)
	}

	// gCtx is canceled if any of the following occurs:
	// - any go routine launched with g encounters an error
	// - ctx is canceled
	g, gCtx := errgroup.WithContext(ctx)

	// start grpc server
	g.Go(system.grpcServer.Start)
	log.Info().Msg("grpc server started")

	gHTTP, shutdownHTTP := system.httpServer.ListenAndServe()
	g.Go(gHTTP.Wait)
	log.Info().Msgf("http server started")

	// wait until the error group context is done
	<-gCtx.Done()

	// restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Info().Msg("shutting down gracefully (press Ctrl+C again to force)")

	// shutdown servers gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdownTime)
	defer cancel()

	if rpcErr := system.grpcServer.Stop(); rpcErr != nil {
		log.Err(rpcErr).Msg("failed to shutdown grpc server gracefully")
	}

	if sErr := shutdownHTTP(shutdownCtx); sErr != nil {
		log.Err(sErr).Msg("failed to shutdown http server gracefully")
	}

	log.Info().Msg("wait for subroutines to complete")
	err = g.Wait()

	return err
}

// helper function configures the global logger from
// the loaded configuration.
func setupLogger(config Config) {
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

func setupProfiler(config Config) {
	profilerType, parsed := profiler.ParseType(config.Profiler.Type)
	if !parsed {
		log.Info().Msgf("No valid profiler so skipping profiling ['%s']", config.Profiler.Type)
		return
	}

	gitrpcProfiler, _ := profiler.New(profilerType)
	gitrpcProfiler.StartProfiling(config.Profiler.ServiceName, version.Version.String())
}
