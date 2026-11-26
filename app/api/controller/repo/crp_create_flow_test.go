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

package repo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/netpolicy"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func crpCreateInput() *CreateInput {
	return &CreateInput{ParentRef: "3", Identifier: "newrepo"}
}

func crpGitCreated(e *crpEnv) {
	e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(&git.CreateRepositoryOutput{UID: "newuid"}, nil)
}

func TestCrpCreate_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		mutate func(in *CreateInput, e *crpEnv)
	}{
		{name: "missing parent", mutate: func(in *CreateInput, _ *crpEnv) { in.ParentRef = "" }},
		{name: "non positive numeric parent", mutate: func(in *CreateInput, _ *crpEnv) { in.ParentRef = "-4" }},
		{name: "invalid identifier", mutate: func(_ *CreateInput, e *crpEnv) {
			e.c.identifierCheck = func(string, *auth.Session) error { return errCrpBoom }
		}},
		{name: "description too long", mutate: func(in *CreateInput, _ *crpEnv) {
			in.Description = strings.Repeat("a", 1025)
		}},
		{name: "description with control characters", mutate: func(in *CreateInput, _ *crpEnv) {
			in.Description = "bad\x00text"
		}},
		{name: "invalid tag", mutate: func(in *CreateInput, _ *crpEnv) {
			in.Tags = types.RepoTags{"key:with:colon": "value"}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			in := crpCreateInput()
			tt.mutate(in, e)

			_, err := e.c.Create(ctx, crpSession(), in)

			require.Error(t, err)
			assert.False(t, e.repoStore.called("Create"))
		})
	}
}

func TestCrpCreate_Defaults(t *testing.T) {
	ctx := context.Background()
	e := crpNewEnv(t)
	crpGitCreated(e)
	in := crpCreateInput()
	in.UID = "legacyid"
	in.Identifier = ""
	in.Description = "  padded  "

	out, err := e.c.Create(ctx, crpSession(), in)

	require.NoError(t, err)
	assert.Equal(t, "legacyid", out.Identifier)
	assert.Equal(t, "padded", out.Description)
	assert.Equal(t, crpDefaultBranch, out.DefaultBranch)
}

func TestCrpCreate_Success(t *testing.T) {
	ctx := context.Background()

	t.Run("empty repository", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("CreateRepository", mock.Anything, mock.MatchedBy(func(p *git.CreateRepositoryParams) bool {
			return len(p.Files) == 0 && p.DefaultBranch == crpDefaultBranch && p.Actor.Email == crpAliceEmail
		})).Return(&git.CreateRepositoryOutput{UID: "newuid"}, nil)

		out, err := e.c.Create(ctx, crpSession(), crpCreateInput())

		require.NoError(t, err)
		assert.Equal(t, "newuid", out.GitUID)
		assert.True(t, out.IsEmpty)
		assert.Equal(t, "https://git.example.com/space/newrepo.git", out.GitURL)
		assert.False(t, out.IsPublic)
		assert.EqualValues(t, 3, out.ParentID)
		assert.EqualValues(t, 3, out.RootSpaceID)
		assert.Empty(t, crpIndexerOf(t, e).indexed)
		assert.False(t, e.public.public["space/newrepo"])
	})

	t.Run("with readme, license and gitignore", func(t *testing.T) {
		e := crpNewEnv(t)
		var files []git.File
		e.git.On("CreateRepository", mock.Anything, mock.MatchedBy(func(p *git.CreateRepositoryParams) bool {
			files = p.Files

			return true
		})).Return(&git.CreateRepositoryOutput{UID: "newuid"}, nil)
		in := crpCreateInput()
		in.Description = "my description"
		in.IsPublic = true
		in.CreateFileOptions = CreateFileOptions{Readme: true, License: "mit", GitIgnore: "Go"}

		out, err := e.c.Create(ctx, crpSession(), in)

		require.NoError(t, err)
		require.Len(t, files, 3)
		assert.Equal(t, "README.md", files[0].Path)
		assert.Equal(t, "# newrepo\nmy description", string(files[0].Content))
		assert.Equal(t, "LICENSE", files[1].Path)
		assert.Equal(t, ".gitignore", files[2].Path)
		assert.False(t, out.IsEmpty)
		assert.True(t, out.IsPublic)
		assert.True(t, e.public.public["space/newrepo"])
		assert.Equal(t, []int64{out.ID}, crpIndexerOf(t, e).indexed)
	})

	t.Run("license none adds no file and indexing failure is tolerated", func(t *testing.T) {
		e := crpNewEnv(t)
		crpGitCreated(e)
		e.c.indexer = &crpIndexer{err: errCrpBoom}
		in := crpCreateInput()
		in.CreateFileOptions = CreateFileOptions{Readme: true, License: "none"}

		out, err := e.c.Create(ctx, crpSession(), in)

		require.NoError(t, err)
		assert.False(t, out.IsEmpty)
	})
}

