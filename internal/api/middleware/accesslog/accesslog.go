// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package accesslog

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/hlog"
)

/*
 * A simple middleware that logs completed requests using the default hlog access handler.
 */
func HlogHandler() func(http.Handler) http.Handler {
	return hlog.AccessHandler(
		func(r *http.Request, status, size int, duration time.Duration) {
			hlog.FromRequest(r).Info().
				Int("status_code", status).
				Int("response_size_bytes", size).
				Dur("elapsed_ms", duration).
				Msg("request completed.")
		},
	)
}
