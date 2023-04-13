// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package codecomments

import (
	"github.com/harness/gitness/gitrpc"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideMigrator,
)

func ProvideMigrator(
	gitRPCClient gitrpc.Interface,
) *Migrator {
	return &Migrator{
		gitRPCClient: gitRPCClient,
	}
}
