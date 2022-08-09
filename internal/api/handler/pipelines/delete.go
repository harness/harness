// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipelines

import (
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/store"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/hlog"
)

// HandleDelete returns an http.HandlerFunc that deletes
// the object from the datastore.
func HandleDelete(pipelines store.PipelineStore) http.HandlerFunc {
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

		err = pipelines.Delete(ctx, pipeline)
		if err != nil {
			render.InternalError(w, err)
			hlog.FromRequest(r).
				Error().Err(err).
				Int64("pipeline_id", pipeline.ID).
				Str("pipeline_slug", pipeline.Slug).
				Msg("cannot delete pipeline")
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
