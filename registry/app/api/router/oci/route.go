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
	"strings"

	middlewareauthn "github.com/harness/gitness/app/api/middleware/authn"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/middleware"
	"github.com/harness/gitness/registry/app/common"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type RouteType string

const (
	Manifests           RouteType = "manifests"            // /v2/:registry/:image/manifests/:reference.
	Blobs               RouteType = "blobs"                // /v2/:registry/:image/blobs/:digest.
	BlobsUploadsSession RouteType = "blob-uploads-session" // /v2/:registry/:image/blobs/uploads/:session_id.
	Tags                RouteType = "tags"                 // /v2/:registry/:image/tags/list.
	Referrers           RouteType = "referrers"            // /v2/:registry/:image/referrers/:digest.
	Invalid             RouteType = "invalid"              // Invalid route.
	// Add other route types here.
)

func GetRouteTypeV2(url string) RouteType {
	url = strings.Trim(url, "/")
	segments := strings.Split(url, "/")
	if len(segments) < 4 {
		return Invalid
	}

	typ := segments[len(segments)-2]

	switch typ {
	case "manifests":
		return Manifests
	case "blobs":
		if segments[len(segments)-1] == "uploads" {
			return BlobsUploadsSession
		}
		return Blobs
	case "uploads":
		return BlobsUploadsSession
	case "tags":
		return Tags
	case "referrers":
		return Referrers
	}
	return Invalid
}

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

	var routeHandlers = map[RouteType]map[string]HandlerBlock{
		Manifests: {
			http.MethodGet:    NewHandlerBlock2(handlerV2.GetManifest, true),
			http.MethodHead:   NewHandlerBlock2(handlerV2.HeadManifest, true),
			http.MethodPut:    NewHandlerBlock2(handlerV2.PutManifest, false),
			http.MethodDelete: NewHandlerBlock2(handlerV2.DeleteManifest, false),
		},
		Blobs: {
			http.MethodGet:    NewHandlerBlock2(handlerV2.GetBlob, true),
			http.MethodHead:   NewHandlerBlock2(handlerV2.HeadBlob, false),
			http.MethodDelete: NewHandlerBlock2(handlerV2.DeleteBlob, false),
		},
		BlobsUploadsSession: {
			http.MethodGet:    NewHandlerBlock2(handlerV2.GetUploadBlobStatus, false),
			http.MethodPatch:  NewHandlerBlock2(handlerV2.PatchBlobUpload, false),
			http.MethodPut:    NewHandlerBlock2(handlerV2.CompleteBlobUpload, false),
			http.MethodDelete: NewHandlerBlock2(handlerV2.CancelBlobUpload, false),
			http.MethodPost:   NewHandlerBlock2(handlerV2.InitiateUploadBlob, false),
		},
		Tags: {
			http.MethodGet: NewHandlerBlock2(handlerV2.GetTags, false),
		},
		Referrers: {
			http.MethodGet: NewHandlerBlock2(handlerV2.GetReferrers, false),
		},
	}
	r.Route("/v2", func(r chi.Router) {
		r.Use(middlewareauthn.Attempt(handlerV2.Authenticator))
		r.Get("/token", func(w http.ResponseWriter, req *http.Request) {
			handlerV2.GetToken(w, req)
		})

		r.With(middleware.OciCheckAuth(common.GenerateOciTokenURL(handlerV2.URLProvider.RegistryURL()))).
			Get("/", func(w http.ResponseWriter, req *http.Request) {
				handlerV2.APIBase(w, req)
			})
		r.Route("/{registryIdentifier}", func(r chi.Router) {
			r.Use(middleware.OciCheckAuth(common.GenerateOciTokenURL(handlerV2.URLProvider.RegistryURL())))
			r.Use(middleware.BlockNonOciSourceToken(handlerV2.URLProvider.RegistryURL()))
			r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				path := req.URL.Path
				methodType := req.Method

				requestType := GetRouteTypeV2(path)

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
