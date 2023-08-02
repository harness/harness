// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleDiff returns raw git diff for PR.
func HandleDiff(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		pullreqNumber, err := request.GetPullReqNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		setSHAs := func(sourceSHA, mergeBaseSHA string) {
			w.Header().Set("X-Source-Sha", sourceSHA)
			w.Header().Set("X-Merge-Base-Sha", mergeBaseSHA)
		}

		if err = pullreqCtrl.RawDiff(ctx, session, repoRef, pullreqNumber, setSHAs, w); err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		if strings.HasPrefix(r.Header.Get("Accept"), "text/plain") {
			if err = pullreqCtrl.RawDiff(ctx, session, repoRef, pullreqNumber, setSHAs, w); err != nil {
				render.TranslatedUserError(w, err)
			}
			return
		}

		stream, err := pullreqCtrl.Diff(ctx, session, repoRef, pullreqNumber)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSONArrayDynamic(ctx, w, stream)
	}
}
