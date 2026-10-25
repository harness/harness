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

// Package insights provides the HTTP handlers of Insights.
package insights

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/insights/ingest"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// PathParamProjectID is the name of the project id path parameter of the ingestion endpoints.
const PathParamProjectID = "project_id"

// HandleIngest returns the handler of the Sentry compatible ingestion endpoints.
// legacy selects the store endpoint (single JSON event) instead of the envelope endpoint.
// The endpoints don't use session authentication: the DSN public key authenticates the request.
func HandleIngest(svc *insights.Service, legacy bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST, OPTIONS")
			writeDetail(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		res, err := svc.Ingest(ctx, insights.IngestInput{
			ProjectID:       chi.URLParam(r, PathParamProjectID),
			SentryAuth:      sentryAuthHeader(r),
			Query:           r.URL.Query(),
			Body:            r.Body,
			ContentEncoding: r.Header.Get("Content-Encoding"),
			Legacy:          legacy,
		})
		if err != nil {
			writeIngestError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, map[string]string{"id": res.EventID})
	}
}

// sentryAuthHeader returns X-Sentry-Auth, falling back to "Authorization: Sentry ...".
func sentryAuthHeader(r *http.Request) string {
	if h := r.Header.Get("X-Sentry-Auth"); h != "" {
		return h
	}
	if h := r.Header.Get("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "sentry ") {
		return h
	}
	return ""
}

func writeIngestError(ctx context.Context, w http.ResponseWriter, err error) {
	var rl *insights.RateLimitedError

	switch {
	case errors.As(err, &rl):
		secs := int(math.Ceil(rl.RetryAfter.Seconds()))
		if secs < 1 {
			secs = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		w.Header().Set("X-Sentry-Rate-Limits", fmt.Sprintf("%d::project", secs))
		writeDetail(w, http.StatusTooManyRequests, "rate limit exceeded")
	case errors.Is(err, insights.ErrInvalidKey),
		errors.Is(err, ingest.ErrMissingKey),
		errors.Is(err, ingest.ErrProjectMismatch):
		writeDetail(w, http.StatusForbidden, "invalid project or key")
	case errors.Is(err, ingest.ErrTooLarge), errors.Is(err, ingest.ErrTooManyItems):
		writeDetail(w, http.StatusRequestEntityTooLarge, "payload too large")
	case ingest.IsClientError(err):
		writeDetail(w, http.StatusBadRequest, "invalid request")
	default:
		log.Ctx(ctx).Err(err).Msg("insights: failed to ingest event")
		writeDetail(w, http.StatusInternalServerError, "internal error")
	}
}

// writeDetail writes the {"detail": ...} error body Sentry SDKs expect.
func writeDetail(w http.ResponseWriter, status int, detail string) {
	render.JSON(w, status, map[string]string{"detail": detail})
}
