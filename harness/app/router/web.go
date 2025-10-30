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

	middlewareweb "github.com/harness/gitness/app/api/middleware/web"
	"github.com/harness/gitness/app/api/openapi"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/web"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	"github.com/unrolled/secure"
)

// NewWebHandler returns a new WebHandler.
func NewWebHandler(
	authenticator authn.Authenticator,
	openapi openapi.Service,
	sec *secure.Secure,
	publicResourceCreationEnabled bool,
	uiSourceOverride string,
) http.Handler {
	// Use go-chi router for inner routing
	r := chi.NewRouter()

	// openapi endpoints
	// TODO: this should not be generated and marshaled on the fly every time?
	r.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		spec := openapi.Generate()
		data, err := spec.MarshalYAML()
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to serialize openapi.yaml")
			render.InternalError(ctx, w)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})

	// swagger endpoints
	r.Group(func(r chi.Router) {
		r.Use(sec.Handler)

		swagger := v5emb.NewHandlerWithConfig(swgui.Config{
			Title:       "API Definition",
			SwaggerJSON: "/openapi.yaml",
			BasePath:    "/swagger",
			// Available settings can be found here:
			// https://swagger.io/docs/open-source-tools/swagger-ui/usage/configuration/
			SettingsUI: map[string]string{
				"queryConfigEnabled":       "false", // block code injection vulnerability
				"defaultModelsExpandDepth": "1",
			},
		})

		r.Handle("/swagger", swagger)
		r.Handle("/swagger/*", swagger)
	})

	// serve all other routes from the embedded filesystem,
	// which in turn serves the user interface.
	r.With(
		sec.Handler,
		middlewareweb.PublicAccess(publicResourceCreationEnabled, authenticator),
	).NotFound(
		web.Handler(uiSourceOverride),
	)

	return r
}
