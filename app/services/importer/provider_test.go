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

package importer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatchesNamespace(t *testing.T) {
	tests := []struct {
		name             string
		providerType     ProviderType
		repoNamespace    string
		spaceSlug        string
		includeSubgroups bool
		want             bool
	}{
		{
			name:             "GitLab exact match with subgroups",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "mygroup",
			spaceSlug:        "mygroup",
			includeSubgroups: true,
			want:             true,
		},
		{
			name:             "GitLab subgroup match with subgroups enabled",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "mygroup/subgroup",
			spaceSlug:        "mygroup",
			includeSubgroups: true,
			want:             true,
		},
		{
			name:             "GitLab nested subgroup match with subgroups enabled",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "mygroup/subgroup/nested",
			spaceSlug:        "mygroup",
			includeSubgroups: true,
			want:             true,
		},
		{
			name:             "GitLab case insensitive match with subgroups",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "MyGroup/SubGroup",
			spaceSlug:        "mygroup",
			includeSubgroups: true,
			want:             true,
		},
		{
			name:             "GitLab no match different group with subgroups",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "othergroup",
			spaceSlug:        "mygroup",
			includeSubgroups: true,
			want:             false,
		},
		{
			name:             "GitLab no match similar group name prefix",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "mygrouptoo",
			spaceSlug:        "mygroup",
			includeSubgroups: true,
			want:             false, // should NOT match - different group, not a subgroup
		},
		{
			name:             "GitLab subgroup match without subgroups disabled",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "mygroup/subgroup",
			spaceSlug:        "mygroup",
			includeSubgroups: false,
			want:             false, // exact match required
		},
		{
			name:             "GitLab exact match without subgroups",
			providerType:     ProviderTypeGitLab,
			repoNamespace:    "mygroup",
			spaceSlug:        "mygroup",
			includeSubgroups: false,
			want:             true,
		},
		{
			name:             "GitHub exact match",
			providerType:     ProviderTypeGitHub,
			repoNamespace:    "myorg",
			spaceSlug:        "myorg",
			includeSubgroups: false,
			want:             true,
		},
		{
			name:             "GitHub case insensitive match",
			providerType:     ProviderTypeGitHub,
			repoNamespace:    "MyOrg",
			spaceSlug:        "myorg",
			includeSubgroups: false,
			want:             true,
		},
		{
			name:             "GitHub no match for subpath",
			providerType:     ProviderTypeGitHub,
			repoNamespace:    "myorg/team",
			spaceSlug:        "myorg",
			includeSubgroups: false,
			want:             false,
		},
		{
			name:             "GitHub no match different org",
			providerType:     ProviderTypeGitHub,
			repoNamespace:    "otherorg",
			spaceSlug:        "myorg",
			includeSubgroups: false,
			want:             false,
		},
		{
			name:             "Bitbucket exact match",
			providerType:     ProviderTypeBitbucket,
			repoNamespace:    "myworkspace",
			spaceSlug:        "myworkspace",
			includeSubgroups: false,
			want:             true,
		},
		{
			name:             "Bitbucket no match",
			providerType:     ProviderTypeBitbucket,
			repoNamespace:    "otherworkspace",
			spaceSlug:        "myworkspace",
			includeSubgroups: false,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesNamespace(tt.providerType, tt.repoNamespace, tt.spaceSlug, tt.includeSubgroups)
			if got != tt.want {
				t.Errorf("matchesNamespace(%v, %q, %q, %v) = %v, want %v",
					tt.providerType, tt.repoNamespace, tt.spaceSlug, tt.includeSubgroups, got, tt.want)
			}
		})
	}
}

