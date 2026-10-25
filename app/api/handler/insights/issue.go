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

// HandleListIssues returns a http.HandlerFunc that lists the Insights issues of a repository.
func HandleListIssues(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		filter, err := request.ParseInsightsIssueFilter(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		issues, total, err := ctrl.ListIssues(ctx, session, repoRef, filter)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(total))
		render.JSON(w, http.StatusOK, issues)
	}
}

// HandleFindIssue returns a http.HandlerFunc that returns an Insights issue.
func HandleFindIssue(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		issueID, err := request.GetInsightsIssueIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		issue, err := ctrl.FindIssue(ctx, session, repoRef, issueID)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, issue)
	}
}

// HandleListIssueEvents returns a http.HandlerFunc that lists the events of an Insights issue.
func HandleListIssueEvents(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		issueID, err := request.GetInsightsIssueIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		pagination := request.ParsePaginationFromRequest(r)

		events, total, err := ctrl.ListIssueEvents(ctx, session, repoRef, issueID, pagination)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.Pagination(r, w, pagination.Page, pagination.Size, int(total))
		render.JSON(w, http.StatusOK, events)
	}
}

// HandleUpdateIssue returns a http.HandlerFunc that updates the status of an Insights issue.
func HandleUpdateIssue(ctrl *insights.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		issueID, err := request.GetInsightsIssueIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(types.InsightsIssueUpdateInput)
		if err = json.NewDecoder(r.Body).Decode(in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		issue, err := ctrl.UpdateIssue(ctx, session, repoRef, issueID, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, issue)
	}
}
