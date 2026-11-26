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
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/go-scm/scm"
	"github.com/stretchr/testify/require"
)

// svaRecordedRequest holds what the test server saw of a request.
type svaRecordedRequest struct {
	path string
	auth string
	key  string
}

// svaProviderServer is a test server answering every request with the same body.
type svaProviderServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests []svaRecordedRequest
}

func svaNewProviderServer(t *testing.T, status int, body any) *svaProviderServer {
	t.Helper()

	srv := &svaProviderServer{}
	srv.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.mu.Lock()
		srv.requests = append(srv.requests, svaRecordedRequest{
			path: r.URL.Path,
			auth: r.Header.Get("Authorization"),
			key:  r.Header.Get("x-api-key"),
		})
		srv.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)

	return srv
}

func (s *svaProviderServer) recorded() []svaRecordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]svaRecordedRequest(nil), s.requests...)
}

// svaProbe sends a GET request through the HTTP client of the SCM client.
func svaProbe(t *testing.T, client *scm.Client, target string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	require.NoError(t, err)

	resp, err := client.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
}

func TestSvaScmClientAuthPerProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		provider     ProviderType
		slug         string
		username     string
		wantAuth     string
		wantAuthPref string
		wantKey      string
	}{
		{name: "github", provider: ProviderTypeGitHub, slug: "owner/repo", wantAuth: "Bearer secret"},
		{name: "gitlab", provider: ProviderTypeGitLab, slug: "owner/repo", wantAuth: "Bearer secret"},
		{name: "stash", provider: ProviderTypeStash, slug: "owner/repo", wantAuth: "Bearer secret"},
		{name: "gogs", provider: ProviderTypeGogs, slug: "owner/repo", wantAuth: "token secret"},
		{name: "gitea", provider: ProviderTypeGitea, slug: "owner/repo", wantAuth: "token secret"},
		{
			name: "bitbucket", provider: ProviderTypeBitbucket, slug: "owner/repo", username: "user",
			wantAuthPref: "Basic ",
		},
		{
			name: "azure", provider: ProviderTypeAzure, slug: "org/project/repo", username: "user",
			wantAuthPref: "Basic ",
		},
		{name: "harness", provider: ProviderTypeHarness, slug: "account/org/project/repo", wantKey: "secret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			srv := svaNewProviderServer(t, http.StatusOK, map[string]any{"name": "repo", "login": "octo"})

			imp := newLoopbackImporter()
			provider := Provider{Type: test.provider, Host: srv.URL, Username: test.username, Password: "secret"}

			client, err := imp.getScmClientWithTransport(provider, test.slug, false)
			require.NoError(t, err)
			require.NotNil(t, client)

			svaProbe(t, client, srv.URL+"/probe")

			reqs := srv.recorded()
			require.Len(t, reqs, 1)
			if test.wantAuth != "" {
				require.Equal(t, test.wantAuth, reqs[0].auth)
			}
			if test.wantAuthPref != "" {
				require.True(t, strings.HasPrefix(reqs[0].auth, test.wantAuthPref), reqs[0].auth)
			}
			require.Equal(t, test.wantKey, reqs[0].key)
		})
	}
}

func TestSvaScmClientWithoutCredentialsIsAnonymous(t *testing.T) {
	t.Parallel()

	srv := svaNewProviderServer(t, http.StatusOK, map[string]any{})

	for _, providerType := range []ProviderType{
		ProviderTypeGitHub, ProviderTypeGitLab, ProviderTypeBitbucket, ProviderTypeStash,
		ProviderTypeGitea, ProviderTypeGogs, ProviderTypeHarness,
	} {
		client, err := newLoopbackImporter().getScmClientWithTransport(
			Provider{Type: providerType, Host: srv.URL}, "account/repo", false)
		require.NoError(t, err, providerType)

		svaProbe(t, client, srv.URL+"/probe")
	}

	for _, req := range srv.recorded() {
		require.Empty(t, req.auth)
		require.Empty(t, req.key)
	}
}