func TestCrpCreate_Failures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(e *crpEnv)
		in    func() *CreateInput
		check func(t *testing.T, e *crpEnv, err error)
	}{
		{
			name: "unknown parent space",
			in:   func() *CreateInput { in := crpCreateInput(); in.ParentRef = "99"; return in },
			check: func(t *testing.T, _ *crpEnv, err error) {
				require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
			},
		},
		{
			name:  "no permission to create repos",
			setup: func(e *crpEnv) { e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom}) },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "public access support lookup failure",
			setup: func(e *crpEnv) { e.public.supportedErr = errCrpBoom },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "public repositories are disabled",
			setup: func(e *crpEnv) { e.public.unsupported = true },
			in:    func() *CreateInput { in := crpCreateInput(); in.IsPublic = true; return in },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errPublicRepoCreationDisabled) },
		},
		{
			name:  "repo check rejects",
			setup: func(e *crpEnv) { e.c.repoCheck = crpRepoCheck{createErr: errCrpBoom} },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name: "identifier already taken",
			in:   func() *CreateInput { in := crpCreateInput(); in.Identifier = crpRepoIdentifier; return in },
			check: func(t *testing.T, _ *crpEnv, err error) {
				require.Error(t, err)
				assert.Equal(t, http.StatusConflict, crpUserErrStatus(t, err))
			},
		},
		{
			name: "git failure",
			setup: func(e *crpEnv) {
				e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, errCrpBoom)
				assert.False(t, e.repoStore.called("Create"))
			},
		},
		{
			name: "repo quota reached cleans up git",
			setup: func(e *crpEnv) {
				crpGitCreated(e)
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.c.resourceLimiter = crpRepoCountLimiter{err: errCrpBoom}
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, limiter.ErrMaxNumReposReached)
				e.git.AssertNumberOfCalls(t, "DeleteRepository", 1)
			},
		},
		{
			name: "storage over limit",
			setup: func(e *crpEnv) {
				crpGitCreated(e)
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.c.resourceLimiter = restoreLimiterStub{
					rootSpaceStorageErr: &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit},
				}
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.Error(t, err)
				assert.False(t, e.repoStore.called("Create"))
			},
		},
		{
			name: "space lookup failure",
			setup: func(e *crpEnv) {
				crpGitCreated(e)
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.spaces.err = errCrpBoom
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name: "store failure",
			setup: func(e *crpEnv) {
				crpGitCreated(e)
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(errCrpBoom)
				e.repoStore.errs["Create"] = errCrpBoom
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, errCrpBoom)
				e.git.AssertNumberOfCalls(t, "DeleteRepository", 1)
			},
		},
		{
			name: "public access failure rolls the repository back",
			setup: func(e *crpEnv) {
				crpGitCreated(e)
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
				e.public.setErr = errCrpBoom
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, errCrpBoom)
				assert.True(t, e.repoStore.called("Purge"))
			},
		},
		{
			name: "public access failure and failed cleanup",
			setup: func(e *crpEnv) {
				crpGitCreated(e)
				e.public.setErr = errCrpBoom
				e.public.deleteErr = errCrpBoom
			},
			check: func(t *testing.T, e *crpEnv, err error) {
				require.ErrorIs(t, err, errCrpBoom)
				assert.False(t, e.repoStore.called("Purge"))
			},
		},
		{
			name: "public access failure and failed purge",
			setup: func(e *crpEnv) {
				crpGitCreated(e)
				e.public.setErr = errCrpBoom
				e.repoStore.errs["Purge"] = errCrpBoom
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}
			in := crpCreateInput()
			if tt.in != nil {
				in = tt.in()
			}

			_, err := e.c.Create(ctx, crpSession(), in)

			tt.check(t, e, err)
		})
	}
}

func TestCrpCreate_FailedImportIsReplaced(t *testing.T) {
	ctx := context.Background()
	e := crpNewEnv(t)
	crpGitCreated(e)
	e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
	e.repoStore.repos[crpRepoID].State = enum.RepoStateImportFailed
	in := crpCreateInput()
	in.Identifier = crpRepoIdentifier

	out, err := e.c.Create(ctx, crpSession(), in)

	require.NoError(t, err)
	assert.Equal(t, crpRepoIdentifier, out.Identifier)
	assert.True(t, e.repoStore.called("Purge"))
}

