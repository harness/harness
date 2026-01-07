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
	"context"
	"fmt"

	"github.com/harness/gitness/cli/operations/server"
	"github.com/harness/gitness/store/database"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Register the server command.
func Register(app *kingpin.Application) {
	cmd := app.Command("migrate", "database migration tool")
	registerCurrent(cmd)
	registerTo(cmd)
}

func getDB(ctx context.Context, envfile string) (*sqlx.DB, error) {
	_ = godotenv.Load(envfile)

	config, err := server.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	db, err := database.Connect(ctx, config.Database.Driver, config.Database.Datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to create database handle: %w", err)
	}

	return db, nil
}

func setupLoggingContext(ctx context.Context) context.Context {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	log := log.Logger.With().Logger()
	return log.WithContext(ctx)
}
