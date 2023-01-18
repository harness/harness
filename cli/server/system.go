// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/cron"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/services"
	"github.com/harness/gitness/internal/webhook"
)

// system stores high level system sub-routines.
type system struct {
	bootstrap     bootstrap.Bootstrap
	server        *server.Server
	gitRPCServer  *gitrpcserver.Server
	webhookServer *webhook.Server
	nightly       *cron.Nightly
	services      services.Services
}

// newSystem returns a new system structure.
func newSystem(bootstrap bootstrap.Bootstrap, server *server.Server, gitRPCServer *gitrpcserver.Server,
	webhookServer *webhook.Server, nightly *cron.Nightly, services services.Services) *system {
	return &system{
		bootstrap:     bootstrap,
		server:        server,
		gitRPCServer:  gitRPCServer,
		webhookServer: webhookServer,
		nightly:       nightly,
		services:      services,
	}
}
