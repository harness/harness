// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
