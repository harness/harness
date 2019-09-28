// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package user

import (
	"context"
	"testing"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock/mockscm"
	"github.com/drone/go-scm/scm"
	"github.com/google/go-cmp/cmp"

	"github.com/golang/mock/gomock"
)

var noContext = context.Background()

func TestFind(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	checkToken := func(ctx context.Context) {
		got, ok := ctx.Value(scm.TokenKey{}).(*scm.Token)
		if !ok {
			t.Errorf("Expect token stored in context")
			return
		}
		want := &scm.Token{
			Token:   "755bb80e5b",
			Refresh: "e08f3fa43e",
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf(diff)
		}
	}

	now := time.Now()
	mockUser := &scm.User{
		Login:   "octocat",
		Email:   "octocat@github.com",
		Avatar:  "https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
		Created: now,
		Updated: now,
	}
	mockUsers := mockscm.NewMockUserService(controller)
	mockUsers.EXPECT().Find(gomock.Any()).Do(checkToken).Return(mockUser, nil, nil)

	client := new(scm.Client)
	client.Users = mockUsers

	want := &core.User{
		Login:   "octocat",
		Email:   "octocat@github.com",
		Avatar:  "https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
		Created: now.Unix(),
		Updated: now.Unix(),
	}
	got, err := New(client, nil).Find(noContext, "755bb80e5b", "e08f3fa43e")
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestFind_Error(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUsers := mockscm.NewMockUserService(controller)
	mockUsers.EXPECT().Find(gomock.Any()).Return(nil, nil, scm.ErrNotFound)

	client := new(scm.Client)
	client.Users = mockUsers

	got, err := New(client, nil).Find(noContext, "755bb80e5b", "e08f3fa43e")
	if err == nil {
		t.Errorf("Expect error finding user")
	}
	if got != nil {
		t.Errorf("Expect nil user on error")
	}
}
