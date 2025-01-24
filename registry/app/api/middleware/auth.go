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
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	registryauth "github.com/harness/gitness/registry/app/auth"
	"github.com/harness/gitness/registry/app/common"

	"github.com/rs/zerolog/log"
)

func OciCheckAuth(urlProvider url.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				session, _ := request.AuthSessionFrom(ctx)
				url := common.GenerateOciTokenURL(urlProvider.RegistryURL(ctx))
				if session.Principal == auth.AnonymousPrincipal {
					scope := getScope(r)
					returnUnauthorised(ctx, w, url, scope)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

// BlockNonOciSourceToken blocks any request that doesn't have AccessPermissionMetadata.
func BlockNonOciSourceToken(urlProvider url.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				if session, oks := request.AuthSessionFrom(ctx); oks {
					if metadata, okt := session.Metadata.(*auth.AccessPermissionMetadata); !okt ||
						metadata.AccessPermissions.Source != jwt.OciSource {
						log.Ctx(ctx).Warn().
							Msg("blocking request - non OCI source tokens are not allowed for usage with oci endpoints")
						scope := getScope(r)
						url := common.GenerateOciTokenURL(urlProvider.RegistryURL(ctx))
						returnUnauthorised(ctx, w, url, scope)
						return
					}
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func CheckAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				session, _ := request.AuthSessionFrom(ctx)
				if session.Principal == auth.AnonymousPrincipal {
					render.Unauthorized(ctx, w)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func CheckMavenAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				session, _ := request.AuthSessionFrom(ctx)
				if session.Principal == auth.AnonymousPrincipal {
					setMavenHeaders(w)
					render.Unauthorized(ctx, w)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func setMavenHeaders(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic realm=\"Harness Registry\"")
}

func getRefsFromName(name string) (spaceRef, repoRef string) {
	name = strings.Trim(name, "/")
	refs := strings.Split(name, "/")
	spaceRef = refs[0]
	repoRef = refs[1]
	return
}

func getScope(r *http.Request) string {
	var scope string
	path := r.URL.Path
	if path != "/v2/" && path != "/v2/token" {
		paramMap := common.ExtractFirstQueryParams(r.URL.Query())
		rootIdentifier, registryIdentifier, _, _, _, _ := oci.ExtractPathVars(path, paramMap)
		var access []registryauth.Access
		access = registryauth.AppendAccess(access, r.Method, rootIdentifier+"/"+registryIdentifier)
		if fromRepo := r.FormValue("from"); fromRepo != "" {
			space, repoName := getRefsFromName(fromRepo)
			access = registryauth.AppendAccess(access, http.MethodGet, space+"/"+repoName)
		}
		scope = registryauth.NewAccessSet(access...).ScopeParam()
	}
	return scope
}

func returnUnauthorised(ctx context.Context, w http.ResponseWriter, url string, scope string) {
	header := fmt.Sprintf(`Bearer realm="%s", service="gitness-registry"`, url)
	if scope != "" {
		header = fmt.Sprintf(`%s, scope="%s"`, header, scope)
	}
	w.Header().Set("WWW-Authenticate", header)
	render.Unauthorized(ctx, w)
}
