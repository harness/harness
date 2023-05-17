// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cron

import (
	"github.com/harness/gitness/gitrpc/server"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideManager)

func ProvideManager(gitrpcconfig server.Config) *Manager {
	cmngr := NewManager()
	_ = AddAllGitRPCCronJobs(cmngr, gitrpcconfig)
	return cmngr
}
