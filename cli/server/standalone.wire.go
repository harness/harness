// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject && !harness
// +build wireinject,!harness

package server

import (
	"context"

	"github.com/harness/gitness/gitrpc"
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/cron"
	"github.com/harness/gitness/internal/router"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database"
	"github.com/harness/gitness/internal/store/memory"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
)

func initSystem(ctx context.Context, config *types.Config) (*system, error) {
	wire.Build(
		newSystem,
		PackageConfigsWireSet,
		bootstrap.WireSet,
		database.WireSet,
		memory.WireSet,
		router.WireSet,
		server.WireSet,
		cron.WireSet,
		space.WireSet,
		repo.WireSet,
		pullreq.WireSet,
		serviceaccount.WireSet,
		user.WireSet,
		authn.WireSet,
		authz.WireSet,
		gitrpcserver.WireSet,
		gitrpc.WireSet,
		store.WireSet,
		check.WireSet,
	)
	return &system{}, nil
}
