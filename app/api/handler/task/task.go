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

package task

import (
	"errors"
	"io"
	"net/http"

	"github.com/harness/gitness/app/api/controller/task"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleCreate returns a http.HandlerFunc that creates a manual task.
func HandleCreate(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(task.CreateInput)
		if err = request.DecodeBody(r, in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		t, err := ctrl.Create(ctx, session, repoRef, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusCreated, t)
	}
}

// HandleList returns a http.HandlerFunc that lists the tasks of a repository.
func HandleList(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		filter, err := request.ParseTaskFilter(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		list, total, err := ctrl.List(ctx, session, repoRef, filter)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(total))
		render.JSON(w, http.StatusOK, list)
	}
}

// HandleListForSpace returns a http.HandlerFunc that lists the tasks of the repositories of a space.
func HandleListForSpace(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		filter, err := request.ParseTaskFilter(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		recursive, err := request.ParseRecursiveFromQuery(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		list, total, err := ctrl.ListForSpace(ctx, session, spaceRef, recursive, filter)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(total))
		render.JSON(w, http.StatusOK, list)
	}
}

// HandleFind returns a http.HandlerFunc that returns a task.
func HandleFind(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, taskNumber, ok := repoAndTask(w, r)
		if !ok {
			return
		}

		t, err := ctrl.Find(ctx, session, repoRef, taskNumber)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, t)
	}
}

// HandleUpdate returns a http.HandlerFunc that updates the editable fields of a task.
func HandleUpdate(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, taskNumber, ok := repoAndTask(w, r)
		if !ok {
			return
		}

		in := new(task.UpdateInput)
		if err := request.DecodeBody(r, in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		t, err := ctrl.Update(ctx, session, repoRef, taskNumber, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, t)
	}
}

// HandleState returns a http.HandlerFunc that changes the state of a task.
func HandleState(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, taskNumber, ok := repoAndTask(w, r)
		if !ok {
			return
		}

		in := new(task.StateInput)
		if err := request.DecodeBody(r, in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		t, err := ctrl.State(ctx, session, repoRef, taskNumber, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, t)
	}
}

// HandleResume returns a http.HandlerFunc that resumes a waiting or blocked task.
func HandleResume(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, taskNumber, ok := repoAndTask(w, r)
		if !ok {
			return
		}

		in := new(task.ResumeInput)
		// the body is optional
		if err := request.DecodeBody(r, in); err != nil && !errors.Is(err, io.EOF) {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		t, err := ctrl.Resume(ctx, session, repoRef, taskNumber, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, t)
	}
}

// HandleListActivities returns a http.HandlerFunc that lists the history of a task.
func HandleListActivities(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, taskNumber, ok := repoAndTask(w, r)
		if !ok {
			return
		}

		pagination := request.ParsePaginationFromRequest(r)

		list, err := ctrl.ListActivities(ctx, session, repoRef, taskNumber, pagination)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.PaginationNoTotal(r, w, pagination.Page, pagination.Size, len(list) < pagination.Size)
		render.JSON(w, http.StatusOK, list)
	}
}

// HandleCreateComment returns a http.HandlerFunc that adds a comment to a task.
func HandleCreateComment(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, taskNumber, ok := repoAndTask(w, r)
		if !ok {
			return
		}

		in := new(task.CommentInput)
		if err := request.DecodeBody(r, in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		if err := ctrl.CreateComment(ctx, session, repoRef, taskNumber, in); err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleListLinks returns a http.HandlerFunc that lists the links of a task.
func HandleListLinks(ctrl *task.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, taskNumber, ok := repoAndTask(w, r)
		if !ok {
			return
		}

		list, err := ctrl.ListLinks(ctx, session, repoRef, taskNumber)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, list)
	}
}

// repoAndTask extracts the repository reference and the task number from the path.
// It writes the error response and returns false if either is invalid.
func repoAndTask(w http.ResponseWriter, r *http.Request) (string, int64, bool) {
	repoRef, err := request.GetRepoRefFromPath(r)
	if err != nil {
		render.TranslatedUserError(r.Context(), w, err)
		return "", 0, false
	}

	taskNumber, err := request.GetTaskNumberFromPath(r)
	if err != nil {
		render.TranslatedUserError(r.Context(), w, err)
		return "", 0, false
	}

	return repoRef, taskNumber, true
}
