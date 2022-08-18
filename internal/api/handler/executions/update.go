// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package executions

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
// requests to update the object details.
func HandleUpdate(pipelines store.PipelineStore, executions store.ExecutionStore) http.HandlerFunc {
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

		sublog := log.With().
			Int64("pipeline_id", pipeline.ID).
			Str("pipeline_slug", pipeline.Slug).
			Logger()

		execution, err := executions.FindSlug(ctx, pipeline.ID, executionParam)
		if err != nil {
			render.NotFound(w, err)
			sublog.Debug().Err(err).
				Str("execution_slug", executionParam).
				Msg("execution not found")
			return
		}

		sublog = sublog.With().
			Str("execution_slug", execution.Slug).
			Int64("execution_id", execution.ID).
			Logger()

		in := new(types.ExecutionInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			sublog.Debug().Err(err).
				Msg("cannot unmarshal json request")
			return
		}

		if in.Name != nil {
			execution.Name = ptr.ToString(in.Name)
		}
		if in.Desc != nil {
			execution.Desc = ptr.ToString(in.Desc)
		}

		if ok, err := check.Execution(execution); !ok {
			render.BadRequest(w, err)
			sublog.Debug().Err(err).
				Msg("cannot validate execution")
			return
		}

		execution.Updated = time.Now().UnixMilli()

		err = executions.Update(ctx, execution)
		if err != nil {
			render.InternalError(w, err)
			sublog.Error().Err(err).
				Msg("cannot update execution")
		} else {
			render.JSON(w, execution, 200)
		}
	}
}
