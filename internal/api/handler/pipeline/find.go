// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"errors"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/controller/pipeline"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleFind writes json-encoded repository information to the http response body.
func HandleFind(pipelineCtrl *pipeline.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		pipelineRef, err := request.GetPipelinePathRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, pipelineUID, err := SplitRef(pipelineRef)
		if err != nil {
			render.TranslatedUserError(w, err)
		}

		pipeline, err := pipelineCtrl.Find(ctx, session, spaceRef, pipelineUID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, pipeline)
	}
}

func SplitRef(ref string) (string, string, error) {
	lastIndex := strings.LastIndex(ref, "/")
	if lastIndex == -1 {
		// The input string does not contain a "/".
		return "", "", errors.New("could not split ref")
	}

	spaceRef := ref[:lastIndex]
	uid := ref[lastIndex+1:]

	return spaceRef, uid, nil
}
