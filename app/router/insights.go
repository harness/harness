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

package router

import (
	"net/http"
	"regexp"

	handlerinsights "github.com/harness/gitness/app/api/handler/insights"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/logging"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// insightsIngestPath matches the Sentry SDK endpoints: /api/{numeric project id}/envelope/ and /store/.
// REST API paths always start with /api/v1/, so the numeric project id can't collide with them.
var insightsIngestPath = regexp.MustCompile(`^/api/\d+/(envelope|store)/?$`)

// InsightsIngestRouter serves the Sentry compatible ingestion endpoints. Requests are not
// authenticated with a Harness session, the DSN public key is validated by the service.
type InsightsIngestRouter struct {
	handler http.Handler
}

// NewInsightsIngestRouter returns the router for the Insights ingestion endpoints.
func NewInsightsIngestRouter(svc *insights.Service) *InsightsIngestRouter {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	// Browser SDKs call the endpoint cross origin; the key travels in the query string or
	// X-Sentry-Auth, never in cookies, so any origin is allowed and credentials are not.
	r.Use(cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{
			"Content-Type", "Content-Encoding", "Accept", "Authorization", "X-Sentry-Auth", "X-Requested-With",
			"Sentry-Trace", "Baggage",
		},
		ExposedHeaders:   []string{"Retry-After", "X-Sentry-Rate-Limits", "X-Sentry-Error"},
		AllowCredentials: false,
		MaxAge:           86400,
	}).Handler)

	envelope := handlerinsights.HandleIngest(svc, false)
	legacyStore := handlerinsights.HandleIngest(svc, true)
	const param = "/api/{" + handlerinsights.PathParamProjectID + "}"
	r.HandleFunc(param+"/envelope/", envelope)
	r.HandleFunc(param+"/envelope", envelope)
	r.HandleFunc(param+"/store/", legacyStore)
	r.HandleFunc(param+"/store", legacyStore)

	return &InsightsIngestRouter{handler: r}
}

func (r *InsightsIngestRouter) Handle(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(logging.NewContext(req.Context(), WithLoggingRouter("insights")))
	r.handler.ServeHTTP(w, req)
}

func (r *InsightsIngestRouter) IsEligibleTraffic(req *http.Request) bool {
	return insightsIngestPath.MatchString(req.URL.Path)
}

func (r *InsightsIngestRouter) Name() string {
	return "insights"
}
