// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package syncer

import (
	"context"
	"database/sql"
	"io/ioutil"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/go-scm/scm"
	"github.com/sirupsen/logrus"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TODO(bradrydzewski) test failure to update user
// TODO(bradrydzewski) test recover from unexpected panic

var noContext = context.Background()

func init() {
	logrus.SetOutput(ioutil.Discard)
	logrus.SetLevel(logrus.TraceLevel)
}

func TestSync(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}

	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*core.Repository{}, nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{
		{
			UID:        "1",
			Slug:       "octocat/hello-world",
			Namespace:  "octocat",
			Name:       "hello-world",
			Private:    false,
			Visibility: core.VisibilityPublic,
		},
	}, nil)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	got, err := s.Sync(context.Background(), user)
	if err != nil {
		t.Error(err)
	}

	want := &core.Batch{
		Insert: []*core.Repository{
			{
				UID:        "1",
				Namespace:  "octocat",
				Name:       "hello-world",
				Slug:       "octocat/hello-world",
				Visibility: core.VisibilityPublic,
				Version:    1,
			},
		},
	}

	ignore := cmpopts.IgnoreFields(core.Repository{},
		"Synced", "Created", "Updated")
	if diff := cmp.Diff(got, want, ignore); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that we are able to recognize when
// a repository has been updated.
func TestSync_Update(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}
	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*core.Repository{
		{UID: "1", Namespace: "octocat", Name: "hello-world"},
		{UID: "2", Namespace: "octocat", Name: "Spoon-Knife", Private: false},
	}, nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{
		{
			UID:       "1",
			Slug:      "octocat/hello-world",
			Namespace: "octocat",
			Name:      "hello-world",
		},
		{
			UID:       "2",
			Slug:      "octocat/Spoon-Knife",
			Namespace: "octocat",
			Name:      "Spoon-Knife",
			Private:   true,
		},
	}, nil)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	got, err := s.Sync(context.Background(), user)
	if err != nil {
		t.Error(err)
	}

	want := &core.Batch{
		Update: []*core.Repository{
			{
				UID:       "2",
				Namespace: "octocat",
				Name:      "Spoon-Knife",
				Slug:      "octocat/Spoon-Knife",
				Private:   true,
			},
		},
	}

	ignore := cmpopts.IgnoreFields(core.Repository{},
		"Synced", "Created", "Updated")
	if diff := cmp.Diff(got, want, ignore); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that we are able to recognize when
// a repository has been renamed.
func TestSync_Rename(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}
	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*core.Repository{
		{ID: 101, UID: "1", Namespace: "octocat", Name: "hello-world"},
		{ID: 102, UID: "2", Namespace: "octocat", Name: "Spoon-Knife"},
	}, nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{
		{
			UID:       "1",
			Slug:      "octocat/hello-world",
			Namespace: "octocat",
			Name:      "hello-world",
		},
		{
			UID:       "2",
			Slug:      "octocat/Spoon-Knife",
			Namespace: "octocat",
			Name:      "Spork-Knife",
		},
	}, nil)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	got, err := s.Sync(context.Background(), user)
	if err != nil {
		t.Error(err)
	}
	want := &core.Batch{
		Update: []*core.Repository{
			{ID: 102, UID: "2", Namespace: "octocat", Name: "Spork-Knife", Slug: "octocat/Spork-Knife"},
		},
	}
	ignore := cmpopts.IgnoreFields(core.Repository{},
		"Synced", "Created", "Updated")
	if diff := cmp.Diff(got, want, ignore); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that we are able to recognize when
