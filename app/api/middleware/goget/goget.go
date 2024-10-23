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
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/types"
)

var goGetTmpl *template.Template
var httpRegex = regexp.MustCompile("https?://")

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
func Middleware(
	config *types.Config,
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
				repoRef, err := request.GetRepoRefFromPath(r)
				if err != nil {
					render.TranslatedUserError(ctx, w, err)
					return
				}

				repository, err := repoCtrl.Find(ctx, session, repoRef)
				if err != nil {
					render.TranslatedUserError(ctx, w, err)
					return
				}

				defaultBranch := config.Git.DefaultBranch
				if repository.DefaultBranch != "" {
					defaultBranch = repository.DefaultBranch
				}

				uiRepoURL := urlProvider.GenerateUIRepoURL(ctx, repoRef)
				goImportURL := httpRegex.ReplaceAllString(uiRepoURL, "")
				cloneURL := urlProvider.GenerateGITCloneURL(ctx, repoRef)
				prefix := fmt.Sprintf("%s/files/%s/~", uiRepoURL, defaultBranch)

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
