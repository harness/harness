// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package executions

import (
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/api/request"
	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/types"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/hlog"
)

// HandleList returns an http.HandlerFunc that writes a json-encoded
// list of objects to the response body.
func HandleList(pipelines store.PipelineStore, executions store.ExecutionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx  = r.Context()
			log  = hlog.FromRequest(r)
			slug = chi.URLParam(r, "pipeline")
			page = request.ParsePage(r)
			size = request.ParseSize(r)
		)

		pipeline, err := pipelines.FindSlug(ctx, slug)
		if err != nil {
			render.NotFound(w, err)
			log.Debug().Err(err).
				Str("pipeline_slug", slug).
				Msg("pipeline not found")
			return
		}

		executions, err := executions.List(ctx, pipeline.ID, types.Params{
			Size: size,
			Page: page,
		})
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Int64("pipeline_id", pipeline.ID).
				Str("pipeline_slug", pipeline.Slug).
				Msg("cannot retrieve list")
		} else {
			render.Pagination(r, w, page, size, 0)
			render.JSON(w, executions, 200)
		}
	}
}
