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

package oci

import (
	"net/http"

	middlewareauthn "github.com/harness/gitness/app/api/middleware/authn"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/middleware"
	"github.com/harness/gitness/registry/app/api/router/utils"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type HandlerBlock struct {
	Handler2      http.HandlerFunc
	RemoteSupport bool
}

func NewHandlerBlock2(h2 http.HandlerFunc, remoteSupport bool) HandlerBlock {
	return HandlerBlock{
		Handler2:      h2,
		RemoteSupport: remoteSupport,
	}
}

type RegistryOCIHandler interface {
	http.Handler
}

func NewOCIHandler(handlerV2 *oci.Handler) RegistryOCIHandler {
	r := chi.NewRouter()

	var routeHandlers = map[utils.RouteType]map[string]HandlerBlock{
		utils.Manifests: {
			http.MethodGet:    NewHandlerBlock2(handlerV2.GetManifest, true),
			http.MethodHead:   NewHandlerBlock2(handlerV2.HeadManifest, true),
			http.MethodPut:    NewHandlerBlock2(handlerV2.PutManifest, false),
			http.MethodDelete: NewHandlerBlock2(handlerV2.DeleteManifest, false),
		},
		utils.Blobs: {
			http.MethodGet:    NewHandlerBlock2(handlerV2.GetBlob, true),
			http.MethodHead:   NewHandlerBlock2(handlerV2.HeadBlob, false),
			http.MethodDelete: NewHandlerBlock2(handlerV2.DeleteBlob, false),
		},
		utils.BlobsUploadsSession: {
			http.MethodGet:    NewHandlerBlock2(handlerV2.GetUploadBlobStatus, false),
			http.MethodPatch:  NewHandlerBlock2(handlerV2.PatchBlobUpload, false),
			http.MethodPut:    NewHandlerBlock2(handlerV2.CompleteBlobUpload, false),
			http.MethodDelete: NewHandlerBlock2(handlerV2.CancelBlobUpload, false),
			http.MethodPost:   NewHandlerBlock2(handlerV2.InitiateUploadBlob, false),
		},
		utils.Tags: {
			http.MethodGet: NewHandlerBlock2(handlerV2.GetTags, false),
		},
		utils.Referrers: {
			http.MethodGet: NewHandlerBlock2(handlerV2.GetReferrers, false),
		},
	}

	r.Route("/v2", func(r chi.Router) {
		r.Use(middleware.StoreOriginalURL)
		r.Use(middlewareauthn.Attempt(handlerV2.Authenticator))
		r.Get("/token", func(w http.ResponseWriter, req *http.Request) {
			handlerV2.GetToken(w, req)
		})

		r.With(middleware.OciCheckAuth(handlerV2.URLProvider)).
			Get("/", func(w http.ResponseWriter, req *http.Request) {
				handlerV2.APIBase(w, req)
			})

		r.Route("/{registryIdentifier}", func(r chi.Router) {
			r.Use(middleware.OciCheckAuth(handlerV2.URLProvider))
			r.Use(middleware.BlockNonOciSourceToken(handlerV2.URLProvider))
			r.Use(middleware.TrackDownloadStat(handlerV2))
			r.Use(middleware.TrackBandwidthStat(handlerV2))

			r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				path := req.URL.Path
				methodType := req.Method

				requestType := utils.GetRouteTypeV2(path)

				if _, ok := routeHandlers[requestType]; ok {
					if h, ok2 := routeHandlers[requestType][methodType]; ok2 {
						h.Handler2(w, req)
						return
					}
				}

				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte("Invalid route"))
				if err != nil {
					log.Error().Err(err).Msg("Failed to write response")
					return
				}
			}))
		})
	})

	return r
}
