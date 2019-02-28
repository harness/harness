// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/go-scm/scm"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

var noContext = context.Background()

func TestFind(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockFile := &core.File{
		Data: []byte("hello world"),
		Hash: []byte(""),
	}

	mockContents := mock.NewMockFileService(controller)
	mockContents.EXPECT().Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml").Return(mockFile, nil)

	service := Contents(mockContents).(*service)

	want := &core.File{
		Data: []byte("hello world"),
		Hash: []byte(""),
	}

	got, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml")
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}

	if len(service.cache.Keys()) == 0 {
		t.Errorf("Expect item added to cache")
	}
}

func TestFindError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockContents := mock.NewMockFileService(controller)
	mockContents.EXPECT().Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml").Return(nil, scm.ErrNotFound)

	service := Contents(mockContents).(*service)

	_, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml")
	if err != scm.ErrNotFound {
		t.Errorf("Expect not found error")
	}
}

func TestFindCache(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockFile := &core.File{
		Data: []byte("hello world"),
		Hash: []byte(""),
	}

	key := fmt.Sprintf(contentKey, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", ".drone.yml")
	service := Contents(nil).(*service)
	service.cache.Add(key, mockFile)

	want := &core.File{
		Data: []byte("hello world"),
		Hash: []byte(""),
	}

	got, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml")
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}