func TestSvaScmClientDefaultHosts(t *testing.T) {
	t.Parallel()

	imp := newLoopbackImporter()

	for _, providerType := range []ProviderType{
		ProviderTypeGitHub, ProviderTypeGitLab, ProviderTypeBitbucket, ProviderTypeStash,
	} {
		client, err := imp.getScmClientWithTransport(Provider{Type: providerType}, "owner/repo", false)
		require.NoError(t, err, providerType)
		require.NotNil(t, client.BaseURL, providerType)
	}

	client, err := imp.getScmClientWithTransport(Provider{Type: ProviderTypeAzure}, "org/project/repo", false)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestSvaScmClientTraceHTTP(t *testing.T) {
	t.Parallel()

	imp := newLoopbackImporter()
	imp.traceHTTP = true

	client, err := imp.getScmClientWithTransport(Provider{Type: ProviderTypeGitHub}, "owner/repo", false)
	require.NoError(t, err)
	require.NotNil(t, client.Client.Transport)
}

func TestSvaScmClientErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider Provider
		slug     string
		authReq  bool
		wantUser bool
	}{
		{name: "missing credentials", provider: Provider{Type: ProviderTypeGitHub}, authReq: true, wantUser: true},
		{
			name: "missing password", provider: Provider{Type: ProviderTypeGitHub, Username: "u"},
			authReq: true, wantUser: true,
		},
		{name: "empty type", provider: Provider{}},
		{name: "unknown type", provider: Provider{Type: "svn"}},
		{name: "gitea without host", provider: Provider{Type: ProviderTypeGitea}},
		{name: "gogs without host", provider: Provider{Type: ProviderTypeGogs}},
		{name: "harness without host", provider: Provider{Type: ProviderTypeHarness}, slug: "account/repo"},
		{name: "harness invalid slug", provider: Provider{Type: ProviderTypeHarness, Host: "http://127.0.0.1:1"}, slug: "x"},
		{name: "azure invalid slug", provider: Provider{Type: ProviderTypeAzure}, slug: "only-one"},
		{name: "azure slug with too many parts", provider: Provider{Type: ProviderTypeAzure}, slug: "a/b/c/d"},
		{
			name:     "invalid host scheme",
			provider: Provider{Type: ProviderTypeGitHub, Host: "ftp://example.com"},
			wantUser: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client, err := newLoopbackImporter().getScmClientWithTransport(test.provider, test.slug, test.authReq)
			require.Error(t, err)
			require.Nil(t, client)
			if test.wantUser {
				var userErr *usererror.Error
				require.ErrorAs(t, err, &userErr)
			}
		})
	}
}

func TestSvaLoadRepositoryFromProviderLooksUpTheUser(t *testing.T) {
	t.Parallel()

	srv := svaNewProviderServer(t, http.StatusOK, map[string]any{
		"login":          "octo",
		"name":           "repo",
		"full_name":      "owner/repo",
		"private":        true,
		"default_branch": "trunk",
		"clone_url":      "https://example.com/owner/repo.git",
		"owner":          map[string]any{"login": "owner"},
	})

	provider := Provider{Type: ProviderTypeGitHub, Host: srv.URL, Password: "secret"}

	info, got, err := newLoopbackImporter().LoadRepositoryFromProvider(context.Background(), provider, "owner/repo")
	require.NoError(t, err)
	require.Equal(t, "octo", got.Username, "a missing username is resolved from the token")
	require.Equal(t, "repo", info.Identifier)
	require.Equal(t, "owner", info.Space)
	require.False(t, info.IsPublic)
	require.Equal(t, "trunk", info.DefaultBranch)
	require.Equal(t, "https://example.com/owner/repo.git", info.CloneURL)

	paths := make([]string, 0, 2)
	for _, req := range srv.recorded() {
		paths = append(paths, req.path)
	}
	require.Contains(t, paths, "/user")
	require.Contains(t, paths, "/repos/owner/repo")
}

func TestSvaLoadRepositoryFromProviderAzure(t *testing.T) {
	t.Parallel()

	srv := svaNewProviderServer(t, http.StatusOK, map[string]any{
		"name": "repo", "defaultBranch": "refs/heads/main", "remoteUrl": "https://dev.azure.com/org/project/_git/repo",
	})

	provider := Provider{Type: ProviderTypeAzure, Host: srv.URL, Username: "user", Password: "secret"}

	_, _, err := newLoopbackImporter().LoadRepositoryFromProvider(context.Background(), provider, "org/project/repo")
	require.NoError(t, err)

	reqs := srv.recorded()
	require.Len(t, reqs, 1)
	require.Contains(t, reqs[0].path, "/repo", "only the repository name is looked up")

	_, _, err = newLoopbackImporter().LoadRepositoryFromProvider(context.Background(), provider, "org/project")
	require.Error(t, err)
}

