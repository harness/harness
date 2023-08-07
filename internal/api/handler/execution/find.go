// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"errors"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/controller/execution"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleFind finds a pipeline from the database.
func HandleFind(executionCtrl *execution.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		pipelineRef, err := request.GetPipelinePathRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		n, err := request.GetExecutionNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, pipelineUID, err := SplitRef(pipelineRef)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		execution, err := executionCtrl.Find(ctx, session, spaceRef, pipelineUID, n)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, execution)
	}
}

// SplitRef splits apart a ref into two parts, otherwise returns an error
// For example: path/to/space/uid will get split into path/to/space and uid
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