func TestCrpImport_EarlyFailures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		mutate func(in *ImportInput, e *crpEnv)
	}{
		{name: "missing parent", mutate: func(in *ImportInput, _ *crpEnv) { in.ParentRef = "" }},
		{name: "invalid identifier", mutate: func(_ *ImportInput, e *crpEnv) {
			e.c.identifierCheck = func(string, *auth.Session) error { return errCrpBoom }
		}},
		{name: "unknown parent", mutate: func(in *ImportInput, _ *crpEnv) { in.ParentRef = "99" }},
		{name: "no permission", mutate: func(_ *ImportInput, e *crpEnv) {
			e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		}},
		{name: "identifier taken", mutate: func(in *ImportInput, _ *crpEnv) { in.Identifier = crpRepoIdentifier }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			in := &ImportInput{ParentRef: "3", Identifier: "imported", UID: ""}
			tt.mutate(in, e)

			_, err := e.c.Import(ctx, crpSession(), in)

			require.Error(t, err)
			assert.False(t, e.repoStore.called("Create"))
		})
	}

	t.Run("legacy uid and default pipeline option", func(t *testing.T) {
		e := crpNewEnv(t)
		in := &ImportInput{ParentRef: "3", UID: "legacy"}

		require.NoError(t, e.c.sanitizeImportInput(in, crpSession()))

		assert.Equal(t, "legacy", in.Identifier)
		assert.Equal(t, importer.PipelineOptionConvert, in.Pipelines)
	})
}

func TestCrpImportProgress(t *testing.T) {
	ctx := context.Background()

	newEnv := func(t *testing.T, jobs crpJobStore, state enum.RepoState) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.withImportJobs(t, jobs)
		e.setRepo(func(r *types.RepositoryCore) { r.State = state })

		return e
	}

	t.Run("running job", func(t *testing.T) {
		e := newEnv(t, crpJobStore{found: &job.Job{State: job.JobStateRunning, RunProgress: 40}}, enum.RepoStateGitImport)

		progress, err := e.c.ImportProgress(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.Equal(t, job.JobStateRunning, progress.State)
		assert.Equal(t, 40, progress.Progress)
	})

	t.Run("missing job of a repo being imported is a failure", func(t *testing.T) {
		e := newEnv(t, crpJobStore{}, enum.RepoStateImportFailed)

		progress, err := e.c.ImportProgress(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.Equal(t, job.JobStateFailed, progress.State)
	})

	t.Run("missing job of a migrated repo is running", func(t *testing.T) {
		e := newEnv(t, crpJobStore{}, enum.RepoStateMigrateGitPush)

		progress, err := e.c.ImportProgress(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.Equal(t, job.JobStateRunning, progress.State)
	})

	t.Run("no import at all", func(t *testing.T) {
		e := newEnv(t, crpJobStore{}, enum.RepoStateActive)

		_, err := e.c.ImportProgress(ctx, crpSession(), crpRepoRef)

		require.Error(t, err)
		assert.Equal(t, http.StatusNotFound, crpUserErrStatus(t, err))
	})

	t.Run("job store failure", func(t *testing.T) {
		e := newEnv(t, crpJobStore{findErr: errCrpBoom}, enum.RepoStateActive)

		_, err := e.c.ImportProgress(ctx, crpSession(), crpRepoRef)

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("unknown repo and denied access", func(t *testing.T) {
		e := newEnv(t, crpJobStore{}, enum.RepoStateActive)

		_, err := e.c.ImportProgress(ctx, crpSession(), crpUnknownRepoRef)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		_, err = e.c.ImportProgress(ctx, crpSession(), crpRepoRef)
		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpPurgeNoAuth_CancelsRunningImport(t *testing.T) {
	ctx := context.Background()

	t.Run("cancels the import job", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withImportJobs(t, crpJobStore{})
		e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)
		repo := crpDeletedRepo()
		repo.State = enum.RepoStateGitImport

		require.NoError(t, e.c.PurgeNoAuth(ctx, crpSession(), repo))

		assert.True(t, e.repoStore.called("Purge"))
	})

	t.Run("cancel failure aborts the purge", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withImportJobs(t, crpJobStore{findErr: errCrpBoom})
		repo := crpDeletedRepo()
		repo.State = enum.RepoStateGitImport

		err := e.c.PurgeNoAuth(ctx, crpSession(), repo)

		require.Error(t, err)
		assert.False(t, e.repoStore.called("Purge"))
	})
}

// crpProviderServer serves a minimal Gitea-compatible API describing a single repository.
func crpProviderServer(t *testing.T, status int) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/owner/proj" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"id":1,"name":"proj","private":true,"default_branch":"trunk",` +
			`"clone_url":"https://git.example.test/owner/proj.git","owner":{"login":"owner","username":"owner"}}`))
	}))
	t.Cleanup(server.Close)

	return server
}

