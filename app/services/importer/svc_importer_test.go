// Copyright 2026 Harness, Inc.
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
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/git"
	gitapi "github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/go-scm/scm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSvcProviderTypeEnum(t *testing.T) {
	values := ProviderType("").Enum()
	assert.Len(t, values, 8)
	assert.Contains(t, values, ProviderTypeGitHub)
	assert.Contains(t, values, ProviderTypeHarness)
}

func TestSvcHash(t *testing.T) {
	assert.Equal(t, hash("a"), hash("a"))
	assert.NotEqual(t, hash("a"), hash("b"))
	assert.Len(t, hash("anything"), 16, "10 bytes of base32 are 16 characters")
}

func TestSvcNewRepoAndToRepo(t *testing.T) {
	principal := &types.Principal{ID: 5}

	repo := NewRepo(10, "root/space", "my-repo", "desc", principal, "main")
	assert.Equal(t, int64(10), repo.ParentID)
	assert.Equal(t, "my-repo", repo.Identifier)
	assert.Equal(t, "root/space/my-repo", repo.Path)
	assert.Equal(t, "desc", repo.Description)
	assert.Equal(t, "main", repo.DefaultBranch)
	assert.Equal(t, int64(5), repo.CreatedBy)
	assert.Equal(t, enum.RepoStateGitImport, repo.State)
	assert.Contains(t, repo.GitUID, "importing-")
	assert.JSONEq(t, `{}`, string(repo.Tags))
	assert.NotZero(t, repo.Created)

	other := NewRepo(11, "root/space", "my-repo", "", principal, "main")
	assert.NotEqual(t, repo.GitUID, other.GitUID, "the temporary git uid depends on the space")

	info := &RepositoryInfo{DefaultBranch: "trunk", IsPublic: true}
	got, isPublic := info.ToRepo(10, "root", "r", "d", principal)
	assert.True(t, isPublic)
	assert.Equal(t, "trunk", got.DefaultBranch)
	assert.Equal(t, "root/r", got.Path)
}

// svcHeaderCapture returns a server that records the headers of the last request.
func svcHeaderCapture(t *testing.T) (*httptest.Server, *http.Header) {
	t.Helper()
	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

func svcGet(t *testing.T, rt http.RoundTripper, url string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
}

func TestSvcTransports(t *testing.T) {
	srv, captured := svcHeaderCapture(t)
	base := http.DefaultTransport

	t.Run("empty credentials return the base transport", func(t *testing.T) {
		assert.Equal(t, base, oauthTransport(base, "", "Bearer"))
		assert.Equal(t, base, authHeaderTransport(base, ""))
		assert.Equal(t, base, basicAuthTransport(base, "", ""))
		assert.Equal(t, base, apiKeyTransport(base, ""))
	})
	t.Run("oauth", func(t *testing.T) {
		svcGet(t, oauthTransport(base, "tok", "Bearer"), srv.URL)
		assert.Equal(t, "Bearer tok", captured.Get("Authorization"))
	})
	t.Run("authorization header", func(t *testing.T) {
		svcGet(t, authHeaderTransport(base, "tok"), srv.URL)
		assert.Equal(t, "token tok", captured.Get("Authorization"))
	})
	t.Run("basic auth", func(t *testing.T) {
		svcGet(t, basicAuthTransport(base, "user", "pass"), srv.URL)
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
		assert.Equal(t, want, captured.Get("Authorization"))
	})
	t.Run("api key", func(t *testing.T) {
		svcGet(t, apiKeyTransport(base, "key"), srv.URL)
		assert.Equal(t, "key", captured.Get("x-api-key"))
	})
}

func TestSvcSlugExtraction(t *testing.T) {
	t.Run("org and project", func(t *testing.T) {
		tests := []struct {
			slug    string
			org     string
			project string
			wantErr bool
		}{
			{"org/project", "org", "project", false},
			{"org/project/repo", "org", "project", false},
			{"org", "", "", true},
			{"a/b/c/d", "", "", true},
		}
		for _, tt := range tests {
			t.Run(tt.slug, func(t *testing.T) {
				org, project, err := extractOrgAndProjectFromSlug(tt.slug)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.org, org)
				assert.Equal(t, tt.project, project)
			})
		}
	})
	t.Run("repo", func(t *testing.T) {
		repo, err := extractRepoFromSlug("org/project/repo")
		require.NoError(t, err)
		assert.Equal(t, "repo", repo)

		_, err = extractRepoFromSlug("org/project")
		require.Error(t, err)
		_, err = extractRepoFromSlug("org/project/repo/extra")
		require.Error(t, err)
	})
	t.Run("harness scope", func(t *testing.T) {
		tests := []struct {
			slug                        string
			account, org, project, repo string
			wantErr                     bool
		}{
			{"acc/repo", "acc", "", "", "repo", false},
			{"acc/org/repo", "acc", "org", "", "repo", false},
			{"acc/org/proj/repo", "acc", "org", "proj", "repo", false},
			{"repo", "", "", "", "", true},
			{"a/b/c/d/e", "", "", "", "", true},
		}
		for _, tt := range tests {
			t.Run(tt.slug, func(t *testing.T) {
				account, org, project, repo, err := extractHarnessScope(tt.slug)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, []string{tt.account, tt.org, tt.project, tt.repo}, []string{account, org, project, repo})
			})
		}
	})
}

