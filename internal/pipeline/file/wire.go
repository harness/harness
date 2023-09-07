// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package file

import (
	"github.com/harness/gitness/gitrpc"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideFileService,
)

// ProvideFileService provides a service which can read file contents
// from a repository.
func ProvideFileService(gitRPCClient gitrpc.Interface) FileService {
	return new(gitRPCClient)
}