func TestSvaLoadRepositoryFromProviderInvalidInput(t *testing.T) {
	t.Parallel()

	imp := newLoopbackImporter()

	_, _, err := imp.LoadRepositoryFromProvider(context.Background(), Provider{Type: ProviderTypeGitHub}, "")
	require.Error(t, err)

	_, _, err = imp.LoadRepositoryFromProvider(context.Background(), Provider{}, "owner/repo")
	require.Error(t, err)

	_, _, err = imp.LoadRepositoriesFromProviderSpace(context.Background(), Provider{Type: ProviderTypeGitHub}, "", false)
	require.Error(t, err)

	_, _, err = imp.LoadRepositoriesFromProviderSpace(context.Background(), Provider{}, "owner", false)
	require.Error(t, err)
}

func TestSvaLoadRepositoryUserLookupFailure(t *testing.T) {
	t.Parallel()

	srv := svaNewProviderServer(t, http.StatusUnauthorized, map[string]string{"message": "bad credentials"})
	provider := Provider{Type: ProviderTypeGitHub, Host: srv.URL, Password: "secret"}

	_, _, err := newLoopbackImporter().LoadRepositoryFromProvider(context.Background(), provider, "owner/repo")
	require.ErrorContains(t, err, "Bad credentials")

	_, _, err = newLoopbackImporter().LoadRepositoriesFromProviderSpace(context.Background(), provider, "owner", false)
	require.ErrorContains(t, err, "Bad credentials")
}

func TestSvaLoadRepositoriesFromProviderSpacePaginationAndFilter(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var pages []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		pages = append(pages, r.URL.Query().Get("page"))
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", `<`+"http://"+r.Host+`/orgs/owner/repos?page=2>; rel="next"`)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "one", "private": false, "default_branch": "main", "clone_url": "https://x/owner/one.git",
					"owner": map[string]any{"login": "owner"}, "full_name": "owner/one"},
				{"name": "foreign", "private": false, "default_branch": "main", "clone_url": "https://x/other/foreign.git",
					"owner": map[string]any{"login": "other"}, "full_name": "other/foreign"},
			})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "two", "private": true, "default_branch": "dev", "clone_url": "https://x/owner/two.git",
					"owner": map[string]any{"login": "OWNER"}, "full_name": "OWNER/two"},
			})
		}
	}))
	t.Cleanup(srv.Close)

	provider := Provider{Type: ProviderTypeGitea, Host: srv.URL, Username: "u", Password: "p"}

	imp := newLoopbackImporter()
	repos, _, err := imp.LoadRepositoriesFromProviderSpace(context.Background(), provider, "owner", false)
	require.NoError(t, err)

	names := make([]string, len(repos))
	for i := range repos {
		names[i] = repos[i].Identifier
	}
	require.Equal(t, []string{"one", "two"}, names, "repositories of other namespaces are filtered out")
	require.True(t, repos[1].IsPublic == false)
	require.Equal(t, "dev", repos[1].DefaultBranch)

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(pages), 2, "all pages must be requested")
}

func TestSvaLoadRepositoriesFromProviderSpaceGitHubSearch(t *testing.T) {
	t.Parallel()

	srv := svaNewProviderServer(t, http.StatusOK, map[string]any{
		"total_count": 1,
		"items": []map[string]any{{
			"name": "one", "private": false, "default_branch": "main", "clone_url": "https://x/owner/one.git",
			"full_name": "owner/one", "owner": map[string]any{"login": "owner"},
		}},
	})

	provider := Provider{Type: ProviderTypeGitHub, Host: srv.URL, Username: "u", Password: "p"}

	imp := newLoopbackImporter()
	repos, _, err := imp.LoadRepositoriesFromProviderSpace(context.Background(), provider, "owner", false)
	require.NoError(t, err)
	require.Len(t, repos, 1)
	require.Equal(t, "one", repos[0].Identifier)
	require.True(t, repos[0].IsPublic)
}

