// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package plugin

import (
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvidePluginManager,
)

// ProvidePluginManager provides an execution runner.
func ProvidePluginManager(
	config *types.Config,
	pluginStore store.PluginStore,
) *PluginManager {
	return NewPluginManager(config, pluginStore)
}
