// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"github.com/harness/gitness/events"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideReaderFactory,
	ProvideReporter,
)

func ProvideReaderFactory(eventsSystem *events.System) (*events.ReaderFactory[*Reader], error) {
	return NewReaderFactory(eventsSystem)
}

func ProvideReporter(eventsSystem *events.System) (*Reporter, error) {
	return NewReporter(eventsSystem)
}
