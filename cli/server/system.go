// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	gitrpccron "github.com/harness/gitness/gitrpc/server/cron"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/services"
)

// system stores high level system sub-routines.
type system struct {
	bootstrap      bootstrap.Bootstrap
	server         *server.Server
	gitRPCServer   *gitrpcserver.Server
	services       services.Services
	gitRPCCronMngr *gitrpccron.Manager
}

// newSystem returns a new system structure.
func newSystem(bootstrap bootstrap.Bootstrap, server *server.Server, gitRPCServer *gitrpcserver.Server,
	gitrpccron *gitrpccron.Manager, services services.Services) *system {
	return &system{
		bootstrap:      bootstrap,
		server:         server,
		gitRPCServer:   gitRPCServer,
		services:       services,
		gitRPCCronMngr: gitrpccron,
	}
}
