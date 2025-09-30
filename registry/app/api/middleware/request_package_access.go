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

package middleware

import (
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/types/enum"
)

func RequestPackageAccess(
	packageHandler packages.Handler,
	reqPermissions ...enum.Permission,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := packageHandler.GetRegistryCheckAccess(r.Context(), r, reqPermissions...)
			if err != nil {
				render.TranslatedUserError(r.Context(), w, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}

func RequestNugetPackageAccess(
	packageHandler packages.Handler,
	reqPermissions ...enum.Permission,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := packageHandler.GetRegistryCheckAccess(r.Context(), r, reqPermissions...)
			if err != nil {
				if errors.Is(err, apiauth.ErrUnauthorized) {
					setNugetAuthChallenge(r.Method, w)
					render.Unauthorized(r.Context(), w)
					return
				}
				render.TranslatedUserError(r.Context(), w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}