func TestSvcMatchesNamespace(t *testing.T) {
	tests := []struct {
		name      string
		provider  ProviderType
		namespace string
		slug      string
		subgroups bool
		wantMatch bool
	}{
		{"exact", ProviderTypeGitHub, "org", "org", false, true},
		{"case insensitive", ProviderTypeGitHub, "Org", "oRG", false, true},
		{"different", ProviderTypeGitHub, "org", "other", false, false},
		{"github ignores subgroups flag", ProviderTypeGitHub, "org/sub", "org", true, false},
		{"gitlab subgroup", ProviderTypeGitLab, "group/sub", "group", true, true},
		{"gitlab same group", ProviderTypeGitLab, "Group", "group", true, true},
		{"gitlab prefix is not a subgroup", ProviderTypeGitLab, "grouptoo", "group", true, false},
		{"gitlab without subgroups", ProviderTypeGitLab, "group/sub", "group", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMatch, matchesNamespace(tt.provider, tt.namespace, tt.slug, tt.subgroups))
		})
	}
}

func TestSvcConvertSCMError(t *testing.T) {
	ctx := context.Background()
	provider := Provider{Type: ProviderTypeGitHub, Host: "https://internal.example"}
	boom := errors.New("dial tcp 10.0.0.1:22: connection refused")

	assert.NoError(t, convertSCMError(ctx, provider, "org/repo", nil, nil))

	tests := []struct {
		name        string
		resp        *scm.Response
		wantMessage string
	}{
		{"no response", nil, "Failed to make an HTTP request"},
		{"moved", &scm.Response{Status: http.StatusMovedPermanently}, "Redirects are not supported"},
		{"found", &scm.Response{Status: http.StatusFound}, "Redirects are not supported"},
		{"temporary redirect", &scm.Response{Status: http.StatusTemporaryRedirect}, "Redirects are not supported"},
		{"not found", &scm.Response{Status: http.StatusNotFound}, "Couldn't find org/repo at github"},
		{"unauthorized", &scm.Response{Status: http.StatusUnauthorized}, "Bad credentials"},
		{"forbidden", &scm.Response{Status: http.StatusForbidden}, "Access denied"},
		{"other", &scm.Response{Status: http.StatusInternalServerError}, "Failed to fetch org/repo from github"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := convertSCMError(ctx, provider, "org/repo", tt.resp, boom)
			require.Error(t, err)
			var uerr *usererror.Error
			require.ErrorAs(t, err, &uerr)
			assert.Contains(t, uerr.Message, tt.wantMessage)
			assert.NotContains(t, uerr.Message, "10.0.0.1", "transport details must not leak to the caller")
			assert.NotContains(t, uerr.Message, "internal.example", "the provider host must not leak to the caller")
		})
	}
}

type svcPipelineConverter struct {
	out []byte
	err error
}

func (c svcPipelineConverter) ConvertBytes([]byte) ([]byte, error) { return c.out, c.err }

