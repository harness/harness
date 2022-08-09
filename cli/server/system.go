// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"github.com/bradrydzewski/my-app/internal/cron"
	"github.com/bradrydzewski/my-app/internal/server"
)

// system stores high level system sub-routines.
type system struct {
	server  *server.Server
	nightly *cron.Nightly
}

// newSystem returns a new system structure.
func newSystem(server *server.Server, nightly *cron.Nightly) *system {
	return &system{
		server:  server,
		nightly: nightly,
	}
}