// the user permission to the repository have been revoked.
func TestSync_Revoke(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}
	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*core.Repository{
		{UID: "1", Namespace: "octocat", Name: "hello-world"},
	}, nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{}, nil)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	got, err := s.Sync(context.Background(), user)
	if err != nil {
		t.Error(err)
	}
	want := &core.Batch{
		Revoke: []*core.Repository{
			{UID: "1", Namespace: "octocat", Name: "hello-world"},
		},
	}
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that we invoke the batch update even
// if there are no batch updates to make. This is important
// because the batcher resets permissions and forces Drone
// to re-synchronize.
func TestSync_EmptyBatch(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}
	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return(nil, nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	batch, err := s.Sync(context.Background(), user)
	if err != nil {
		t.Error(err)
	}
	if want, got := len(batch.Insert), 0; got != want {
		t.Errorf("Want %d batch inserts, got %d", want, got)
	}
	if want, got := len(batch.Update), 0; got != want {
		t.Errorf("Want %d batch updates, got %d", want, got)
	}
	if want, got := len(batch.Revoke), 0; got != want {
		t.Errorf("Want %d batch revokes, got %d", want, got)
	}
}

// this test verifies that an error returned by the source
// code management system causes the synchronization process to
// exit and is returned to the caller.
func TestSync_RemoteError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}
	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return(nil, scm.ErrNotFound)

	s := New(
		repoService,
		nil,
		userStore,
		nil,
	)
	_, err := s.Sync(context.Background(), user)
	if got, want := err, scm.ErrNotFound; got != want {
		t.Errorf("Want error %s, got %s", want, got)
	}
}

// this test verifies that an error returned by the internal
// repository datastore causes the synchronization process to
// exit and is returned to the caller.
func TestSync_StoreError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}
	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{}, nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)

	s := Synchronizer{
		repoz: repoService,
		users: userStore,
		repos: repoStore,
	}
	_, err := s.Sync(context.Background(), user)
	if got, want := err, sql.ErrNoRows; got != want {
		t.Errorf("Want error %s, got %s", want, got)
	}
}

// this test verifies that an error returned by the batcher
// causes the synchronization process to exit and is returned
// to the caller.
func TestSync_BatchError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}
	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{}, nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(sql.ErrNoRows)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	_, err := s.Sync(context.Background(), user)
	if got, want := err, sql.ErrNoRows; got != want {
		t.Errorf("Want error %s, got %s", want, got)
	}
}

// this test verifies that sub-repositories are skipped. They
// are unsupported by Drone and should not be ignored.
func TestSync_SkipSubrepo(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}

	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*core.Repository{}, nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{
		{
			UID:        "1",
			Slug:       "octocat/hello/world",
			Namespace:  "octocat",
			Name:       "hello-world",
			Private:    false,
			Visibility: core.VisibilityPublic,
		},
	}, nil)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	got, err := s.Sync(context.Background(), user)
	if err != nil {
		t.Error(err)
	}

	want := &core.Batch{}
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

func TestSyncArchive(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	user := &core.User{ID: 1}

	userStore := mock.NewMockUserStore(controller)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)
	userStore.EXPECT().Update(gomock.Any(), user).Return(nil)

	batcher := mock.NewMockBatcher(controller)
	batcher.EXPECT().Batch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	repoStore := mock.NewMockRepositoryStore(controller)
	repoStore.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*core.Repository{}, nil)

	repoService := mock.NewMockRepositoryService(controller)
	repoService.EXPECT().List(gomock.Any(), user).Return([]*core.Repository{
		{
			UID:        "1",
			Slug:       "octocat/hello-world",
			Namespace:  "octocat",
			Name:       "hello-world",
			Private:    false,
			Visibility: core.VisibilityPublic,
			Archived:   true,
		},
	}, nil)

	s := New(
		repoService,
		repoStore,
		userStore,
		batcher,
	)
	got, err := s.Sync(context.Background(), user)
	if err != nil {
		t.Error(err)
	}

	want := &core.Batch{}

	ignore := cmpopts.IgnoreFields(core.Repository{},
		"Synced", "Created", "Updated")
	if diff := cmp.Diff(got, want, ignore); len(diff) != 0 {
		t.Errorf(diff)
	}
}
