// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package sign

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone-yaml/yaml/signer"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/core"

	"github.com/go-chi/chi"
)

type payload struct {
	Data string `json:"data"`
}

// HandleSign returns an http.HandlerFunc that processes http
// requests to sign a pipeline configuration file.
func HandleSign(repos core.RepositoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			namespace = chi.URLParam(r, "owner")
			name      = chi.URLParam(r, "name")
		)
		repo, err := repos.FindName(r.Context(), namespace, name)
		if err != nil {
			render.NotFound(w, err)
			return
		}

		in := new(payload)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			return
		}

		k := []byte(repo.Secret)
		d := []byte(in.Data)
		out, err := signer.Sign(d, k)
		if err != nil {
			render.InternalError(w, err)
			return
		}

		render.JSON(w, &payload{Data: out}, 200)
	}
}
