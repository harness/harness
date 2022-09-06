// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject
// +build wireinject

package server

import (
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/cron"
	"github.com/harness/gitness/internal/router"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/store/database"
	"github.com/harness/gitness/internal/store/memory"
	"github.com/harness/gitness/types"

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
		authn.WireSet,
		authz.WireSet,
	)
	return &system{}, nil
}
