// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package commit

import (
	"github.com/harness/gitness/gitrpc"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideCommitService,
)

// ProvideCommitService provides a service which can fetch commit
// information about a repository.
func ProvideCommitService(gitRPCClient gitrpc.Interface) CommitService {
	return new(gitRPCClient)
}
