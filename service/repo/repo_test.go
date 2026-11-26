// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package repo

import (
	"context"
	"reflect"
	"testing"
	"time"

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
		t.Errorf("Diff: %s", diff)
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
		t.Errorf("Diff: %s", diff)
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
		t.Errorf("Diff: %s", diff)
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

// TestList_ConcurrentPages proves that when the provider reports the
// total page count up front (meta.Page.Last), List fetches the
// remaining pages concurrently and still returns every repository,
// regardless of which goroutine finishes first.
func TestList_ConcurrentPages(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	pages := map[int][]*scm.Repository{
		0: {{Namespace: "octocat", Name: "repo-1"}}, // the first request has no page number set
		2: {{Namespace: "octocat", Name: "repo-2"}},
		3: {{Namespace: "octocat", Name: "repo-3"}},
	}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, opts scm.ListOptions) ([]*scm.Repository, *scm.Response, error) {
			repos, ok := pages[opts.Page]
			if !ok {
				t.Errorf("unexpected page requested: %d", opts.Page)
			}
			res := &scm.Response{}
			if opts.Page == 0 {
				res.Page.Last = 3
			}
			return repos, res, nil
		}).
		Times(3)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)
	got, err := service.List(noContext, mockUser)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 repos, got %d: %+v", len(got), got)
	}
	names := map[string]bool{}
	for _, r := range got {
		names[r.Name] = true
	}
	for _, want := range []string{"repo-1", "repo-2", "repo-3"} {
		if !names[want] {
			t.Errorf("want repo %q in result, got %+v", want, got)
		}
	}
}

// TestList_ConcurrentPagesOrder proves that the result keeps page
// order even when a later page completes first: page 2 is held until
// page 3 has returned, and the result must still be page 1, 2, 3.
func TestList_ConcurrentPagesOrder(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	page3Done := make(chan struct{})

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, opts scm.ListOptions) ([]*scm.Repository, *scm.Response, error) {
			switch opts.Page {
			case 0:
				res := &scm.Response{}
				res.Page.Last = 3
				return []*scm.Repository{{Namespace: "octocat", Name: "repo-1"}}, res, nil
			case 2:
				// do not complete until page 3 has completed
				select {
				case <-page3Done:
				case <-time.After(5 * time.Second):
					t.Error("page 3 never completed; pages are not fetched concurrently")
				}
				return []*scm.Repository{{Namespace: "octocat", Name: "repo-2"}}, &scm.Response{}, nil
			case 3:
				defer close(page3Done)
				return []*scm.Repository{{Namespace: "octocat", Name: "repo-3"}}, &scm.Response{}, nil
			}
			t.Errorf("unexpected page requested: %d", opts.Page)
			return nil, &scm.Response{}, nil
		}).
		Times(3)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)
	got, err := service.List(noContext, mockUser)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range got {
		names = append(names, r.Name)
	}
	if want := []string{"repo-1", "repo-2", "repo-3"}; !reflect.DeepEqual(names, want) {
		t.Errorf("want repos in page order %v, got %v", want, names)
	}
}

// TestList_ConcurrentPagesErr proves that an error fetching any page
// fails the whole call.
func TestList_ConcurrentPagesErr(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	mockRepoService.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, opts scm.ListOptions) ([]*scm.Repository, *scm.Response, error) {
			if opts.Page == 0 {
				res := &scm.Response{}
				res.Page.Last = 2
				return []*scm.Repository{{Namespace: "octocat", Name: "repo-1"}}, res, nil
			}
			return nil, &scm.Response{}, scm.ErrNotAuthorized
		}).
		Times(2)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	service := New(client, mockRenewer, "", false)
	_, err := service.List(noContext, mockUser)
	if err != scm.ErrNotAuthorized {
		t.Errorf("want not authorized error, got %v", err)
	}
}

// TestList_SequentialFallback proves that when the provider only
// exposes an opaque next-page cursor (no Page.Last), List still walks
// every page sequentially, in order.
func TestList_SequentialFallback(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRepoService := mockscm.NewMockRepositoryService(controller)
	first := mockRepoService.EXPECT().
		List(gomock.Any(), scm.ListOptions{Size: 100}).
		Return([]*scm.Repository{{Namespace: "octocat", Name: "repo-1"}},
			&scm.Response{Page: scm.Page{NextURL: "https://example.com/repos?cursor=abc"}}, nil)
	mockRepoService.EXPECT().
		List(gomock.Any(), scm.ListOptions{Size: 100, URL: "https://example.com/repos?cursor=abc"}).
		Return([]*scm.Repository{{Namespace: "octocat", Name: "repo-2"}}, &scm.Response{}, nil).
		After(first)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Repositories = mockRepoService

	want := []*core.Repository{
		{Namespace: "octocat", Name: "repo-1", Slug: "octocat/repo-1", Visibility: "public"},
		{Namespace: "octocat", Name: "repo-2", Slug: "octocat/repo-2", Visibility: "public"},
	}

	service := New(client, mockRenewer, "", false)
	got, err := service.List(noContext, mockUser)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff: %s", diff)
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
		t.Errorf("Diff: %s", diff)
	}
}
