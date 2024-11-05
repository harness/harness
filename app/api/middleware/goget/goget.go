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

package goget

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	"github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/store"

	"github.com/rs/zerolog/log"
)

var (
	goGetTmpl *template.Template
	httpRegex = regexp.MustCompile("https?://")
)

func init() {
	var err error
	goGetTmpl, err = template.New("goget").Parse(`<!doctype html>
<html>
	<head>
		<meta name="go-import" content="{{.GoImport}}">
		<meta name="go-source" content="{{.GoSource}}">
	</head>
	<body>
		{{.GoCLI}}
	</body>
</html>`)
	if err != nil {
		panic(err)
	}
}

// Middleware writes to response with html meta tags go-import and go-source.
//
//nolint:gocognit
func Middleware(
	maxRepoPathDepth int,
	repoCtrl *repo.Controller,
	urlProvider url.Provider,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Query().Get("go-get") != "1" {
					next.ServeHTTP(w, r)
					return
				}
				ctx := r.Context()

				session, _ := request.AuthSessionFrom(ctx)
				importPath, err := request.GetRepoRefFromPath(r)
				if err != nil {
					render.TranslatedUserError(ctx, w, err)
					return
				}

				// go get can also be used with (sub)module suffixes (e.g 127.0.0.1/my-project/my-repo/v2)
				// for which go expects us to return the import path corresponding to the repository root
				// (e.g. 127.0.0.1/my-project/my-repo).
				//
				// WARNING: This can lead to ambiguities as we allow matching paths between spaces and repos:
				//  1. (space)foo     + (repo)bar + (sufix)v2 = 127.0.0.1/my-project/my-repo/v2
				//  2. (space)foo/bar + (repo)v2              = 127.0.0.1/my-project/my-repo/v2
				//
				// We handle ambiguities by always choosing the repo with the longest path (e.g. 2. case above).
				// To solve go get related ambiguities users would have to move their repositories.
				var repository *repo.RepositoryOutput
				var repoRef string

				pathSegments := paths.Segments(importPath)
				if len(pathSegments) > maxRepoPathDepth {
					pathSegments = pathSegments[:maxRepoPathDepth]
				}

				for l := len(pathSegments); l >= 2; l-- {
					repoRef = paths.Concatenate(pathSegments[:l]...)

					repository, err = repoCtrl.Find(ctx, session, repoRef)
					if err == nil {
						break
					}
					if errors.Is(err, store.ErrResourceNotFound) {
						log.Ctx(ctx).Debug().Err(err).
							Msgf("repository %q doesn't exist, assume submodule and try again", repoRef)
						continue
					}
					if errors.Is(err, auth.ErrNotAuthorized) {
						// To avoid leaking information about repos' existence we continue as if it wasn't found.
						// WARNING: This can lead to different import results depending on access (very unlikely)
						log.Ctx(ctx).Debug().Err(err).
							Msgf("user has no access on repository %q, assume submodule and try again", repoRef)
						continue
					}

					render.TranslatedUserError(ctx, w, fmt.Errorf("failed to find repo for path %q: %w", repoRef, err))
					return
				}

				if repository == nil {
					render.NotFound(ctx, w)
					return
				}

				uiRepoURL := urlProvider.GenerateUIRepoURL(ctx, repoRef)
				cloneURL := urlProvider.GenerateGITCloneURL(ctx, repoRef)
				goImportURL := httpRegex.ReplaceAllString(cloneURL, "")
				goImportURL = strings.TrimSuffix(goImportURL, ".git")
				prefix := fmt.Sprintf("%s/files/%s/~", uiRepoURL, repository.DefaultBranch)

				insecure := ""
				if strings.HasPrefix(uiRepoURL, "http:") {
					insecure = "--insecure"
				}

				goImportContent := fmt.Sprintf("%s git %s", goImportURL, cloneURL)
				goSourceContent := fmt.Sprintf("%s _ %s %s", goImportURL, prefix+"{/dir}", prefix+"{/dir}/{file}#L{line}")
				goGetCliContent := fmt.Sprintf("go get %s %s", insecure, goImportURL)
				err = goGetTmpl.Execute(w, struct {
					GoImport string
					GoSource string
					GoCLI    string
				}{
					GoImport: goImportContent,
					GoSource: goSourceContent,
					GoCLI:    goGetCliContent,
				})
				if err != nil {
					render.TranslatedUserError(ctx, w, err)
				}
			},
		)
	}
}
