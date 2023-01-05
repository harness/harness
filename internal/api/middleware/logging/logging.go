// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logging

import (
	"net/http"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/request"

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

			// add requestID to context for internal usage + gitrpc client!
			ctx = request.WithRequestID(ctx, reqID)
			ctx = gitrpc.WithRequestID(ctx, reqID)

			// update logging context with request ID
			log := zerolog.Ctx(ctx)
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("http.request_id", reqID)
			})

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
