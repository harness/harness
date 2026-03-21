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

package builds

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/trigger/dag"

	"github.com/go-chi/chi"
)

// HandleRetry returns an http.HandlerFunc that processes http
// requests to retry and re-execute a build.
//
// Optional query parameters:
//   - stage: a single pipeline name to retry. When set, only this pipeline
//     (and its dependents, if cascade is true) is included in the new build.
	//   - cascade: when "true" and stage is set, automatically include all
	//     downstream dependents of the requested stage. Ignored when stage
	//     is not provided.
func HandleRetry(
	repos core.RepositoryStore,
	builds core.BuildStore,
	stages core.StageStore,
	triggerer core.Triggerer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			namespace = chi.URLParam(r, "owner")
			name      = chi.URLParam(r, "name")
			user, _   = request.UserFrom(r.Context())
		)
		number, err := strconv.ParseInt(chi.URLParam(r, "number"), 10, 64)
		if err != nil {
			render.BadRequest(w, err)
			return
		}
		repo, err := repos.FindName(r.Context(), namespace, name)
		if err != nil {
			render.NotFound(w, err)
			return
		}
		prev, err := builds.FindNumber(r.Context(), repo.ID, number)
		if err != nil {
			render.NotFound(w, err)
			return
		}

		switch prev.Status {
		case core.StatusBlocked:
			render.BadRequestf(w, "cannot start a blocked build")
			return
		case core.StatusDeclined:
			render.BadRequestf(w, "cannot start a declined build")
			return
		}

		hook := &core.Hook{
			Parent:       prev.Number,
			Trigger:      user.Login,
			Event:        prev.Event,
			Action:       prev.Action,
			Link:         prev.Link,
			Timestamp:    prev.Timestamp,
			Title:        prev.Title,
			Message:      prev.Message,
			Before:       prev.Before,
			After:        prev.After,
			Ref:          prev.Ref,
			Fork:         prev.Fork,
			Source:       prev.Source,
			Target:       prev.Target,
			Author:       prev.Author,
			AuthorName:   prev.AuthorName,
			AuthorEmail:  prev.AuthorEmail,
			AuthorAvatar: prev.AuthorAvatar,
			Deployment:   prev.Deploy,
			DeploymentID: prev.DeployID,
			Debug:        r.FormValue("debug") == "true",
			Cron:         prev.Cron,
			Sender:       prev.Sender,
			Params:       map[string]string{},
		}

		if stageName := strings.TrimSpace(r.FormValue("stage")); stageName != "" {
			hook.Stages = []string{stageName}
			if r.FormValue("cascade") == "true" {
				prevStages, err := stages.List(r.Context(), prev.ID)
				if err != nil {
					render.InternalError(w, err)
					return
				}
				found := false
				d := dag.New()
				for _, s := range prevStages {
					d.Add(s.Name, s.DependsOn...)
					if s.Name == stageName {
						found = true
					}
				}
				if !found {
					render.BadRequestf(w, "stage %q not found in build #%d", stageName, prev.Number)
					return
				}
				for _, desc := range d.Descendants(stageName) {
					hook.Stages = append(hook.Stages, desc)
				}
			}
		}

		for key, value := range r.URL.Query() {
			if key == "access_token" {
				continue
			}
			if key == "debug" {
				continue
			}
			if key == "stage" {
				continue
			}
			if key == "cascade" {
				continue
			}
			if len(value) == 0 {
				continue
			}
			hook.Params[key] = value[0]
		}
		for key, value := range prev.Params {
			hook.Params[key] = value
		}

		result, err := triggerer.Trigger(r.Context(), repo, hook)
		if err != nil {
			render.InternalError(w, err)
		} else {
			render.JSON(w, result, 200)
		}
	}
}
