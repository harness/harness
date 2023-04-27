// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleRawDiff returns raw git diff for PR.
func HandleRawDiff(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
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

		w.WriteHeader(http.StatusOK)
	}
}
