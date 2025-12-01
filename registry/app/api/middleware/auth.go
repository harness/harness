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
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	registryauth "github.com/harness/gitness/registry/app/auth"
	"github.com/harness/gitness/registry/app/common"
	"github.com/harness/gitness/registry/app/pkg"
	gitnessenum "github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func OciCheckAuth(h *oci.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				session, _ := request.AuthSessionFrom(ctx)
				//nolint:nestif
				if auth.IsAnonymousSession(session) {
					url := common.GenerateOciTokenURL(h.URLProvider.RegistryURL(ctx))
					info, err := h.GetRegistryInfo(r, true)
					scope := getScope(r)

					if strings.Contains(scope, ":push,") ||
						strings.HasSuffix(scope, ",push") ||
						strings.HasSuffix(r.RequestURI, "/v2/") ||
						r.Header.Get("Authorization") == "" {
						returnUnauthorised(ctx, w, url, scope)
						return
					}

					if err != nil {
						log.Ctx(ctx).Error().Stack().Str("middleware",
							"OciCheckAuth").Err(err).Msgf("error while fetching the artifact info: %v",
							err)
						returnUnauthorised(ctx, w, url, scope)
						return
					}

					space, err := h.SpaceFinder.FindByID(ctx, info.ParentID)
					if err != nil {
						log.Ctx(ctx).Error().Stack().Str("middleware",
							"OciCheckAuth").Err(err).Msgf("error while fetching the space with ID: %d err: %v", info.ParentID,
							err)
						returnUnauthorised(ctx, w, url, scope)
						return
					}

					isPublic, err := h.PublicAccessService.Get(ctx,
						gitnessenum.PublicResourceTypeRegistry, space.Path+"/"+info.RegIdentifier)
					if err != nil {
						log.Ctx(ctx).Error().Stack().Str("middleware",
							"OciCheckAuth").Err(err).Msgf("failed to check if public access is supported err: %v",
							err)
						returnUnauthorised(ctx, w, url, scope)
						return
					}
					if !isPublic {
						returnUnauthorised(ctx, w, url, scope)
						return
					}
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
				if session, oks := request.AuthSessionFrom(ctx); oks && !auth.IsAnonymousSession(session) {
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

func CheckSig() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// Check if the sig query parameter exists
				if token := r.URL.Query().Get("sig"); token != "" {
					// Set the token as a Bearer token in the Authorization header
					token = strings.TrimPrefix(token, "Bearer ")
					token = strings.TrimPrefix(token, "Basic ")
					r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func CheckAuthWithChallenge(
	p pkg.ArtifactInfoProvider,
	spaceFinder refcache.SpaceFinder,
	publicAccessService publicaccess.Service,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				session, _ := request.AuthSessionFrom(ctx)
				if auth.IsAnonymousSession(session) {
					info, err := p.GetPackageArtifactInfo(r)
					if err != nil {
						log.Ctx(ctx).Error().Stack().Str("middleware",
							"CheckAuthWithChallenge").Err(err).Msgf("error while fetching the artifact info: %v",
							err)
						setAuthenticateHeader(w)
						render.Unauthorized(ctx, w)
						return
					}
					space, err := spaceFinder.FindByID(ctx, info.BaseArtifactInfo().ParentID)
					if err != nil {
						log.Ctx(ctx).Error().Stack().Str("middleware",
							"CheckAuthWithChallenge").Err(err).
							Msgf("error while fetching the space with ID: %d err: %v",
								info.BaseArtifactInfo().ParentID,
								err)
						setAuthenticateHeader(w)
						render.Unauthorized(ctx, w)
						return
					}

					isPublic, err := publicAccessService.Get(ctx,
						gitnessenum.PublicResourceTypeRegistry, space.Path+"/"+info.BaseArtifactInfo().RegIdentifier)
					if !isPublic || err != nil {
						setAuthenticateHeader(w)
						render.Unauthorized(ctx, w)
						return
					}
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func CheckAuthHeader() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				apiKeyHeader := r.Header.Get("x-api-key")
				if apiKeyHeader != "" {
					r.Header.Set("Authorization", apiKeyHeader)
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func CheckNugetAPIKey() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				apiKeyHeader := r.Header.Get("x-nuget-apikey")
				if apiKeyHeader != "" {
					r.Header.Set("Authorization", apiKeyHeader)
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func setNugetAuthChallenge(method string, w http.ResponseWriter) {
	if method == http.MethodPut || method == http.MethodDelete {
		w.Header().Set("WWW-Authenticate", "ApiKey realm=\"Harness Registry\"")
	} else {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Harness Registry\"")
	}
}

func setAuthenticateHeader(w http.ResponseWriter) {
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
		rootIdentifier, registryIdentifier, _, _, _, _ := oci.ExtractPathVars(r.Context(), path, paramMap)
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
	header := fmt.Sprintf(`Bearer realm="%s",service="gitness-registry"`, url)
	if scope != "" {
		header = fmt.Sprintf(`%s,scope="%s"`, header, scope)
	}
	w.Header().Set("WWW-Authenticate", header)
	render.Unauthorized(ctx, w)
}
