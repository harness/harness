// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package executions

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/types"
	"github.com/bradrydzewski/my-app/types/check"
	"github.com/go-chi/chi"
	"github.com/gosimple/slug"
	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/hlog"
)

// HandleCreate returns an http.HandlerFunc that creates
// the object and persists to the datastore.
func HandleCreate(pipelines store.PipelineStore, executions store.ExecutionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx   = r.Context()
			log   = hlog.FromRequest(r)
			param = chi.URLParam(r, "pipeline")
		)

		pipeline, err := pipelines.FindSlug(ctx, param)
		if err != nil {
			render.NotFound(w, err)
			log.Debug().Err(err).
				Str("pipeline_slug", param).
				Msg("pipeline not found")
			return
		}

		sublog := log.With().
			Int64("pipeline_id", pipeline.ID).
			Str("pipeline_slug", pipeline.Slug).
			Logger()

		in := new(types.ExecutionInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			sublog.Debug().Err(err).
				Msg("cannot unmarshal json request")
			return
		}

		execution := &types.Execution{
			Pipeline: pipeline.ID,
			Slug:     ptr.ToString(in.Slug),
			Name:     ptr.ToString(in.Name),
			Desc:     ptr.ToString(in.Desc),
			Created:  time.Now().UnixMilli(),
			Updated:  time.Now().UnixMilli(),
		}

		// if the slug is empty we can derrive
		// the slug from the name.
		if execution.Slug == "" {
			execution.Slug = slug.Make(execution.Name)
		}

		// if the name is empty we can derrive
		// the name from the slug.
		if execution.Name == "" {
			execution.Name = execution.Slug
		}

		if ok, err := check.Execution(execution); !ok {
			render.BadRequest(w, err)
			sublog.Debug().Err(err).
				Int64("execution_id", execution.ID).
				Str("execution_slug", execution.Slug).
				Msg("cannot validate execution")
			return
		}

		err = executions.Create(ctx, execution)
		if err != nil {
			render.InternalError(w, err)
			sublog.Error().Err(err).
				Int64("execution_id", execution.ID).
				Str("execution_slug", execution.Slug).
				Msg("cannot create execution")
		} else {
			render.JSON(w, execution, 200)
		}
	}
}
