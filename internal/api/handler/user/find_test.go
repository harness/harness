// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/mocks"
	"github.com/harness/gitness/types"

	"github.com/google/go-cmp/cmp"
)

func TestFind(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &types.User{
		ID:    1,
		Email: "octocat@github.com",
	}

	users := mocks.NewMockUserStore(controller)
	users.EXPECT().Find(gomock.Any(), mockUser.ID).Return(mockUser, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/user", nil)
	r = r.WithContext(
		request.WithAuthSession(
			request.WithUser(
				r.Context(),
				mockUser),
			&auth.Session{Principal: *types.PrincipalFromUser(mockUser), Metadata: &auth.EmptyMetadata{}}),
	)

	HandleFind(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := &types.User{}, mockUser
	if err := json.NewDecoder(w.Body).Decode(got); err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}