func crpImportEnv(t *testing.T, jobs crpJobStore, serverURL string) (*crpEnv, *ImportInput) {
	t.Helper()

	e := crpNewEnv(t)
	imp := importer.NewImporter(
		crpDefaultBranch, e.c.urlProvider, e.git, crpTx{}, e.c.repoStore, nil, nil, e.c.repoFinder, nil,
		e.c.indexer, e.c.publicAccess, e.c.eventReporter, e.c.auditService, e.c.settings,
		netpolicy.Policy{AllowLoopback: true}, false,
	)
	e.withImporter(t, jobs, imp)

	return e, &ImportInput{
		ParentRef: "3", Identifier: "imported", Description: "from elsewhere",
		Provider:     importer.Provider{Type: importer.ProviderTypeGitea, Host: serverURL, Username: "u", Password: "token"},
		ProviderRepo: "owner/proj",
	}
}

func TestCrpImport_Success(t *testing.T) {
	ctx := context.Background()
	server := crpProviderServer(t, http.StatusOK)
	e, in := crpImportEnv(t, crpJobStore{}, server.URL)

	out, err := e.c.Import(ctx, crpSession(), in)

	require.NoError(t, err)
	assert.Equal(t, "imported", out.Identifier)
	assert.Equal(t, "https://git.example.com/space/imported.git", out.GitURL)
	assert.Equal(t, enum.RepoStateGitImport, out.State)
	assert.True(t, out.Importing)
	assert.False(t, out.IsPublic)
	assert.True(t, e.repoStore.called("Create"))
}

func TestCrpImport_Failures(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid provider slug", func(t *testing.T) {
		server := crpProviderServer(t, http.StatusOK)
		e, in := crpImportEnv(t, crpJobStore{}, server.URL)
		in.ProviderRepo = "not-a-valid-slug"

		_, err := e.c.Import(ctx, crpSession(), in)

		require.Error(t, err)
		assert.False(t, e.repoStore.called("Create"))
	})

	t.Run("provider rejects the request", func(t *testing.T) {
		server := crpProviderServer(t, http.StatusUnauthorized)
		e, in := crpImportEnv(t, crpJobStore{}, server.URL)

		_, err := e.c.Import(ctx, crpSession(), in)

		require.Error(t, err)
		assert.False(t, e.repoStore.called("Create"))
	})

	tests := []struct {
		name  string
		setup func(e *crpEnv)
		check func(t *testing.T, err error)
	}{
		{name: "repo quota reached", setup: func(e *crpEnv) {
			e.c.resourceLimiter = crpRepoCountLimiter{err: errCrpBoom}
		}, check: func(t *testing.T, err error) { require.ErrorIs(t, err, limiter.ErrMaxNumReposReached) }},
		{name: "storage over limit", setup: func(e *crpEnv) {
			e.c.resourceLimiter = restoreLimiterStub{
				rootSpaceStorageErr: &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit},
			}
		}, check: func(t *testing.T, err error) { require.Error(t, err) }},
		{name: "space lookup failure", setup: func(e *crpEnv) { e.spaces.err = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "store failure", setup: func(e *crpEnv) { e.repoStore.errs["Create"] = errCrpBoom },
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
		{name: "duplicate repository", setup: func(e *crpEnv) { e.repoStore.errs["Create"] = gitnessstore.ErrDuplicate },
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Equal(t, http.StatusConflict, crpUserErrStatus(t, err))
			}},
		{name: "job failure", setup: func(e *crpEnv) {},
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, errCrpBoom) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := crpProviderServer(t, http.StatusOK)
			jobs := crpJobStore{}
			if tt.name == "job failure" {
				jobs.createErr = errCrpBoom
			}
			e, in := crpImportEnv(t, jobs, server.URL)
			tt.setup(e)

			_, err := e.c.Import(ctx, crpSession(), in)

			tt.check(t, err)
		})
	}
}
