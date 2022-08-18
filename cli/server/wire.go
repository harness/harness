// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject
// +build wireinject

package server

import (
	"github.com/harness/scm/internal/cron"
	"github.com/harness/scm/internal/router"
	"github.com/harness/scm/internal/server"
	"github.com/harness/scm/internal/store/database"
	"github.com/harness/scm/internal/store/memory"
	"github.com/harness/scm/types"

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
