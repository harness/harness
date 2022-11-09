// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/harness/gitness/types"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleFind writes json-encoded repository information to the http response body.
func HandleFind(repoCtrl *repo.Controller, config *types.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		repo, err := repoCtrl.Find(ctx, session, repoRef, config)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		parse, err := url.Parse(repo.URL)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		if parse.Host == "" {
			parse.Host = r.Host
		}

		if parse.Scheme == "" {
			parse.Scheme = "http"
			if !strings.Contains(parse.Host, "localhost") {
				parse.Scheme = "https"
			}
		}

		repo.URL = parse.String()

		render.JSON(w, http.StatusOK, repo)
	}
}
