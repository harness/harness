// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package url

import (
	"fmt"
	"github.com/harness/gitness/types"
	"net/url"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideURLProvider)

const harnessCodeAPIURLRaw = "http://app.harness.io/gateway/code/api/v1/"

func ProvideURLProvider(config *types.Config) (*Provider, error) {
	harnessCodeApiUrl, err := url.Parse(harnessCodeAPIURLRaw)
	if err != nil {
		return nil, fmt.Errorf("provided harnessCodeAPIURLRaw '%s' is invalid: %w", harnessCodeAPIURLRaw, err)
	}
	return NewProvider(
		config.URL.API,
		config.URL.APIInternal,
		config.URL.Git,
		config.URL.CIURL,
		harnessCodeApiUrl,
	)
}
