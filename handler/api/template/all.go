// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

import (
	"net/http"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
)

// HandleListAll returns an http.HandlerFunc that writes a json-encoded
// list of templates to the response body.
func HandleListAll(templateStore core.TemplateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := templateStore.ListAll(r.Context())
		if err != nil {
			render.NotFound(w, err)
			return
		}
		render.JSON(w, list, 200)
	}
}
