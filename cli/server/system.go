// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/cron"
	"github.com/harness/gitness/internal/server"
)

// system stores high level system sub-routines.
type system struct {
	bootstrap bootstrap.Bootstrap
	server    *server.Server
	nightly   *cron.Nightly
}

// newSystem returns a new system structure.
func newSystem(bootstrap bootstrap.Bootstrap, server *server.Server, nightly *cron.Nightly) *system {
	return &system{
		bootstrap: bootstrap,
		server:    server,
		nightly:   nightly,
	}
}
