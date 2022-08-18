// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipelines

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/api/request"
	"github.com/harness/scm/mocks"
	"github.com/harness/scm/types"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestHandleList(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &types.User{
		ID:    1,
		Email: "octocat@github.com",
	}

	mockList := []*types.Pipeline{
		{
			Name: "test",
			Desc: "desc",
		},
	}

	projs := mocks.NewMockPipelineStore(controller)
	projs.EXPECT().List(gomock.Any(), mockUser.ID, gomock.Any()).Return(mockList, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		request.WithUser(r.Context(), mockUser),
	)

	HandleList(projs)(w, r)
	if got, want := w.Code, http.StatusOK; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := []*types.Pipeline{}, mockList
	json.NewDecoder(w.Body).Decode(&got)
	if diff := cmp.Diff(got, want); len(diff) > 0 {
		t.Errorf(diff)
	}
}

func TestListErr(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &types.User{
		ID:    1,
		Email: "octocat@github.com",
	}

	projs := mocks.NewMockPipelineStore(controller)
	projs.EXPECT().List(gomock.Any(), mockUser.ID, gomock.Any()).Return(nil, render.ErrNotFound)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		request.WithUser(r.Context(), mockUser),
	)

	HandleList(projs)(w, r)
	if got, want := w.Code, http.StatusInternalServerError; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := &render.Error{}, render.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) > 0 {
		t.Errorf(diff)
	}
}
