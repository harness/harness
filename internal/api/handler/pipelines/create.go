// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipelines

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/types"
	"github.com/bradrydzewski/my-app/types/check"

	"github.com/dchest/uniuri"
	"github.com/gosimple/slug"
	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/hlog"
)

// HandleCreate returns an http.HandlerFunc that creates
// a new pipeline.
func HandleCreate(pipelines store.PipelineStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		in := new(types.PipelineInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Msg("cannot unmarshal json request")
			return
		}

		pipeline := &types.Pipeline{
			Slug:    ptr.ToString(in.Slug),
			Name:    ptr.ToString(in.Name),
			Desc:    ptr.ToString(in.Desc),
			Token:   uniuri.NewLen(uniuri.UUIDLen),
			Created: time.Now().UnixMilli(),
			Updated: time.Now().UnixMilli(),
		}

		// if the slug is empty we can derrive
		// the slug from the pipeline name.
		if pipeline.Slug == "" {
			pipeline.Slug = slug.Make(pipeline.Name)
		}

		// if the name is empty we can derrive
		// the name from the pipeline slug.
		if pipeline.Name == "" {
			pipeline.Name = pipeline.Slug
		}

		if ok, err := check.Pipeline(pipeline); !ok {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Str("pipeline_slug", pipeline.Slug).
				Msg("cannot create pipeline")
			return
		}

		err = pipelines.Create(ctx, pipeline)
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Str("pipeline_name", pipeline.Name).
				Str("pipeline_slug", pipeline.Slug).
				Msg("cannot create pipeline")
			return
		}

		render.JSON(w, pipeline, 200)
	}
}
