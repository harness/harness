// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package url

import (
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideURLProvider)

func ProvideURLProvider(config *types.Config) (*Provider, error) {
	return NewProvider(
		config.URL.API,
		config.URL.APIInternal,
		config.URL.Git,
	)
}
