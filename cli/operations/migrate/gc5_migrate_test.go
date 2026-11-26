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

package migrate

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/alecthomas/kingpin.v2"
)

// gc5SetupEnv points the configuration at a fresh sqlite database inside a temporary directory.
func gc5SetupEnv(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("GITNESS_DATABASE_DRIVER", "sqlite3")
	t.Setenv("GITNESS_DATABASE_DATASOURCE", filepath.Join(dir, "gc5.sqlite3"))

	return dir
}

func gc5NewApp() *kingpin.Application {
	app := kingpin.New("gc5", "test")
	app.Terminate(nil)
	app.ErrorWriter(io.Discard)
	app.UsageWriter(io.Discard)
	Register(app)

	return app
}

// gc5CaptureStdout runs fn and returns what it printed to stdout.
func gc5CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(out)
}

func TestGc5RegisterCommands(t *testing.T) {
	app := gc5NewApp()

	_, err := app.Parse([]string{"migrate", "to"})
	require.Error(t, err, "the target version is mandatory")

	_, err = app.Parse([]string{"migrate", "unknown"})
	require.Error(t, err)
}

func TestGc5MigrateToAndCurrent(t *testing.T) {
	gc5SetupEnv(t)

	// migrate a fresh database to the first version.
	_, err := gc5NewApp().Parse([]string{"migrate", "to", "0001_create_table_b_spaces"})
	require.NoError(t, err)

	var parseErr error
	out := gc5CaptureStdout(t, func() {
		_, parseErr = gc5NewApp().Parse([]string{"migrate", "current"})
	})
	require.NoError(t, parseErr)
	require.Contains(t, out, "0001_create_table_b_spaces")
}

func TestGc5MigrateToUnknownVersion(t *testing.T) {
	gc5SetupEnv(t)

	_, err := gc5NewApp().Parse([]string{"migrate", "to", "no-such-version"})
	require.Error(t, err)
}

func TestGc5MigrateInvalidConfig(t *testing.T) {
	gc5SetupEnv(t)
	t.Setenv("GITNESS_DATABASE_MAX_OPEN_CONNS", "not-a-number")

	_, err := gc5NewApp().Parse([]string{"migrate", "current"})
	require.ErrorContains(t, err, "failed to load configuration")

	_, err = gc5NewApp().Parse([]string{"migrate", "to", "0001_create_table_b_spaces"})
	require.ErrorContains(t, err, "failed to load configuration")
}

func TestGc5MigrateDatabaseConnectFailure(t *testing.T) {
	dir := gc5SetupEnv(t)
	t.Setenv("GITNESS_DATABASE_DRIVER", "no-such-driver")
	t.Setenv("GITNESS_DATABASE_DATASOURCE", filepath.Join(dir, "x"))

	_, err := gc5NewApp().Parse([]string{"migrate", "current"})
	require.ErrorContains(t, err, "failed to create database handle")
}

func TestGc5GetDBLoadsEnvFile(t *testing.T) {
	dir := gc5SetupEnv(t)
	// the env file doesn't override variables that are already set, but a missing file is tolerated.
	db, err := getDB(t.Context(), filepath.Join(dir, "missing.env"))
	require.NoError(t, err)
	require.Equal(t, "sqlite3", db.DriverName())
	require.NoError(t, db.Close())
}

func TestGc5SetupLoggingContext(t *testing.T) {
	ctx := setupLoggingContext(t.Context())

	require.NotNil(t, ctx)
}
