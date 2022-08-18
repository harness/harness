// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipelines

import (
	"net/http"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/types"
	"github.com/rs/zerolog/hlog"

	"github.com/go-chi/chi"
)

type pipelineToken struct {
	*types.Pipeline
	Token string `json:"token"`
}

// HandleFind returns an http.HandlerFunc that writes the
// json-encoded pipeline details to the response body.
func HandleFind(pipelines store.PipelineStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "pipeline")

		pipeline, err := pipelines.FindSlug(ctx, id)
		if err != nil {
			render.NotFound(w, err)
			hlog.FromRequest(r).
				Debug().Err(err).
				Str("pipeline_slug", id).
				Msg("pipeline not found")
			return
		}

		// if the caller requests the pipeline details without
		// the token, return the token object as-is.
		if r.FormValue("token") != "true" {
			render.JSON(w, pipeline, 200)
			return
		}

		// if the caller requests the pipeline details with
		// the token then it can be safely included.
		render.JSON(w, &pipelineToken{pipeline, pipeline.Token}, 200)
	}
}
