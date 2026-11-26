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
	"errors"
	"io"
	"testing"

	"github.com/harness/gitness/app/services"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gopkg.in/alecthomas/kingpin.v2"
)

var errGc5Initializer = errors.New("gc5 initializer failure")

func gc5NewServerApp(initializer func(context.Context, *types.Config) (*System, error)) *kingpin.Application {
	app := kingpin.New("gc5", "test")
	app.Terminate(nil)
	app.ErrorWriter(io.Discard)
	app.UsageWriter(io.Discard)
	Register(app, initializer)

	return app
}

func gc5RestoreLogLevel(t *testing.T) {
	t.Helper()

	level := zerolog.GlobalLevel()
	t.Cleanup(func() { zerolog.SetGlobalLevel(level) })
}

func TestGc5SetupLoggerLevels(t *testing.T) {
	gc5RestoreLogLevel(t)

	tests := []struct {
		name  string
		trace bool
		debug bool
		want  zerolog.Level
	}{
		{"default", false, false, zerolog.InfoLevel},
		{"debug", false, true, zerolog.DebugLevel},
		{"trace wins over debug", true, true, zerolog.TraceLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetupLogger(&types.Config{Trace: tt.trace, Debug: tt.debug})

			require.Equal(t, tt.want, zerolog.GlobalLevel())
		})
	}
}

func TestGc5SetupProfilerUnknownType(t *testing.T) {
	cfg := &types.Config{}
	cfg.Profiler.Type = "unknown"

	// an unsupported profiler type is skipped without error.
	SetupProfiler(cfg)
}

func TestGc5RegisterServerCommand(t *testing.T) {
	gc5RestoreLogLevel(t)
	t.Setenv("HOME", t.TempDir())

	t.Run("initializer failure is reported", func(t *testing.T) {
		app := gc5NewServerApp(func(context.Context, *types.Config) (*System, error) {
			return nil, errGc5Initializer
		})

		_, err := app.Parse([]string{"server"})
		require.ErrorIs(t, err, errGc5Initializer)
		require.ErrorContains(t, err, "wiring the system")
	})

	t.Run("bootstrap failure is reported", func(t *testing.T) {
		errBootstrap := errors.New("gc5 bootstrap failure")
		app := gc5NewServerApp(func(context.Context, *types.Config) (*System, error) {
			return &System{bootstrap: func(context.Context) error { return errBootstrap }}, nil
		})

		_, err := app.Parse([]string{"server", "--no-enable-ci"})
		require.ErrorIs(t, err, errBootstrap)
		require.ErrorContains(t, err, "bootstrapping the system")
	})

	t.Run("configuration failure is reported", func(t *testing.T) {
		t.Setenv("GITNESS_DATABASE_MAX_OPEN_CONNS", "not-a-number")
		app := gc5NewServerApp(func(context.Context, *types.Config) (*System, error) {
			t.Error("initializer must not run with an invalid configuration")
			return nil, errGc5Initializer
		})

		_, err := app.Parse([]string{"server", "/does/not/exist.env"})
		require.ErrorContains(t, err, "loading configuration")
	})
}

func TestGc5NewSystem(t *testing.T) {
	metricServer := ProvideNoOpMetricServer()
	require.NotNil(t, metricServer)

	system := NewSystem(nil, nil, nil, nil, nil, services.Services{}, metricServer)

	require.NotNil(t, system)
	require.Equal(t, metricServer, system.metricServer)
}
