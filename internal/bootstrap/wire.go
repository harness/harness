// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package bootstrap

import (
	"github.com/google/wire"
	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/types"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideBootstrap)

func ProvideBootstrap(config *types.Config, userCtrl *user.Controller) Bootstrap {
	return System(config, userCtrl)
}
