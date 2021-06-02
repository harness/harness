// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
)

type templateInput struct {
	Name      string `json:"name"`
	Data      string `json:"data"`
	Namespace string `json:"namespace"`
}

// HandleCreate returns an http.HandlerFunc that processes http
// requests to create a new template.
func HandleCreate(templateStore core.TemplateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := new(templateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			return
		}

		t := &core.Template{
			Name:      in.Name,
			Data:      in.Data,
			Namespace: in.Namespace,
		}

		err = t.Validate()
		if err != nil {
			render.BadRequest(w, err)
			return
		}

		err = templateStore.Create(r.Context(), t)
		if err != nil {
			render.InternalError(w, err)
			return
		}

		render.JSON(w, t, 200)
	}
}
