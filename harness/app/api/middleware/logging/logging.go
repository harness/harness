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

package logging

import (
	"net/http"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/logging"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

const (
	requestIDHeader = "X-Request-Id"
)

// HLogRequestIDHandler provides a middleware that injects request_id into the logging and execution context.
// It prefers the X-Request-Id header, if that doesn't exist it creates a new request id similar to zerolog.
func HLogRequestIDHandler() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// read requestID from header (or create new one if none exists)
			var reqID string
			if reqIDs, ok := r.Header[requestIDHeader]; ok && len(reqIDs) > 0 && len(reqIDs[0]) > 0 {
				reqID = reqIDs[0]
			} else {
				// similar to zerolog requestID generation
				reqID = xid.New().String()
			}

			// add requestID to context for internal usage client!
			ctx = request.WithRequestID(ctx, reqID)
			ctx = git.WithRequestID(ctx, reqID)

			// update logging context with request ID
			logging.UpdateContext(ctx, logging.WithRequestID(reqID))

			// write request ID to response headers
			w.Header().Set(requestIDHeader, reqID)

			// continue serving request
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HLogAccessLogHandler provides an hlog based middleware that logs access logs.
func HLogAccessLogHandler() func(http.Handler) http.Handler {
	return hlog.AccessHandler(
		func(r *http.Request, status, size int, duration time.Duration) {
			hlog.FromRequest(r).Info().
				Int("http.status_code", status).
				Int("http.response_size_bytes", size).
				Dur("http.elapsed_ms", duration).
				Msg("http request completed.")
		},
	)
}

func URLHandler(fieldKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := zerolog.Ctx(r.Context())
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				if r.URL.RawPath != "" {
					return c.Str(fieldKey, r.URL.RawPath+"?"+r.URL.RawQuery)
				}
				return c.Str(fieldKey, r.URL.Path+"?"+r.URL.RawQuery)
			})
			next.ServeHTTP(w, r)
		})
	}
}
