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

package generic

import (
	"net/http"

	middlewareauthn "github.com/harness/gitness/app/api/middleware/authn"
	"github.com/harness/gitness/registry/app/api/handler/generic"
	"github.com/harness/gitness/registry/app/api/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler interface {
	http.Handler
}

func NewGenericArtifactHandler(handler *generic.Handler) Handler {
	r := chi.NewRouter()

	var routeHandlers = map[string]http.HandlerFunc{
		http.MethodPut: handler.PushArtifact,
		http.MethodGet: handler.PullArtifact,
	}
	r.Route("/generic", func(r chi.Router) {
		r.Use(middlewareauthn.Attempt(handler.Authenticator))
		r.Use(middleware.TrackDownloadStatForGenericArtifact(handler))
		r.Use(middleware.TrackBandwidthStatForGenericArtifacts(handler))

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
