// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipelines

import (
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/api/request"
	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/types"

	"github.com/rs/zerolog/log"
)

// HandleList returns an http.HandlerFunc that writes a json-encoded
// list of pipelines to the response body.
func HandleList(pipelines store.PipelineStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			page = request.ParsePage(r)
			size = request.ParseSize(r)
			ctx  = r.Context()
		)

		viewer, _ := request.UserFrom(ctx)
		list, err := pipelines.List(ctx, viewer.ID, types.Params{
			Page: page,
			Size: size,
		})
		if err != nil {
			render.InternalError(w, err)
			log.Ctx(ctx).Error().
				Err(err).Msg("cannot list pipelines")
		} else {
			render.Pagination(r, w, page, size, 0)
			render.JSON(w, list, 200)
		}
	}
}
