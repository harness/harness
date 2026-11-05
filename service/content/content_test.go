// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package contents

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/drone/mock/mockscm"
	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/gitea"
	"github.com/drone/go-scm/scm/driver/gitee"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/driver/gogs"
	"github.com/drone/go-scm/scm/driver/stash"
	"github.com/google/go-cmp/cmp"

	"github.com/golang/mock/gomock"
)

var noContext = context.Background()

// Exercise the actual SCM drivers: a mock ContentService cannot detect URL
// fragments, query injection, invalid percent escapes, or double encoding.
func TestFindPreservesFilenameInSCMRequest(t *testing.T) {
	for _, driver := range []struct {
		name   string
		new    func(string) (*scm.Client, error)
		prefix string
		query  string
		json   bool
	}{
		{"github", github.New, "/repos/owner/repo/contents/", "ref=commit", true},
		{"gitee", gitee.New, "/repos/owner/repo/contents/", "ref=commit", true},
		{"gitlab", gitlab.New, "/api/v4/projects/owner/repo/repository/files/", "ref=commit", true},
		{"gitea", gitea.New, "/api/v1/repos/owner/repo/raw/commit/", "", false},
		{"gogs", gogs.New, "/api/v1/repos/owner/repo/raw/commit/", "", false},
		{"stash", stash.New, "/rest/api/1.0/projects/owner/repos/repo/raw/", "at=commit", false},
		{"bitbucket", bitbucket.New, "/2.0/repositories/owner/repo/src/commit/", "", false},
	} {
		for _, filename := range []string{".ci/pipeline.libsonnet", "dir/a#b%c?.libsonnet", "q?ref=other#ignored", "%2e%2e/%2fsecret", "dir/space and 日本語.libsonnet"} {
			t.Run(driver.name+"/"+filename, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.URL.Path != driver.prefix+filename {
						t.Errorf("filename changed in transport: want %q, got %q", driver.prefix+filename, r.URL.Path)
					}
					wantQuery := driver.query
					if driver.name == "bitbucket" && calls == 2 {
						wantQuery = "format=meta"
					}
					want, _ := url.ParseQuery(wantQuery)
					if diff := cmp.Diff(want, r.URL.Query()); diff != "" {
						t.Errorf("filename changed query parameters: %s", diff)
					}
					if strings.Contains(filename, "%2e") && !strings.Contains(r.URL.EscapedPath(), "%252e") {
						t.Error("literal encoded traversal was not escaped as filename data")
					}
					w.Header().Set("Content-Type", "application/json")
					switch {
					case driver.json:
						fmt.Fprint(w, `{"content":"aGVsbG8=","encoding":"base64"}`)
					case driver.name == "bitbucket" && calls == 2:
						fmt.Fprint(w, `{"commit":{"hash":"commit"}}`)
					default:
						fmt.Fprint(w, "hello")
					}
				}))
				defer server.Close()
				client, err := driver.new(server.URL)
				if err != nil {
					t.Fatal(err)
				}
				controller := gomock.NewController(t)
				renewer := mock.NewMockRenewer(controller)
				renewer.EXPECT().Renew(gomock.Any(), gomock.Any(), false).Return(nil)
				s := New(client, renewer).(*service)
				s.attempts, s.wait = 1, 0
				file, err := s.Find(noContext, &core.User{}, "owner/repo", "commit", "refs/heads/main", filename)
				if err != nil {
					t.Fatal(err)
				}
				if string(file.Data) != "hello" || calls == 0 {
					t.Fatalf("unexpected SCM response: %q (%d requests)", file.Data, calls)
				}
			})
		}
	}
}

func TestFind(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockFile := &scm.Content{
		Path: ".drone.yml",
		Data: []byte("hello world"),
	}

	mockContents := mockscm.NewMockContentService(controller)
	mockContents.EXPECT().Find(gomock.Any(), "octocat/hello-world", ".drone.yml", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa").Return(mockFile, nil, nil)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Contents = mockContents

	want := &core.File{
		Data: []byte("hello world"),
		Hash: []byte(""),
	}

	service := New(client, mockRenewer)
	got, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml")
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff: %s", diff)
	}
}

func TestFind_Error(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockContents := mockscm.NewMockContentService(controller)
	mockContents.EXPECT().Find(gomock.Any(), "octocat/hello-world", ".drone.yml", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa").Return(nil, nil, scm.ErrNotFound)

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false)

	client := new(scm.Client)
	client.Contents = mockContents

	s := New(client, mockRenewer)
	s.(*service).attempts = 1
	s.(*service).wait = 0
	_, err := s.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml")
	if err != scm.ErrNotFound {
		t.Errorf("Expect not found error, got %s", err)
	}
}

func TestFind_RenewalError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(scm.ErrNotAuthorized)

	client := new(scm.Client)

	service := New(client, mockRenewer)
	_, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master", ".drone.yml")
	if err == nil {
		t.Errorf("Expect error refreshing token")
	}
}
