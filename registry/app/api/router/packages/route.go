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

package packages

import (
	"net/http"

	middlewareauthn "github.com/harness/gitness/app/api/middleware/authn"
	"github.com/harness/gitness/registry/app/api/handler/generic"
	"github.com/harness/gitness/registry/app/api/handler/maven"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/handler/python"
	"github.com/harness/gitness/registry/app/api/middleware"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
)

type Handler interface {
	http.Handler
}

/**
 * NewRouter creates a new router for the package API.
 * It sets up the routes and middleware for handling package-related requests.
 * Paths look like:
 * For all packages: /{rootIdentifier}/{registryName}/<package_type>/<package specific routes> .
 */
func NewRouter(
	packageHandler packages.Handler,
	mavenHandler *maven.Handler,
	genericHandler *generic.Handler,
	pythonHandler python.Handler,
) Handler {
	r := chi.NewRouter()

	r.Route("/{rootIdentifier}/{registryIdentifier}", func(r chi.Router) {
		r.Use(middleware.StoreOriginalURL)

		r.Route("/maven", func(r chi.Router) {
			r.Use(middleware.CheckMavenAuthHeader())
			r.Use(middlewareauthn.Attempt(packageHandler.GetAuthenticator()))
			r.Use(middleware.CheckMavenAuth())
			r.Use(middleware.TrackDownloadStatForMavenArtifact(mavenHandler))
			r.Use(middleware.TrackBandwidthStatForMavenArtifacts(mavenHandler))
			r.Get("/*", mavenHandler.GetArtifact)
			r.Head("/*", mavenHandler.HeadArtifact)
			r.Put("/*", mavenHandler.PutArtifact)
		})

		r.Route("/generic", func(r chi.Router) {
			r.Use(middlewareauthn.Attempt(packageHandler.GetAuthenticator()))
			r.Use(middleware.TrackDownloadStatForGenericArtifact(genericHandler))
			r.Use(middleware.TrackBandwidthStatForGenericArtifacts(genericHandler))

			r.Get("/*", genericHandler.PullArtifact)
			r.Put("/*", genericHandler.PushArtifact)
		})

		r.Route("/python", func(r chi.Router) {
			r.Use(middlewareauthn.Attempt(packageHandler.GetAuthenticator()))

			// TODO (Arvind): Move this to top layer with total abstraction
			r.With(middleware.StoreArtifactInfo(pythonHandler)).
				With(middleware.RequestPackageAccess(packageHandler, enum.PermissionArtifactsUpload)).
				Post("/*", pythonHandler.UploadPackageFile)
			r.With(middleware.StoreArtifactInfo(pythonHandler)).
				With(middleware.RequestPackageAccess(packageHandler, enum.PermissionArtifactsDownload)).
				Get("/files/{image}/{version}/{filename}", pythonHandler.DownloadPackageFile)
			r.With(middleware.StoreArtifactInfo(pythonHandler)).
				With(middleware.RequestPackageAccess(packageHandler, enum.PermissionArtifactsDownload)).
				Get("/simple/{image}", pythonHandler.PackageMetadata)
			r.With(middleware.StoreArtifactInfo(pythonHandler)).
				With(middleware.RequestPackageAccess(packageHandler, enum.PermissionArtifactsDownload)).
				Get("/simple/{image}/", pythonHandler.PackageMetadata)
		})
	})

	return r
}
