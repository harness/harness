// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipelines

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/types"
	"github.com/harness/scm/types/check"

	"github.com/go-chi/chi"
	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/hlog"
)

// HandleUpdate returns an http.HandlerFunc that processes http
// requests to update the pipeline details.
func HandleUpdate(pipelines store.PipelineStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)
		id := chi.URLParam(r, "pipeline")

		pipeline, err := pipelines.FindSlug(ctx, id)
		if err != nil {
			render.NotFound(w, err)
			log.Debug().Err(err).
				Str("pipeline_slug", id).
				Msg("pipeline not found")
			return
		}

		in := new(types.PipelineInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Int64("pipeline_id", pipeline.ID).
				Str("pipeline_slug", pipeline.Slug).
				Msg("cannot unmarshal json request")
			return
		}

		if in.Name != nil {
			pipeline.Name = ptr.ToString(in.Name)
		}
		if in.Desc != nil {
			pipeline.Desc = ptr.ToString(in.Desc)
		}

		if ok, err := check.Pipeline(pipeline); !ok {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Int64("pipeline_id", pipeline.ID).
				Str("pipeline_slug", pipeline.Slug).
				Msg("cannot update pipeline")
			return
		}

		pipeline.Updated = time.Now().UnixMilli()

		err = pipelines.Update(ctx, pipeline)
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Int64("pipeline_id", pipeline.ID).
				Str("pipeline_slug", pipeline.Slug).
				Msg("cannot update the pipeline")
		} else {
			render.JSON(w, pipeline, 200)
		}
	}
}
