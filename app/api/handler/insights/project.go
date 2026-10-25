// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package insights

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/app/api/controller/insights"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
)

// HandleCreateProject returns a http.HandlerFunc that creates an Insights project.
func HandleCreateProject(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(types.InsightsProjectCreateInput)
		if err = json.NewDecoder(r.Body).Decode(in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		project, err := ctrl.CreateProject(ctx, session, repoRef, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusCreated, project)
	}
}

// HandleListProjects returns a http.HandlerFunc that lists the Insights projects of a repository.
func HandleListProjects(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		projects, err := ctrl.ListProjects(ctx, session, repoRef)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, projects)
	}
}

// HandleUpdateProject returns a http.HandlerFunc that updates an Insights project.
func HandleUpdateProject(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		projectID, err := request.GetInsightsProjectIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(types.InsightsProjectUpdateInput)
		if err = json.NewDecoder(r.Body).Decode(in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		project, err := ctrl.UpdateProject(ctx, session, repoRef, projectID, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, project)
	}
}

// HandleRotateProjectKey returns a http.HandlerFunc that rotates the public key of an Insights project.
func HandleRotateProjectKey(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		projectID, err := request.GetInsightsProjectIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		project, err := ctrl.RotateProjectKey(ctx, session, repoRef, projectID)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, project)
	}
}

// HandleDeleteProject returns a http.HandlerFunc that deletes an Insights project.
func HandleDeleteProject(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		projectID, err := request.GetInsightsProjectIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		if err = ctrl.DeleteProject(ctx, session, repoRef, projectID); err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
