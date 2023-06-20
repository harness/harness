// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package migrate

import (
	"context"
	"fmt"

	"github.com/harness/gitness/cli/server"
	"github.com/harness/gitness/store/database"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"gopkg.in/alecthomas/kingpin.v2"
)

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

// Register the server command.
func Register(app *kingpin.Application) {
	cmd := app.Command("migrate", "database migration tool")
	registerCurrent(cmd)
	registerTo(cmd)
}
