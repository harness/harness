// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package sse

import (
	"github.com/harness/gitness/pubsub"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideEventsStreaming,
)

func ProvideEventsStreaming(pubsub pubsub.PubSub) Streamer {
	const namespace = "sse"
	return NewStreamer(pubsub, namespace)
}
