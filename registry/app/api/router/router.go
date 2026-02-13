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

package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/app/api/middleware/address"
	"github.com/harness/gitness/app/api/middleware/logging"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/handler/swagger"
	generic2 "github.com/harness/gitness/registry/app/api/router/generic"
	"github.com/harness/gitness/registry/app/api/router/harness"
	"github.com/harness/gitness/registry/app/api/router/maven"
	"github.com/harness/gitness/registry/app/api/router/oci"
	"github.com/harness/gitness/registry/app/api/router/packages"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"
)

type AppRouter interface {
	http.Handler
}

func GetAppRouter(
	ociHandler oci.RegistryOCIHandler,
	appHandler harness.APIHandler,
	baseURL string,
	mavenHandler maven.Handler,
	genericHandler generic2.Handler,
	packageHandler packages.Handler,
) AppRouter {
	r := chi.NewRouter()

	// Logging specific
	r.Use(hlog.URLHandler("http.url"))
	r.Use(hlog.MethodHandler("http.method"))
	r.Use(logging.HLogRequestIDHandler())
	r.Use(logging.HLogAccessLogHandler())
	r.Use(address.Handler("", ""))

	r.Use(audit.Middleware())

	r.Group(func(r chi.Router) {
		r.Handle(fmt.Sprintf("%s/*", baseURL), appHandler)
		r.Handle("/v2/*", ociHandler)
		// deprecated
		r.Handle("/maven/*", mavenHandler)
		// deprecated
		r.Handle("/generic/*", genericHandler)

		r.Mount("/pkg/", packageHandler)
		r.Handle("/registry/swagger*", swagger.GetSwaggerHandler("/registry"))
	})

	return r
}