func TestSvcConvertPipelineFiles(t *testing.T) {
	ctx := context.Background()
	gen := func(c svcPipelineConverter) func() pipelineConverter {
		return func() pipelineConverter { return c }
	}

	t.Run("a single file becomes the default pipeline", func(t *testing.T) {
		out := convertPipelineFiles(ctx, []pipelineFile{{OriginalPath: ".drone.yml", Content: []byte("in")}},
			gen(svcPipelineConverter{out: []byte("converted")}))
		require.Len(t, out, 1)
		assert.Equal(t, "default-pipeline", out[0].Name)
		assert.Equal(t, ".harness/pipeline.yaml", out[0].ConvertedPath)
		assert.Equal(t, ".drone.yml", out[0].OriginalPath)
		assert.Equal(t, "converted", string(out[0].Content))
	})
	t.Run("multiple files are named after their basename", func(t *testing.T) {
		out := convertPipelineFiles(ctx, []pipelineFile{
			{OriginalPath: ".github/workflows/build.yml"}, {OriginalPath: ".github/workflows/test.yaml"},
		}, gen(svcPipelineConverter{out: []byte("x")}))
		require.Len(t, out, 2)
		assert.Equal(t, "pipeline-build", out[0].Name)
		assert.Equal(t, ".harness/build.yaml", out[0].ConvertedPath)
		assert.Equal(t, "pipeline-test", out[1].Name)
		assert.Equal(t, ".harness/test.yaml", out[1].ConvertedPath)
	})
	t.Run("files that fail to convert are skipped", func(t *testing.T) {
		out := convertPipelineFiles(ctx, []pipelineFile{{OriginalPath: "a.yml"}},
			gen(svcPipelineConverter{err: errors.New("boom")}))
		assert.Empty(t, out)
	})
	t.Run("no files", func(t *testing.T) {
		assert.Empty(t, convertPipelineFiles(ctx, nil, gen(svcPipelineConverter{})))
	})
}

type svcGit struct {
	git.Interface
	files map[string][]gitapi.FileContent
	err   error
	calls []string
}

func (g *svcGit) MatchFiles(_ context.Context, p *git.MatchFilesParams) (*git.MatchFilesOutput, error) {
	g.calls = append(g.calls, p.DirPath+"|"+p.Pattern)
	if g.err != nil {
		return nil, g.err
	}
	return &git.MatchFilesOutput{Files: g.files[p.DirPath+"|"+p.Pattern]}, nil
}

func TestSvcMatchFiles(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{GitUID: "uid", DefaultBranch: "main"}

	g := &svcGit{files: map[string][]gitapi.FileContent{"|.drone.yml": {{Path: ".drone.yml", Content: []byte("c")}}}}
	imp := &Importer{git: g}

	files, err := imp.matchFiles(ctx, repo, "main", "", ".drone.yml", 100)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, ".drone.yml", files[0].OriginalPath)
	assert.Equal(t, "c", string(files[0].Content))
	assert.Empty(t, files[0].Name)

	g.err = errors.New("git down")
	_, err = imp.matchFiles(ctx, repo, "main", "", ".drone.yml", 100)
	require.Error(t, err)
}

func TestSvcConvertPipelinesLookupOrder(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{GitUID: "uid", DefaultBranch: "main"}

	t.Run("nothing found", func(t *testing.T) {
		g := &svcGit{files: map[string][]gitapi.FileContent{}}
		out := (&Importer{git: g}).convertPipelines(ctx, repo)
		assert.Empty(t, out)
		assert.Equal(t, []string{
			"|.drone.yml", "|bitbucket-pipelines.yml", "|.gitlab-ci.yml", ".circleci|config.yml",
			".github/workflows|*.yml", ".github/workflows|*.yaml",
		}, g.calls, "supported providers are probed in a fixed order")
	})
	t.Run("git failures are tolerated", func(t *testing.T) {
		g := &svcGit{err: errors.New("git down")}
		assert.Empty(t, (&Importer{git: g}).convertPipelines(ctx, repo))
	})
	t.Run("the first provider with a convertible pipeline wins", func(t *testing.T) {
		g := &svcGit{files: map[string][]gitapi.FileContent{
			"|.drone.yml": {{Path: ".drone.yml", Content: []byte("kind: pipeline\nname: default\n")}},
		}}
		out := (&Importer{git: g}).convertPipelines(ctx, repo)
		require.Len(t, out, 1)
		assert.Equal(t, ".drone.yml", out[0].OriginalPath)
		assert.Equal(t, ".harness/pipeline.yaml", out[0].ConvertedPath)
		assert.NotEmpty(t, out[0].Content)
		assert.Equal(t, []string{"|.drone.yml"}, g.calls, "later providers are not probed")
	})
}

