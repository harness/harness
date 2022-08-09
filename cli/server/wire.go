// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//+build wireinject

package server

import (
	"github.com/bradrydzewski/my-app/internal/cron"
	"github.com/bradrydzewski/my-app/internal/router"
	"github.com/bradrydzewski/my-app/internal/server"
	"github.com/bradrydzewski/my-app/internal/store/database"
	"github.com/bradrydzewski/my-app/internal/store/memory"
	"github.com/bradrydzewski/my-app/types"

	"github.com/google/wire"
)

func initSystem(config *types.Config) (*system, error) {
	wire.Build(
		database.WireSet,
		memory.WireSet,
		router.WireSet,
		server.WireSet,
		cron.WireSet,
		newSystem,
	)
	return &system{}, nil
}
