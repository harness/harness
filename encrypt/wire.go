// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package encrypt

import (
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideEncrypter,
)

func ProvideEncrypter(config *types.Config) (Encrypter, error) {
	if config.Encrypter.Secret == "" {
		return &none{}, nil
	}
	return New(config.Encrypter.Secret, config.Encrypter.MixedContent)
}