func TestSvcJobInputRoundTrips(t *testing.T) {
	t.Run("link job", func(t *testing.T) {
		j := &JobRepositoryLink{}
		def, err := j.getJobDef(j.jobIDFromRepoID(42), JobLinkRepoInput{RepoID: 42, IsPublic: true})
		require.NoError(t, err)
		assert.Equal(t, "link-repo-42", def.UID)
		assert.Equal(t, jobTypeRepositoryLink, def.Type)
		assert.Equal(t, jobLinkRepoMaxRetries, def.MaxRetries)
		assert.Equal(t, jobLinkRepoMaxDuration, def.Timeout)

		in, err := j.getJobInput(def.Data)
		require.NoError(t, err)
		assert.Equal(t, JobLinkRepoInput{RepoID: 42, IsPublic: true}, in)

		_, err = j.getJobInput("%%%")
		require.Error(t, err)
		_, err = j.getJobInput(base64.StdEncoding.EncodeToString([]byte("not json")))
		require.Error(t, err)
	})
	t.Run("reference sync job", func(t *testing.T) {
		j := &JobReferenceSync{}
		input := ReferenceSyncInput{
			SourceRepoID: 1, TargetRepoID: 2, RefSpecType: RefSpecTypeAllBranches, SourceRef: "a", TargetRef: "b",
		}
		def, err := j.getJobDef("uid", input)
		require.NoError(t, err)
		assert.Equal(t, refSyncJobType, def.Type)
		assert.Equal(t, refSyncJobMaxRetries, def.MaxRetries)
		assert.Equal(t, refSyncJobMaxDuration, def.Timeout)

		got, err := j.getJobInput(def.Data)
		require.NoError(t, err)
		assert.Equal(t, input, got)

		_, err = j.getJobInput("not json")
		require.Error(t, err)
	})
	t.Run("repository job", func(t *testing.T) {
		j := &JobRepository{encrypter: svcEncrypter{}}
		assert.Equal(t, "import-repo-7", j.jobIDFromRepoID(7))

		input := RepositoryInput{Input: Input{RepoID: 7, Public: true}}
		def, err := j.getJobDef("uid", input)
		require.NoError(t, err)
		assert.Equal(t, jobTypeRepositoryImport, def.Type)
		assert.Equal(t, importJobMaxRetries, def.MaxRetries)
		assert.Equal(t, importJobMaxDuration, def.Timeout)

		got, err := j.getJobInput(def.Data)
		require.NoError(t, err)
		assert.Equal(t, input, got)

		_, err = j.getJobInput("%%%")
		require.Error(t, err)
		_, err = j.getJobInput(base64.StdEncoding.EncodeToString([]byte("enc:not json")))
		require.Error(t, err)

		_, err = (&JobRepository{encrypter: svcEncrypter{encErr: errors.New("boom")}}).getJobDef("uid", input)
		require.Error(t, err)
		_, err = (&JobRepository{encrypter: svcEncrypter{decErr: errors.New("boom")}}).getJobInput(def.Data)
		require.Error(t, err)
	})
}

type svcEncrypter struct {
	encErr error
	decErr error
}

func (e svcEncrypter) Encrypt(plaintext string) ([]byte, error) {
	if e.encErr != nil {
		return nil, e.encErr
	}
	return []byte("enc:" + plaintext), nil
}

func (e svcEncrypter) Decrypt(ciphertext []byte) (string, error) {
	if e.decErr != nil {
		return "", e.decErr
	}
	return string(ciphertext[len("enc:"):]), nil
}