func TestLoadRepositoriesFromProviderSpace_GitLab(t *testing.T) {
	// Override baseTransport to allow loopback for testing
	originalTransport := baseTransport
	baseTransport = http.DefaultTransport
	defer func() { baseTransport = originalTransport }()

	// Mock GitLab API response for group projects endpoint
	gitlabRepos := []map[string]any{
		{
			"id":                  1,
			"name":                "repo1",
			"path":                "repo1",
			"path_with_namespace": "mygroup/repo1",
			"namespace": map[string]any{
				"path": "mygroup",
			},
			"default_branch":   "main",
			"visibility":       "private",
			"ssh_url_to_repo":  "git@gitlab.com:mygroup/repo1.git",
			"http_url_to_repo": "https://gitlab.com/mygroup/repo1.git",
		},
		{
			"id":                  2,
			"name":                "repo2",
			"path":                "repo2",
			"path_with_namespace": "mygroup/subgroup/repo2",
			"namespace": map[string]any{
				"path": "mygroup/subgroup",
			},
			"default_branch":   "main",
			"visibility":       "public",
			"ssh_url_to_repo":  "git@gitlab.com:mygroup/subgroup/repo2.git",
			"http_url_to_repo": "https://gitlab.com/mygroup/subgroup/repo2.git",
		},
		{
			"id":                  3,
			"name":                "other-repo",
			"path":                "other-repo",
			"path_with_namespace": "othergroup/other-repo",
			"namespace": map[string]any{
				"path": "othergroup",
			},
			"default_branch":   "main",
			"visibility":       "private",
			"ssh_url_to_repo":  "git@gitlab.com:othergroup/other-repo.git",
			"http_url_to_repo": "https://gitlab.com/othergroup/other-repo.git",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is using the group projects endpoint with include_subgroups
		expectedPath := "/api/v4/groups/mygroup/projects"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.URL.Query().Get("include_subgroups") != "true" {
			t.Errorf("Expected include_subgroups=true, got %s", r.URL.Query().Get("include_subgroups"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gitlabRepos)
	}))
	defer server.Close()

	provider := Provider{
		Type:     ProviderTypeGitLab,
		Host:     server.URL,
		Username: "testuser",
		Password: "testtoken",
	}

	repos, _, err := LoadRepositoriesFromProviderSpace(context.Background(), provider, "mygroup", true)
	if err != nil {
		t.Fatalf("LoadRepositoriesFromProviderSpace failed: %v", err)
	}

	// Should include repos from mygroup and mygroup/subgroup, but not othergroup
	if len(repos) != 2 {
		t.Errorf("Expected 2 repos, got %d. Repos: %+v", len(repos), repos)
	}

	// Verify repo identifiers
	repoNames := make(map[string]bool)
	for _, repo := range repos {
		repoNames[repo.Identifier] = true
	}
	if !repoNames["repo1"] {
		t.Error("Expected repo1 to be included")
	}
	if !repoNames["repo2"] {
		t.Error("Expected repo2 (from subgroup) to be included")
	}
	if repoNames["other-repo"] {
		t.Error("Expected other-repo to be excluded (different group)")
	}
}

func TestLoadRepositoryFromProvider_Harness(t *testing.T) {
	originalTransport := baseTransport
	baseTransport = http.DefaultTransport
	defer func() { baseTransport = originalTransport }()

	type harnessRepository struct {
		UID           string `json:"uid"`
		Path          string `json:"path"`
		DefaultBranch string `json:"default_branch"`
		GitURL        string `json:"git_url"`
		IsPublic      bool   `json:"is_public"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/sample-repo" {
			t.Errorf("Expected path %s, got %s", "/api/v1/repos/sample-repo", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("accountIdentifier") != "account" || q.Get("orgIdentifier") != "org" ||
			q.Get("projectIdentifier") != "project" {
			t.Errorf("Expected account/org/project query params, got %v", q)
		}
		if r.Header.Get("x-api-key") != "testkey" {
			t.Errorf("Expected x-api-key header %q, got %q", "testkey", r.Header.Get("x-api-key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(harnessRepository{
			UID:           "sample-repo",
			Path:          "account/org/project/sample-repo",
			DefaultBranch: "main",
			GitURL:        "https://harness-code.example.com/account/org/project/sample-repo.git",
			IsPublic:      true,
		})
	}))
	defer server.Close()

	provider := Provider{
		Type:     ProviderTypeHarness,
		Host:     server.URL,
		Password: "testkey",
	}

	repoInfo, _, err := LoadRepositoryFromProvider(context.Background(), provider, "account/org/project/sample-repo")
	if err != nil {
		t.Fatalf("LoadRepositoryFromProvider failed: %v", err)
	}

	if repoInfo.Identifier != "sample-repo" {
		t.Errorf("Expected identifier %q, got %q", "sample-repo", repoInfo.Identifier)
	}
	if repoInfo.CloneURL != "https://harness-code.example.com/account/org/project/sample-repo.git" {
		t.Errorf("Expected clone URL %q, got %q",
			"https://harness-code.example.com/account/org/project/sample-repo.git", repoInfo.CloneURL)
	}
	if !repoInfo.IsPublic {
		t.Error("Expected repo to be public per the provider response")
	}
	if repoInfo.DefaultBranch != "main" {
		t.Errorf("Expected default branch %q, got %q", "main", repoInfo.DefaultBranch)
	}
}

func TestLoadRepositoryFromProvider_Harness_AccountOnlyScope(t *testing.T) {
	originalTransport := baseTransport
	baseTransport = http.DefaultTransport
	defer func() { baseTransport = originalTransport }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("accountIdentifier") != "account" || q.Get("orgIdentifier") != "" || q.Get("projectIdentifier") != "" {
			t.Errorf("Expected only accountIdentifier set, got %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid":     "sample-repo",
			"git_url": "https://harness-code.example.com/account/sample-repo.git",
		})
	}))
	defer server.Close()

	provider := Provider{
		Type: ProviderTypeHarness,
		Host: server.URL,
	}

	if _, _, err := LoadRepositoryFromProvider(context.Background(), provider, "account/sample-repo"); err != nil {
		t.Fatalf("LoadRepositoryFromProvider failed: %v", err)
	}
}

func TestLoadRepositoryFromProvider_Harness_NotFound(t *testing.T) {
	originalTransport := baseTransport
	baseTransport = http.DefaultTransport
	defer func() { baseTransport = originalTransport }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer server.Close()

	provider := Provider{
		Type: ProviderTypeHarness,
		Host: server.URL,
	}

	_, _, err := LoadRepositoryFromProvider(context.Background(), provider, "account/org/project/missing-repo")
	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

func TestLoadRepositoryFromProvider_Harness_MissingHost(t *testing.T) {
	provider := Provider{
		Type: ProviderTypeHarness,
	}

	if _, _, err := LoadRepositoryFromProvider(context.Background(), provider, "account/sample-repo"); err == nil {
		t.Fatal("Expected error when provider host is missing, got nil")
	}
}

func TestLoadRepositoryFromProvider_Harness_InvalidSlug(t *testing.T) {
	provider := Provider{
		Type: ProviderTypeHarness,
		Host: "https://harness-code.example.com",
	}

	if _, _, err := LoadRepositoryFromProvider(context.Background(), provider, "sample-repo"); err == nil {
		t.Fatal("Expected error for slug missing account scope, got nil")
	}
}