func TestSvaLoadRepositoriesFromProviderSpaceErrors(t *testing.T) {
	t.Parallel()

	for _, providerType := range []ProviderType{ProviderTypeGitHub, ProviderTypeGitea} {
		srv := svaNewProviderServer(t, http.StatusForbidden, map[string]string{"message": "forbidden"})
		provider := Provider{Type: providerType, Host: srv.URL, Username: "u", Password: "p"}

		_, _, err := newLoopbackImporter().LoadRepositoriesFromProviderSpace(context.Background(), provider, "owner", false)
		require.ErrorContains(t, err, "Access denied", providerType)
	}

	_, _, err := newLoopbackImporter().LoadRepositoriesFromProviderSpace(
		context.Background(), Provider{Type: ProviderTypeGitea}, "owner", false)
	require.ErrorContains(t, err, "Could not create client")
}

func TestSvaConvertSCMError(t *testing.T) {
	t.Parallel()

	provider := Provider{Type: ProviderTypeGitHub, Host: "https://example.com"}

	tests := []struct {
		name     string
		resp     *scm.Response
		err      error
		wantNil  bool
		wantText string
	}{
		{name: "no error", wantNil: true},
		{name: "no response", err: errSvaBoom, wantText: "Failed to make an HTTP request to github"},
		{name: "moved", resp: &scm.Response{Status: http.StatusMovedPermanently}, err: errSvaBoom, wantText: "Redirects"},
		{name: "found", resp: &scm.Response{Status: http.StatusFound}, err: errSvaBoom, wantText: "Redirects"},
		{name: "see other", resp: &scm.Response{Status: http.StatusSeeOther}, err: errSvaBoom, wantText: "Redirects"},
		{
			name: "temporary", resp: &scm.Response{Status: http.StatusTemporaryRedirect}, err: errSvaBoom,
			wantText: "Redirects",
		},
		{
			name: "permanent", resp: &scm.Response{Status: http.StatusPermanentRedirect}, err: errSvaBoom,
			wantText: "Redirects",
		},
		{
			name: "not found", resp: &scm.Response{Status: http.StatusNotFound}, err: errSvaBoom,
			wantText: "Couldn't find owner/repo",
		},
		{
			name: "unauthorized", resp: &scm.Response{Status: http.StatusUnauthorized}, err: errSvaBoom,
			wantText: "Bad credentials",
		},
		{name: "forbidden", resp: &scm.Response{Status: http.StatusForbidden}, err: errSvaBoom, wantText: "Access denied"},
		{
			name: "server error", resp: &scm.Response{Status: http.StatusInternalServerError}, err: errSvaBoom,
			wantText: "Failed to fetch",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := convertSCMError(context.Background(), provider, "owner/repo", test.resp, test.err)
			if test.wantNil {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), test.wantText)
			// The details of the failure must never reach the caller.
			require.NotContains(t, err.Error(), errSvaBoom.Error())
			require.NotContains(t, err.Error(), "example.com")
		})
	}
}

func TestSvaSlugExtraction(t *testing.T) {
	t.Parallel()

	t.Run("org and project", func(t *testing.T) {
		t.Parallel()

		org, project, err := extractOrgAndProjectFromSlug("org/project/repo")
		require.NoError(t, err)
		require.Equal(t, "org", org)
		require.Equal(t, "project", project)

		org, project, err = extractOrgAndProjectFromSlug("org/project")
		require.NoError(t, err)
		require.Equal(t, "org", org)
		require.Equal(t, "project", project)

		for _, slug := range []string{"org", "", "a/b/c/d"} {
			_, _, err = extractOrgAndProjectFromSlug(slug)
			require.Error(t, err, slug)
		}
	})

	t.Run("repo", func(t *testing.T) {
		t.Parallel()

		repo, err := extractRepoFromSlug("org/project/repo")
		require.NoError(t, err)
		require.Equal(t, "repo", repo)

		for _, slug := range []string{"org/project", "repo", "a/b/c/d"} {
			_, err = extractRepoFromSlug(slug)
			require.Error(t, err, slug)
		}
	})

	t.Run("harness scope", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			slug                        string
			account, org, project, repo string
			wantErr                     bool
		}{
			{slug: "acc/repo", account: "acc", repo: "repo"},
			{slug: "acc/org/repo", account: "acc", org: "org", repo: "repo"},
			{slug: "acc/org/project/repo", account: "acc", org: "org", project: "project", repo: "repo"},
			{slug: "repo", wantErr: true},
			{slug: "a/b/c/d/e", wantErr: true},
		}
		for _, test := range tests {
			account, org, project, repo, err := extractHarnessScope(test.slug)
			if test.wantErr {
				require.Error(t, err, test.slug)
				continue
			}
			require.NoError(t, err, test.slug)
			require.Equal(t, test.account, account)
			require.Equal(t, test.org, org)
			require.Equal(t, test.project, project)
			require.Equal(t, test.repo, repo)
		}
	})
}

