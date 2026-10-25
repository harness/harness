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

// Package shield provides the HTTP handlers of Shield.
package shield

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/api/controller/shield"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	// PathParamFindingID is the name of the finding id path parameter.
	PathParamFindingID = "finding_id"

	queryParamMinRisk = "min_risk"
)

// HandleCreateScan returns a http.HandlerFunc that schedules a manual scan.
func HandleCreateScan(ctrl *shield.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(shield.ScanInput)
		// the body is optional.
		if err = json.NewDecoder(r.Body).Decode(in); err != nil && !errors.Is(err, io.EOF) {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		scan, err := ctrl.CreateScan(ctx, session, repoRef, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusAccepted, scan)
	}
}

// HandleListScans returns a http.HandlerFunc that lists the scans of a repository.
func HandleListScans(ctrl *shield.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		page, size := request.ParsePage(r), request.ParseLimit(r)
		scans, err := ctrl.ListScans(ctx, session, repoRef, page, size)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, scans)
	}
}

// ParseFindingFilter parses the finding list query parameters.
func ParseFindingFilter(r *http.Request) (*types.ShieldFindingFilter, error) {
	filter := &types.ShieldFindingFilter{
		Page: request.ParsePage(r),
		Size: request.ParseLimit(r),
	}

	for _, s := range queryList(r, request.QueryParamState) {
		state, ok := enum.ShieldFindingState(s).Sanitize()
		if !ok {
			return nil, usererror.BadRequestf("Unknown finding state %q", s)
		}
		filter.States = append(filter.States, state)
	}

	for _, k := range queryList(r, request.QueryParamKind) {
		kind, ok := enum.ShieldFindingKind(k).Sanitize()
		if !ok {
			return nil, usererror.BadRequestf("Unknown finding kind %q", k)
		}
		filter.Kinds = append(filter.Kinds, kind)
	}

	if raw := r.URL.Query().Get(queryParamMinRisk); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 100 {
			return nil, usererror.BadRequestf("Parameter '%s' must be an integer between 0 and 100", queryParamMinRisk)
		}
		filter.MinRiskScore = v
	}

	return filter, nil
}

// queryList returns the values of a query parameter that may be repeated or comma separated.
func queryList(r *http.Request, name string) []string {
	var out []string
	for _, v := range r.URL.Query()[name] {
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// HandleListFindings returns a http.HandlerFunc that lists the findings of a repository.
func HandleListFindings(ctrl *shield.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		filter, err := ParseFindingFilter(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		findings, total, err := ctrl.ListFindings(ctx, session, repoRef, filter)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(total))
		render.JSON(w, http.StatusOK, findings)
	}
}

// HandleUpdateFinding returns a http.HandlerFunc that ignores or accepts the risk of a finding.
func HandleUpdateFinding(ctrl *shield.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		id, err := request.PathParamAsPositiveInt64(r, PathParamFindingID)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(shield.FindingUpdateInput)
		if err = json.NewDecoder(r.Body).Decode(in); err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		finding, err := ctrl.UpdateFinding(ctx, session, repoRef, id, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, finding)
	}
}
