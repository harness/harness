// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package executions

import (
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/store"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/hlog"
)

// HandleFind returns an http.HandlerFunc that writes the
// json-encoded execution details to the response body.
func HandleFind(pipelines store.PipelineStore, executions store.ExecutionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx            = r.Context()
			log            = hlog.FromRequest(r)
			pipelineParam  = chi.URLParam(r, "pipeline")
			executionParam = chi.URLParam(r, "execution")
		)

		pipeline, err := pipelines.FindSlug(ctx, pipelineParam)
		if err != nil {
			render.NotFound(w, err)
			log.Debug().Err(err).
				Str("pipeline_slug", pipelineParam).
				Msg("pipeline not found")
			return
		}

		execution, err := executions.FindSlug(ctx, pipeline.ID, executionParam)
		if err != nil {
			render.NotFound(w, err)
			log.Debug().Err(err).
				Int64("pipeline_id", pipeline.ID).
				Str("pipeline_slug", pipeline.Slug).
				Str("execution_slug", executionParam).
				Msg("execution not found")
			return
		}

		render.JSON(w, execution, 200)
	}
}
