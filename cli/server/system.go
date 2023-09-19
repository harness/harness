// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	gitrpccron "github.com/harness/gitness/gitrpc/server/cron"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/pipeline/plugin"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/services"

	"github.com/drone/runner-go/poller"
)

// System stores high level System sub-routines.
type System struct {
	bootstrap      bootstrap.Bootstrap
	server         *server.Server
	gitRPCServer   *gitrpcserver.GRPCServer
	pluginManager  *plugin.PluginManager
	poller         *poller.Poller
	services       services.Services
	gitRPCCronMngr *gitrpccron.Manager
}

// NewSystem returns a new system structure.
func NewSystem(bootstrap bootstrap.Bootstrap, server *server.Server, poller *poller.Poller,
	gitRPCServer *gitrpcserver.GRPCServer, pluginManager *plugin.PluginManager,
	gitrpccron *gitrpccron.Manager, services services.Services) *System {
	return &System{
		bootstrap:      bootstrap,
		server:         server,
		poller:         poller,
		gitRPCServer:   gitRPCServer,
		pluginManager:  pluginManager,
		services:       services,
		gitRPCCronMngr: gitrpccron,
	}
}
