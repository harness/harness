// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package user

import (
	"context"
	"net/http"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/logger"
)

// HandleSync returns an http.HandlerFunc synchronizes and then
// write a json-encoded list of repositories to the response body.
func HandleSync(syncer core.Syncer, repos core.RepositoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, _ := request.UserFrom(r.Context())

		// performs asynchronous account synchronization.
		// this requires long polling to determine when the
		// sync is complete.
		if r.FormValue("async") == "true" {
			ctx := context.Background()
			go func(ctx context.Context, viewer *core.User) {
				_, err := syncer.Sync(ctx, viewer)
				if err != nil {
					logger.FromContext(ctx).WithError(err).
						Debugln("api: cannot synchronize account")
				}
			}(ctx, viewer)
			w.WriteHeader(204)
			return
		}

		_, err := syncer.Sync(r.Context(), viewer)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("api: cannot synchronize account")
			return
		}
		list, err := repos.List(r.Context(), viewer.ID)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("api: cannot synchronize account")
		} else {
			render.JSON(w, list, 200)
		}
	}
}
