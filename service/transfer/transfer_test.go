// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package transfer

import (
	"context"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"

	"github.com/golang/mock/gomock"
)

var nocontext = context.Background()

func TestTransfer(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:     1,
		UserID: 2,
		UID:    "123",
	}
	mockRepos := []*core.Repository{
		mockRepo,
	}
	mockCollabs := []*core.Collaborator{
		{
			UserID: 1, // do not match non-admin
			Admin:  false,
		},
		{
			UserID: 2, // do not match existing owner
			Admin:  true,
		},
		{
			UserID: 3,
			Admin:  true,
		},
	}
	mockUser := &core.User{
		ID: 2,
	}

	checkRepo := func(ctx context.Context, updated *core.Repository) error {
		if updated.UserID != 3 {
			t.Errorf("Expect repository owner id assigned to user id 3")
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().List(gomock.Any(), mockUser.ID).Return(mockRepos, nil).Times(1)
	repos.EXPECT().Update(gomock.Any(), mockRepo).Do(checkRepo).Times(1)

	perms := mock.NewMockPermStore(controller)
	perms.EXPECT().List(gomock.Any(), mockRepo.UID).Return(mockCollabs, nil).Times(1)

	r := New(
		repos,
		perms,
	)

	err := r.Transfer(nocontext, mockUser)
	if err != nil {
		t.Error(err)
	}
}

func TestTransfer_NoOwner(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:     1,
		UserID: 2,
		UID:    "123",
	}
	mockRepos := []*core.Repository{
		mockRepo,
	}
	mockCollabs := []*core.Collaborator{
		{
			UserID: 2, // same user id
			Admin:  true,
		},
	}
	mockUser := &core.User{
		ID: 2,
	}

	checkRepo := func(ctx context.Context, updated *core.Repository) error {
		if updated.UserID != 0 {
			t.Errorf("Expect repository owner id reset to zero value")
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().List(gomock.Any(), mockUser.ID).Return(mockRepos, nil).Times(1)
	repos.EXPECT().Update(gomock.Any(), mockRepo).Do(checkRepo).Times(1)

	perms := mock.NewMockPermStore(controller)
	perms.EXPECT().List(gomock.Any(), mockRepo.UID).Return(mockCollabs, nil).Times(1)

	r := New(
		repos,
		perms,
	)

	err := r.Transfer(nocontext, mockUser)
	if err != nil {
		t.Error(err)
	}
}
