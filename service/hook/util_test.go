// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package hook

import (
	"context"
	"io"
	"testing"

	"github.com/drone/drone/mock/mockscm"
	"github.com/drone/go-scm/scm"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestFindHook(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	hooks := []*scm.Hook{
		{Target: "http://192.168.0.%31/hook"},
		{Target: "https://drone.company.com/hook"},
	}
	remote := mockscm.NewMockRepositoryService(controller)
	remote.EXPECT().ListHooks(gomock.Any(), "octocat/hello-world", gomock.Any()).Return(hooks, nil, nil)

	client := new(scm.Client)
	client.Repositories = remote

	hook, err := findHook(context.Background(), client, "octocat/hello-world", "drone.company.com")
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(hook, hooks[1]); len(diff) > 0 {
		t.Errorf(diff)
	}
}

func TestFindHook_ListError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	remote := mockscm.NewMockRepositoryService(controller)
	remote.EXPECT().ListHooks(gomock.Any(), "octocat/hello-world", gomock.Any()).Return(nil, nil, io.EOF)

	client := new(scm.Client)
	client.Repositories = remote

	_, err := findHook(context.Background(), client, "octocat/hello-world", "core.company.com")
	if err == nil {
		t.Errorf("Want hook request failure to return error")
	}
}

func TestReplaceHook_CreateHook(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	hooks := []*scm.Hook{}
	hookInput := &scm.HookInput{
		Target: "https://drone.company.com/hook",
	}

	remote := mockscm.NewMockRepositoryService(controller)
	remote.EXPECT().ListHooks(gomock.Any(), "octocat/hello-world", gomock.Any()).Return(hooks, nil, nil)
	remote.EXPECT().CreateHook(gomock.Any(), "octocat/hello-world", hookInput).Return(nil, nil, nil)

	client := new(scm.Client)
	client.Repositories = remote

	err := replaceHook(context.Background(), client, "octocat/hello-world", hookInput)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceHook_UpdateHook(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	hooks := []*scm.Hook{
		{
			ID:     "1",
			Target: "https://drone.company.com/hook",
		},
	}
	hookInput := &scm.HookInput{
		Target: "https://drone.company.com/hook",
	}

	remote := mockscm.NewMockRepositoryService(controller)
	remote.EXPECT().ListHooks(gomock.Any(), "octocat/hello-world", gomock.Any()).Return(hooks, nil, nil)
	remote.EXPECT().DeleteHook(gomock.Any(), "octocat/hello-world", "1").Return(nil, nil)
	remote.EXPECT().CreateHook(gomock.Any(), "octocat/hello-world", hookInput).Return(nil, nil, nil)

	client := new(scm.Client)
	client.Repositories = remote

	err := replaceHook(context.Background(), client, "octocat/hello-world", hookInput)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceHook_DeleteError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	hooks := []*scm.Hook{
		{
			ID:     "1",
			Target: "https://drone.company.com/hook",
		},
	}
	hookInput := &scm.HookInput{
		Target: "https://drone.company.com/hook",
	}

	remote := mockscm.NewMockRepositoryService(controller)
	remote.EXPECT().ListHooks(gomock.Any(), "octocat/hello-world", gomock.Any()).Return(hooks, nil, nil)
	remote.EXPECT().DeleteHook(gomock.Any(), "octocat/hello-world", "1").Return(nil, io.EOF)

	client := new(scm.Client)
	client.Repositories = remote

	err := replaceHook(context.Background(), client, "octocat/hello-world", hookInput)
	if err == nil {
		t.Errorf("Expect error if hook deletion fails")
	}
}

func TestReplaceHook_DeleteFindError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	hookInput := &scm.HookInput{
		Target: "https://drone.company.com/hook",
	}

	remote := mockscm.NewMockRepositoryService(controller)
	remote.EXPECT().ListHooks(gomock.Any(), "octocat/hello-world", gomock.Any()).Return(nil, nil, io.EOF)

	client := new(scm.Client)
	client.Repositories = remote

	err := replaceHook(context.Background(), client, "octocat/hello-world", hookInput)
	if err == nil {
		t.Errorf("Expect error if hook deletion fails")
	}
}

func TestReplaceHook_CreateError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	hooks := []*scm.Hook{}
	hookInput := &scm.HookInput{
		Target: "https://drone.company.com/hook",
	}

	remote := mockscm.NewMockRepositoryService(controller)
	remote.EXPECT().ListHooks(gomock.Any(), "octocat/hello-world", gomock.Any()).Return(hooks, nil, nil)
	remote.EXPECT().CreateHook(gomock.Any(), "octocat/hello-world", hookInput).Return(nil, nil, io.EOF)

	client := new(scm.Client)
	client.Repositories = remote

	err := replaceHook(context.Background(), client, "octocat/hello-world", hookInput)
	if err == nil {
		t.Errorf("Expect error if hook creation fails")
	}
}
