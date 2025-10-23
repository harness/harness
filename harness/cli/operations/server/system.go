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
	"github.com/harness/gitness/app/pipeline/resolver"
	"github.com/harness/gitness/app/server"
	"github.com/harness/gitness/app/services"
	"github.com/harness/gitness/ssh"

	"github.com/drone/runner-go/poller"
)

// System stores high level System sub-routines.
type System struct {
	bootstrap       bootstrap.Bootstrap
	server          *server.Server
	sshServer       *ssh.Server
	resolverManager *resolver.Manager
	poller          *poller.Poller
	services        services.Services
}

// NewSystem returns a new system structure.
func NewSystem(
	bootstrap bootstrap.Bootstrap,
	server *server.Server,
	sshServer *ssh.Server,
	poller *poller.Poller,
	resolverManager *resolver.Manager,
	services services.Services,
) *System {
	return &System{
		bootstrap:       bootstrap,
		server:          server,
		sshServer:       sshServer,
		poller:          poller,
		resolverManager: resolverManager,
		services:        services,
	}
}
