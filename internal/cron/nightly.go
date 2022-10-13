// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cron

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// Nightly is a sub-routine that periodically purges historical data.
type Nightly struct {
	// Inject required stores here
}

// NewNightly returns a new Nightly sub-routine.
func NewNightly() *Nightly {
	return &Nightly{}
}

// Run runs the purge sub-routine.
func (n *Nightly) Run(ctx context.Context) {
	const hoursPerDay = 24
	ticker := time.NewTicker(hoursPerDay * time.Hour)
	logger := log.Ctx(ctx)
	for {
		select {
		case <-ctx.Done():
			return // break
		case <-ticker.C:
			// TODO replace this with your nightly
			// cron tasks.
			logger.Trace().Msg("cron job executed")
		}
	}
}
