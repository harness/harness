// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package livelog

import (
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideLogStream,
)

// ProvideLogStream provides an implementation of a logs streamer
// TODO: Implement Redis backend once implemented and add the check in config
func ProvideLogStream(config *types.Config) LogStream {
	return NewMemory()
}
