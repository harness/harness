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

package server

import (
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/pipeline/plugin"
	"github.com/harness/gitness/app/server"
	"github.com/harness/gitness/app/services"
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	gitrpccron "github.com/harness/gitness/gitrpc/server/cron"

	"github.com/drone/runner-go/poller"
)

// System stores high level System sub-routines.
type System struct {
	bootstrap      bootstrap.Bootstrap
	server         *server.Server
	gitRPCServer   *gitrpcserver.GRPCServer
	pluginManager  *plugin.Manager
	poller         *poller.Poller
	services       services.Services
	gitRPCCronMngr *gitrpccron.Manager
}

// NewSystem returns a new system structure.
func NewSystem(bootstrap bootstrap.Bootstrap, server *server.Server, poller *poller.Poller,
	gitRPCServer *gitrpcserver.GRPCServer, pluginManager *plugin.Manager,
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
