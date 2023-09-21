// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject
// +build wireinject

package main

import (
	"github.com/harness/gitness/gitrpc/server"

	"github.com/google/wire"
)

func initSystem() (*system, error) {
	wire.Build(
		newSystem,
		ProvideGitRPCServerConfig,
		server.WireSet,
		ProvideRedis,
	)
	return &system{}, nil
}
