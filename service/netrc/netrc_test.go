// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package netrc

import (
	"context"
	"net/url"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/go-scm/scm"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

var noContext = context.Background()

func TestNetrc(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{Private: true, HTTPURL: "https://github.com/octocat/hello-world"}
	mockUser := &core.User{
		Token:   "755bb80e5b",
		Refresh: "e08f3fa43e",
	}
	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, true)

	mockClient := &scm.Client{Driver: scm.DriverGithub}

	s := New(mockClient, mockRenewer, false, "", "")
	got, err := s.Create(noContext, mockUser, mockRepo)
	if err != nil {
		t.Error(err)
	}

	want := &core.Netrc{
		Machine:  "github.com",
		Login:    "755bb80e5b",
		Password: "x-oauth-basic",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestNetrc_Gitlab(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{Private: true, HTTPURL: "https://gitlab.com/octocat/hello-world"}
	mockUser := &core.User{
		Token:   "755bb80e5b",
		Refresh: "e08f3fa43e",
	}
	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, true)

	s := Service{
		renewer: mockRenewer,
		client:  &scm.Client{Driver: scm.DriverGitlab},
	}
	got, err := s.Create(noContext, mockUser, mockRepo)
	if err != nil {
		t.Error(err)
	}

	want := &core.Netrc{
		Machine:  "gitlab.com",
		Login:    "oauth2",
		Password: "755bb80e5b",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestNetrc_Gogs(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{Private: true, HTTPURL: "https://try.gogs.io/octocat/hello-world"}
	mockUser := &core.User{
		Token:   "755bb80e5b",
		Refresh: "e08f3fa43e",
	}
	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, true)

	s := Service{
		renewer: mockRenewer,
		client:  &scm.Client{Driver: scm.DriverGogs},
	}
	got, err := s.Create(noContext, mockUser, mockRepo)
	if err != nil {
		t.Error(err)
	}

	want := &core.Netrc{
		Machine:  "try.gogs.io",
		Login:    "755bb80e5b",
		Password: "x-oauth-basic",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestNetrc_Bitbucket(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{Private: true, HTTPURL: "https://bitbucket.org/octocat/hello-world"}
	mockUser := &core.User{
		Token:   "755bb80e5b",
		Refresh: "e08f3fa43e",
	}
	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, true)

	s := Service{
		renewer: mockRenewer,
		client:  &scm.Client{Driver: scm.DriverBitbucket},
	}
	got, err := s.Create(noContext, mockUser, mockRepo)
	if err != nil {
		t.Error(err)
	}

	want := &core.Netrc{
		Machine:  "bitbucket.org",
		Login:    "x-token-auth",
		Password: "755bb80e5b",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestNetrc_Gitee(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{Private: true, HTTPURL: "https://gitee.com/kit101/drone-yml-test"}
	mockUser := &core.User{
		Token:   "755bb80e5b",
		Refresh: "e08f3fa43e",
	}
	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, true)

	s := Service{
		renewer: mockRenewer,
		client:  &scm.Client{Driver: scm.DriverGitee},
	}
	got, err := s.Create(noContext, mockUser, mockRepo)
	if err != nil {
		t.Error(err)
	}

	want := &core.Netrc{
		Machine:  "gitee.com",
		Login:    "oauth2",
		Password: "755bb80e5b",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestNetrc_Nil(t *testing.T) {
	s := Service{
		private: false,
	}
	netrc, _ := s.Create(noContext, &core.User{}, &core.Repository{Private: false})
	if netrc != nil {
		t.Errorf("Expect nil netrc file when public repository")
	}
}

func TestNetrc_MalformedURL(t *testing.T) {
	s := Service{
		private: true,
	}
	_, err := s.Create(noContext, &core.User{}, &core.Repository{HTTPURL: ":::"})
	if _, ok := err.(*url.Error); !ok {
		t.Errorf("Expect error when URL malformed")
	}
}

func TestNetrc_StaticLogin(t *testing.T) {
	s := Service{
		private:  true,
		username: "octocat",
		password: "password",
	}
	got, err := s.Create(noContext, &core.User{}, &core.Repository{HTTPURL: "https://github.com/octocat/hello-world"})
	if err != nil {
		t.Error(err)
	}

	want := &core.Netrc{
		Machine:  "github.com",
		Login:    "octocat",
		Password: "password",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestNetrc_RenewErr(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{Private: true, HTTPURL: "https://github.com/octocat/hello-world"}
	mockUser := &core.User{
		Token:   "755bb80e5b",
		Refresh: "e08f3fa43e",
	}
	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, true).Return(scm.ErrNotAuthorized)

	s := Service{
		renewer: mockRenewer,
		client:  &scm.Client{Driver: scm.DriverGithub},
	}
	_, err := s.Create(noContext, mockUser, mockRepo)
	if err != scm.ErrNotAuthorized {
		t.Errorf("Want not authorized error, got %v", err)
	}
}
