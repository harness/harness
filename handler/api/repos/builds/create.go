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
	"encoding/json"
	"net/http"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/go-scm/scm"

	"github.com/go-chi/chi"
)

type createBuild struct {
	Params map[string]string `json:"params"`
	Commit *string           `json:"commit"`
	Branch *string           `json:"branch"`
	Ref    *string           `json:"ref"`
}

// HandleCreate returns an http.HandlerFunc that processes http
// requests to create a build for the specified commit.
func HandleCreate(
	repos core.RepositoryStore,
	commits core.CommitService,
	triggerer core.Triggerer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx       = r.Context()
			namespace = chi.URLParam(r, "owner")
			name      = chi.URLParam(r, "name")
			user, _   = request.UserFrom(ctx)
		)
		in := new(createBuild)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			return
		}
		if in.Ref == nil || in.Branch == nil {
			render.BadRequestf(w, "Missing branch or ref")
			return
		}
		if in.Params == nil {
			// cannot remember if parameters must be non-nil,
			// so we set the value to prevent a possible
			// downstream nil pointer. just being overly cautious ...
			in.Params = map[string]string{}
		}
		repo, err := repos.FindName(ctx, namespace, name)
		if err != nil {
			render.NotFound(w, err)
			return
		}
		var commit *core.Commit
		if in.Commit == nil {
			commit, err = commits.Find(ctx, user, repo.Slug, *in.Commit)
		} else if in.Ref != nil {
			commit, err = commits.FindRef(ctx, user, repo.Slug, *in.Ref)
		} else if in.Branch != nil {
			ref := scm.ExpandRef(*in.Branch, "refs/heads")
			commit, err = commits.FindRef(ctx, user, repo.Slug, ref)
		}
		if err != nil {
			render.NotFound(w, err)
			return
		}

		hook := &core.Hook{
			Trigger:      user.Login,
			Event:        core.EventPush,
			Link:         commit.Link,
			Timestamp:    commit.Author.Date,
			Title:        "", // we expect this to be empty.
			Message:      commit.Message,
			Before:       "", // we expect this to be empty.
			After:        commit.Sha,
			Ref:          "", // set below
			Source:       "", // set below
			Target:       "", // set below
			Author:       commit.Author.Login,
			AuthorName:   commit.Author.Name,
			AuthorEmail:  commit.Author.Email,
			AuthorAvatar: commit.Author.Avatar,
			Sender:       "", // todo: what value should we use?
			Params:       in.Params,
		}
		if in.Branch != nil {
			hook.Source = *in.Branch
			hook.Target = *in.Branch
		}
		if in.Ref != nil {
			branch := scm.TrimRef(*in.Ref)
			hook.Source = branch
			hook.Target = branch
		}

		result, err := triggerer.Trigger(r.Context(), repo, hook)
		if err != nil {
			render.InternalError(w, err)
		} else {
			render.JSON(w, result, 200)
		}
	}
}
