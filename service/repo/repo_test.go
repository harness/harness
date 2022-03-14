// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package repo

import (
	"context"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/drone/mock/mockscm"
	"github.com/drone/go-scm/scm"
	"github.com/google/go-cmp/cmp"

	"github.com/golang/mock/gomock"
)

var noContext = context.Background()

func TestFind(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockRepo := &scm.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().Find(gomock.Any(), "octocat/hello-world").Return(mockRepo, nil, nil)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)

	want := &core.Repository{
		Namespace:  "octocat",
		Name:       "hello-world",
		Slug:       "octocat/hello-world",
		Visibility: "public",
	}

	got, err := service.Find(noContext, mockUser, "octocat/hello-world")
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestFind_Err(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().Find(gomock.Any(), "octocat/hello-world").Return(nil, nil, scm.ErrNotFound)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)
	_, err := service.Find(noContext, mockUser, "octocat/hello-world")
	if err != scm.ErrNotFound {
		t.Errorf("Expect not found error, got %v", err)
	}
}

func TestFind_RefreshErr(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(scm.ErrNotAuthorized)

	service := New(nil, mockRenewer, "", false)
	_, err := service.Find(noContext, mockUser, "octocat/hello-world")
	if err == nil {
		t.Errorf("Expect error refreshing token")
	}
}

func TestFindPerm(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockPerm := &scm.Perm{
		Pull:  true,
		Push:  true,
		Admin: true,
	}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().FindPerms(gomock.Any(), "octocat/hello-world").Return(mockPerm, nil, nil)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)

	want := &core.Perm{
		Read:  true,
		Write: true,
		Admin: true,
	}

	got, err := service.FindPerm(noContext, mockUser, "octocat/hello-world")
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestFindPerm_Err(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().FindPerms(gomock.Any(), "octocat/hello-world").Return(nil, nil, scm.ErrNotFound)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)
	_, err := service.FindPerm(noContext, mockUser, "octocat/hello-world")
	if err != scm.ErrNotFound {
		t.Errorf("Expect not found error, got %v", err)
	}
}

func TestFindPerm_RefreshErr(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(scm.ErrNotAuthorized)

	service := New(nil, mockRenewer, "", false)
	_, err := service.FindPerm(noContext, mockUser, "octocat/hello-world")
	if err == nil {
		t.Errorf("Expect error refreshing token")
	}
}

func TestList(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockRepos := []*scm.Repository{
		{
			Namespace: "octocat",
			Name:      "hello-world",
		},
	}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockRepos, &scm.Response{}, nil)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	want := []*core.Repository{
		{
			Namespace:  "octocat",
			Name:       "hello-world",
			Slug:       "octocat/hello-world",
			Visibility: "public",
		},
	}

	service := New(client, mockRenewer, "", false)
	got, err := service.List(noContext, mockUser)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestList_Err(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, &scm.Response{}, scm.ErrNotAuthorized)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)
	_, err := service.List(noContext, mockUser)
	if err != scm.ErrNotAuthorized {
		t.Errorf("Want not authorized error, got %v", err)
	}
}

func TestList_RefreshErr(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(scm.ErrNotAuthorized)

	service := New(nil, mockRenewer, "", false)
	_, err := service.List(noContext, mockUser)
	if err == nil {
		t.Errorf("Expect error refreshing token")
	}
}

func TestListWithNilRepo(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockRepos := []*scm.Repository{
		{
			Namespace: "octocat",
			Name:      "hello-world",
		},
		nil,
	}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockRepos, &scm.Response{}, nil)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	want := []*core.Repository{
		{
			Namespace:  "octocat",
			Name:       "hello-world",
			Slug:       "octocat/hello-world",
			Visibility: "public",
		},
	}

	service := New(client, mockRenewer, "", false)
	got, err := service.List(noContext, mockUser)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}
