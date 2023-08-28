// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package plugin

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/plugin"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

func HandleList(pluginCtrl *plugin.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		filter := request.ParseListQueryFilterFromRequest(r)
		ret, totalCount, err := pluginCtrl.List(ctx, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(totalCount))
		render.JSON(w, http.StatusOK, ret)
	}
}
