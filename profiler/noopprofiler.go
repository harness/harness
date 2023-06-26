// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package profiler

import "github.com/rs/zerolog/log"

type NoopProfiler struct {
}

func (noopProfiler *NoopProfiler) StartProfiling(serviceName, serviceVersion string) {
	log.Info().Msg("Not starting profiler")
}
