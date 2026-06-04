// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ghclient "github.com/harness/gitness/app/services/linkedpr/github"

	gh "github.com/drone/go-scm/scm/driver/github"
)

const canonicalRepoJSON = `{
  "id": 280125018,
  "node_id": "MDEwOlJlcG9zaXRvcnkyODAxMjUwMTg=",
  "name": "HLC_PCP",
  "full_name": "anurag-harness/HLC_PCP",
  "default_branch": "master",
  "private": true,
  "html_url": "https://github.com/anurag-harness/HLC_PCP"
}`

func startGHRepoServer(t *testing.T, status int, body string, authCheck func(string)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/anurag-harness/HLC_PCP", func(w http.ResponseWriter, r *http.Request) {
		if authCheck != nil {
			authCheck(r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	return httptest.NewServer(mux)
}

func TestGetRepository_HappyPath(t *testing.T) {
	srv := startGHRepoServer(t, 200, canonicalRepoJSON, nil)
	defer srv.Close()

	scmClient, err := gh.New(srv.URL)
	if err != nil {
		t.Fatalf("gh.New: %v", err)
	}
	c := ghclient.NewClient(scmClient)

	repo, err := c.GetRepository(context.Background(), "anurag-harness/HLC_PCP")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if repo.NodeID != "MDEwOlJlcG9zaXRvcnkyODAxMjUwMTg=" {
		t.Errorf("NodeID: got %q", repo.NodeID)
	}
}

func TestGetRepository_NotFound(t *testing.T) {
	srv := startGHRepoServer(t, 404, `{"message":"Not Found"}`, nil)
	defer srv.Close()

	scmClient, _ := gh.New(srv.URL)
	c := ghclient.NewClient(scmClient)

	if _, err := c.GetRepository(context.Background(), "anurag-harness/HLC_PCP"); err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestGetRepository_AuthHeaderSent(t *testing.T) {
	var got string
	srv := startGHRepoServer(t, 200, canonicalRepoJSON, func(h string) { got = h })
	defer srv.Close()

	c, err := ghclient.NewClientFromConnector(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClientFromConnector: %v", err)
	}
	if _, err := c.GetRepository(context.Background(), "anurag-harness/HLC_PCP"); err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if !strings.Contains(got, "Bearer test-token") {
		t.Errorf("Authorization header: got %q, want Bearer test-token", got)
	}
}

func TestNewClientFromConnector_DefaultURL(t *testing.T) {
	c, err := ghclient.NewClientFromConnector("", "")
	if err != nil {
		t.Fatalf("NewClientFromConnector: %v", err)
	}
	if c == nil {
		t.Fatal("client nil")
	}
}

func TestNewClientFromConnector_GHEURL(t *testing.T) {
	c, err := ghclient.NewClientFromConnector("https://github.acme.com/api/v3", "")
	if err != nil {
		t.Fatalf("NewClientFromConnector: %v", err)
	}
	if c == nil {
		t.Fatal("client nil")
	}
}

func TestNewClientFromConnector_InvalidURL(t *testing.T) {
	if _, err := ghclient.NewClientFromConnector("://not a url", ""); err == nil {
		t.Errorf("expected error for invalid URL")
	}
}