func TestSvaRepositoryInfoToRepo(t *testing.T) {
	t.Parallel()

	info := RepositoryInfo{Identifier: "repo", IsPublic: true, DefaultBranch: "trunk"}

	repo, isPublic := info.ToRepo(9, "root/space", "repo", "a description", &types.Principal{ID: 4})
	require.True(t, isPublic)
	require.Equal(t, int64(9), repo.ParentID)
	require.Equal(t, "repo", repo.Identifier)
	require.Equal(t, "root/space/repo", repo.Path)
	require.Equal(t, "a description", repo.Description)
	require.Equal(t, "trunk", repo.DefaultBranch)
	require.Equal(t, int64(4), repo.CreatedBy)
	require.Equal(t, enum.RepoStateGitImport, repo.State)
	require.True(t, strings.HasPrefix(repo.GitUID, "importing-"), "the git UID is a placeholder until the import runs")
	require.JSONEq(t, `{}`, string(repo.Tags))
}

func TestSvaNewRepoGitUIDIsPerRepository(t *testing.T) {
	t.Parallel()

	one := NewRepo(1, "space", "one", "", &types.Principal{}, "main")
	other := NewRepo(1, "space", "other", "", &types.Principal{}, "main")
	require.NotEqual(t, strings.Split(one.GitUID, "-")[1], strings.Split(other.GitUID, "-")[1])
	require.Equal(t, hash("1:one"), strings.Split(one.GitUID, "-")[1])
	require.Len(t, hash("anything"), 16)
}

func TestSvaConnectorDefaults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	noop := connectorServiceNoop{}

	_, err := noop.GetAccessInfo(ctx, ConnectorDef{})
	require.True(t, errors.IsInvalidArgument(err))

	path, identifier := noop.ResolveConnectorRef("parent/space", "conn")
	require.Equal(t, "parent/space", path)
	require.Equal(t, "conn", identifier)
	require.Equal(t, "conn", noop.EncodeConnectorRef("parent/space", "parent/space", "conn"))

	err = webhookServiceNoop{}.UpsertWebhook(ctx, UpsertWebhookInput{})
	require.True(t, errors.IsInvalidArgument(err))

	require.NotErrorIs(t, ErrWebhookUpsertUserFault, ErrWebhookUpsertInfra)
}

func TestSvaAccessInfoURLWithCredentials(t *testing.T) {
	t.Parallel()

	got, err := AccessInfo{Username: "user", Password: "p@ss/word", URL: "https://example.com/owner/repo.git"}.
		URLWithCredentials()
	require.NoError(t, err)
	require.Equal(t, "https://user:p%40ss%2Fword@example.com/owner/repo.git", got)

	_, err = AccessInfo{URL: "://missing-scheme"}.URLWithCredentials()
	require.ErrorContains(t, err, "failed to parse repository clone url")
}

func TestSvaTransportsReturnBaseWithoutCredentials(t *testing.T) {
	t.Parallel()

	base := http.DefaultTransport

	require.Equal(t, base, oauthTransport(base, "", "Bearer"))
	require.Equal(t, base, authHeaderTransport(base, ""))
	require.Equal(t, base, basicAuthTransport(base, "", ""))
	require.Equal(t, base, apiKeyTransport(base, ""))

	require.NotEqual(t, base, oauthTransport(base, "t", "Bearer"))
	require.NotEqual(t, base, authHeaderTransport(base, "t"))
	require.NotEqual(t, base, basicAuthTransport(base, "u", ""))
	require.NotEqual(t, base, basicAuthTransport(base, "", "p"))
	require.NotEqual(t, base, apiKeyTransport(base, "k"))
}

func TestSvaImporterBlocksPrivateProviderHosts(t *testing.T) {
	t.Parallel()

	// The default policy only allows public addresses, so a loopback provider host is refused.
	imp := newTestImporter(netpolicy.Policy{})

	_, err := imp.getScmClientWithTransport(Provider{Type: ProviderTypeGitHub, Host: "http://127.0.0.1:8080"}, "", false)
	require.Error(t, err)
}
