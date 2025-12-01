//  Copyright 2023 Harness, Inc.
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

package maven

import (
	"net/http"

	middlewareauthn "github.com/harness/gitness/app/api/middleware/authn"
	"github.com/harness/gitness/registry/app/api/handler/maven"
	"github.com/harness/gitness/registry/app/api/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler interface {
	http.Handler
}

func NewMavenHandler(handler *maven.Handler) Handler {
	r := chi.NewRouter()

	var routeHandlers = map[string]http.HandlerFunc{
		http.MethodHead: handler.HeadArtifact,
		http.MethodGet:  handler.GetArtifact,
		http.MethodPut:  handler.PutArtifact,
	}

	r.Route("/maven", func(r chi.Router) {
		r.Use(middleware.StoreOriginalPath)
		r.Use(middleware.CheckAuthHeader())
		r.Use(middlewareauthn.Attempt(handler.Authenticator))
		r.Use(middleware.CheckAuthWithChallenge(handler, handler.SpaceFinder, handler.PublicAccessService))
		r.Use(middleware.TrackDownloadStatForMavenArtifact(handler))
		r.Use(middleware.TrackBandwidthStatForMavenArtifacts(handler))

		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			methodType := req.Method

			if h, ok := routeHandlers[methodType]; ok {
				h(w, req)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Invalid route"))
			if err != nil {
				log.Error().Err(err).Msg("Failed to write response")
				return
			}
		}))
	})

	return r
}
