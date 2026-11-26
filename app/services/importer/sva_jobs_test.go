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
	"encoding/base64"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func svaNewScheduler(t *testing.T, jobStore *svaJobStore) *job.Scheduler {
	t.Helper()

	scheduler, err := job.NewScheduler(jobStore, nil, nil, nil, "instance", 1, time.Hour)
	require.NoError(t, err)

	return scheduler
}

func TestSvaJobRepositoryDefinitionRoundTrip(t *testing.T) {
	t.Parallel()

	jobRepo := &JobRepository{encrypter: svaEncrypter{}}
	input := RepositoryInput{Input: Input{
		RepoID: 5, Public: true, GitUser: "user", GitPass: "pass",
		CloneURL: "https://example.com/r.git", Pipelines: PipelineOptionConvert,
	}}

	def, err := jobRepo.getJobDef("import-repo-5", input)
	require.NoError(t, err)
	require.Equal(t, "import-repo-5", def.UID)
	require.Equal(t, jobTypeRepositoryImport, def.Type)
	require.Equal(t, importJobMaxRetries, def.MaxRetries)
	require.Equal(t, importJobMaxDuration, def.Timeout)
	require.NoError(t, def.Validate())
	require.NotContains(t, def.Data, "pass", "the job payload must not be stored in plain text")

	got, err := jobRepo.getJobInput(def.Data)
	require.NoError(t, err)
	require.Equal(t, input, got)
}

func TestSvaJobRepositoryDefinitionErrors(t *testing.T) {
	t.Parallel()

	t.Run("encryption failure", func(t *testing.T) {
		t.Parallel()

		jobRepo := &JobRepository{encrypter: svaEncrypter{encryptErr: errSvaBoom}}
		_, err := jobRepo.getJobDef("uid", RepositoryInput{})
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("invalid base64", func(t *testing.T) {
		t.Parallel()

		_, err := (&JobRepository{encrypter: svaEncrypter{}}).getJobInput("***")
		require.ErrorContains(t, err, "base64")
	})

	t.Run("decryption failure", func(t *testing.T) {
		t.Parallel()

		data := base64.StdEncoding.EncodeToString([]byte("x"))
		_, err := (&JobRepository{encrypter: svaEncrypter{decryptErr: errSvaBoom}}).getJobInput(data)
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		data := base64.StdEncoding.EncodeToString(svaReverse([]byte("not json")))
		_, err := (&JobRepository{encrypter: svaEncrypter{}}).getJobInput(data)
		require.ErrorContains(t, err, "unmarshal")
	})

	t.Run("handle fails on undecodable input", func(t *testing.T) {
		t.Parallel()

		_, err := (&JobRepository{encrypter: svaEncrypter{}}).Handle(context.Background(), "***", nil)
		require.Error(t, err)
	})
}

func TestSvaJobRepositoryRun(t *testing.T) {
	t.Parallel()

	jobStore := &svaJobStore{}
	jobRepo := &JobRepository{encrypter: svaEncrypter{}, scheduler: svaNewScheduler(t, jobStore)}
	repo := &types.Repository{ID: 5}
	provider := Provider{Type: ProviderTypeGitHub, Username: "user", Password: "pass"}

	require.NoError(t, jobRepo.Run(
		context.Background(), provider, repo, true, "https://example.com/r.git", PipelineOptionIgnore))
	require.Len(t, jobStore.created, 1)
	require.Equal(t, "import-repo-5", jobStore.created[0].UID)

	input, err := jobRepo.getJobInput(jobStore.created[0].Data)
	require.NoError(t, err)
	require.Equal(t, int64(5), input.RepoID)
	require.True(t, input.Public)
	require.Equal(t, "pass", input.GitPass)
	require.Equal(t, PipelineOptionIgnore, input.Pipelines)

	t.Run("scheduler failure", func(t *testing.T) {
		t.Parallel()

		failing := &JobRepository{
			encrypter: svaEncrypter{},
			scheduler: svaNewScheduler(t, &svaJobStore{err: errSvaBoom}),
		}
		err := failing.Run(context.Background(), provider, repo, false, "https://example.com/r.git", "")
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("encryption failure", func(t *testing.T) {
		t.Parallel()

		failing := &JobRepository{encrypter: svaEncrypter{encryptErr: errSvaBoom}}
		err := failing.Run(context.Background(), provider, repo, false, "https://example.com/r.git", "")
		require.ErrorIs(t, err, errSvaBoom)
	})
}

func TestSvaJobRepositoryRunMany(t *testing.T) {
	t.Parallel()

	provider := Provider{Type: ProviderTypeGitHub, Password: "pass"}

	t.Run("creates one job per repository", func(t *testing.T) {
		t.Parallel()

		jobStore := &svaJobStore{}
		jobRepo := &JobRepository{encrypter: svaEncrypter{}, scheduler: svaNewScheduler(t, jobStore)}

		err := jobRepo.RunMany(context.Background(), "group", provider,
			[]int64{1, 2}, []bool{true, false},
			[]string{"https://example.com/a.git", "https://example.com/b.git"}, PipelineOptionConvert)
		require.NoError(t, err)
		require.Len(t, jobStore.created, 2)
		require.Equal(t, "import-repo-1", jobStore.created[0].UID)
		require.Equal(t, "import-repo-2", jobStore.created[1].UID)
		require.Equal(t, "group", jobStore.created[0].GroupID)

		second, err := jobRepo.getJobInput(jobStore.created[1].Data)
		require.NoError(t, err)
		require.False(t, second.Public)
		require.Equal(t, "https://example.com/b.git", second.CloneURL)
	})

	t.Run("length mismatch", func(t *testing.T) {
		t.Parallel()

		jobRepo := &JobRepository{encrypter: svaEncrypter{}}
		err := jobRepo.RunMany(context.Background(), "group", provider,
			[]int64{1, 2}, []bool{true, false}, []string{"https://example.com/a.git"}, "")
		require.ErrorContains(t, err, "slice length mismatch")
	})

	t.Run("encryption failure", func(t *testing.T) {
		t.Parallel()

		jobRepo := &JobRepository{encrypter: svaEncrypter{encryptErr: errSvaBoom}}
		err := jobRepo.RunMany(context.Background(), "group", provider,
			[]int64{1}, []bool{true}, []string{"https://example.com/a.git"}, "")
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("scheduler failure", func(t *testing.T) {
		t.Parallel()

		jobRepo := &JobRepository{
			encrypter: svaEncrypter{},
			scheduler: svaNewScheduler(t, &svaJobStore{err: errSvaBoom}),
		}
		err := jobRepo.RunMany(context.Background(), "group", provider,
			[]int64{1}, []bool{true}, []string{"https://example.com/a.git"}, "")
		require.ErrorIs(t, err, errSvaBoom)
	})
}

func TestSvaJobRepositoryGetProgress(t *testing.T) {
	t.Parallel()

	running := &job.Job{UID: "import-repo-1", State: job.JobStateRunning, RunProgress: 40}

	tests := []struct {
		name      string
		jobs      map[string]*job.Job
		storeErr  error
		repoState enum.RepoState
		wantState job.State
		wantErr   error
	}{
		{name: "existing job", jobs: map[string]*job.Job{"import-repo-1": running}, wantState: job.JobStateRunning},
		{name: "missing job of an importing repo", repoState: enum.RepoStateGitImport, wantState: job.JobStateFailed},
		{name: "missing job of a failed import", repoState: enum.RepoStateImportFailed, wantState: job.JobStateFailed},
		{name: "missing job of a migrate import", repoState: enum.RepoStateMigrateDataImport, wantState: job.JobStateRunning},
		{name: "missing job of a migrate push", repoState: enum.RepoStateMigrateGitPush, wantState: job.JobStateRunning},
		{name: "no import ever", repoState: enum.RepoStateActive, wantErr: ErrNotFound},
		{name: "store failure", storeErr: errSvaBoom, wantErr: errSvaBoom},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			jobRepo := &JobRepository{scheduler: svaNewScheduler(t, &svaJobStore{jobs: test.jobs, err: test.storeErr})}

			progress, err := jobRepo.GetProgress(context.Background(), &types.RepositoryCore{ID: 1, State: test.repoState})
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantState, progress.State)
		})
	}
}

func TestSvaJobRepositoryCancelIgnoresFinishedImports(t *testing.T) {
	t.Parallel()

	// A repository that is not importing has nothing to cancel, so the scheduler is never reached.
	jobRepo := &JobRepository{}
	require.NoError(t, jobRepo.Cancel(context.Background(), &types.Repository{ID: 1, State: enum.RepoStateActive}))
}

func TestSvaJobRepositoryProviderPassThrough(t *testing.T) {
	t.Parallel()

	jobRepo := &JobRepository{importer: svaNewImporter(&svaGit{})}

	_, _, err := jobRepo.LoadRepositoryFromProvider(
		context.Background(), Provider{Type: ProviderTypeGitHub}, "no-slash")
	require.Error(t, err)

	_, _, err = jobRepo.LoadRepositoriesFromProviderSpace(
		context.Background(), Provider{Type: ProviderTypeGitHub}, "", false)
	require.Error(t, err)
}

func TestSvaJobRegistration(t *testing.T) {
	t.Parallel()

	executor := job.NewExecutor(nil, nil)

	require.NoError(t, (&JobRepository{}).Register(executor))
	require.NoError(t, (&JobRepositoryLink{}).Register(executor))
	require.NoError(t, (&JobReferenceSync{}).Register(executor))

	// registering the same type a second time must be rejected.
	require.Error(t, (&JobRepository{}).Register(executor))
	require.Error(t, (&JobRepositoryLink{}).Register(executor))
	require.Error(t, (&JobReferenceSync{}).Register(executor))
}

func TestSvaJobRepositoryLinkDefinition(t *testing.T) {
	t.Parallel()

	link := &JobRepositoryLink{}
	require.Equal(t, "link-repo-12", link.jobIDFromRepoID(12))

	def, err := link.getJobDef("link-repo-12", JobLinkRepoInput{RepoID: 12, IsPublic: true})
	require.NoError(t, err)
	require.Equal(t, jobTypeRepositoryLink, def.Type)
	require.Equal(t, jobLinkRepoMaxDuration, def.Timeout)
	require.NoError(t, def.Validate())

	input, err := link.getJobInput(def.Data)
	require.NoError(t, err)
	require.Equal(t, JobLinkRepoInput{RepoID: 12, IsPublic: true}, input)

	_, err = link.getJobInput("***")
	require.ErrorContains(t, err, "base64")

	_, err = link.getJobInput(base64.StdEncoding.EncodeToString([]byte("nope")))
	require.ErrorContains(t, err, "unmarshal")

	_, err = link.Handle(context.Background(), "***", nil)
	require.Error(t, err)
}

func TestSvaJobRepositoryLinkRun(t *testing.T) {
	t.Parallel()

	jobStore := &svaJobStore{}
	link := &JobRepositoryLink{scheduler: svaNewScheduler(t, jobStore)}

	require.NoError(t, link.Run(context.Background(), 12, true))
	require.Len(t, jobStore.created, 1)
	require.Equal(t, "link-repo-12", jobStore.created[0].UID)

	failing := &JobRepositoryLink{scheduler: svaNewScheduler(t, &svaJobStore{err: errSvaBoom})}
	require.ErrorIs(t, failing.Run(context.Background(), 12, true), errSvaBoom)
}

func TestSvaJobReferenceSyncDefinition(t *testing.T) {
	t.Parallel()

	sync := &JobReferenceSync{}
	input := ReferenceSyncInput{
		SourceRepoID: 1, TargetRepoID: 2, RefSpecType: RefSpecTypeReference,
		SourceRef: "refs/heads/a", TargetRef: "refs/heads/b",
	}

	def, err := sync.getJobDef("uid", input)
	require.NoError(t, err)
	require.Equal(t, refSyncJobType, def.Type)
	require.Equal(t, refSyncJobMaxRetries, def.MaxRetries)
	require.NoError(t, def.Validate())

	got, err := sync.getJobInput(def.Data)
	require.NoError(t, err)
	require.Equal(t, input, got)

	_, err = sync.getJobInput("not json")
	require.ErrorContains(t, err, "unmarshal")
}

func TestSvaJobReferenceSyncRun(t *testing.T) {
	t.Parallel()

	jobStore := &svaJobStore{}
	sync := &JobReferenceSync{scheduler: svaNewScheduler(t, jobStore)}

	require.NoError(t, sync.Run(context.Background(), 1, 2, RefSpecTypeAllBranches, "", ""))
	require.Len(t, jobStore.created, 1)
	require.Equal(t, refSyncJobType, jobStore.created[0].Type)
	require.NotEmpty(t, jobStore.created[0].UID)

	got, err := sync.getJobInput(jobStore.created[0].Data)
	require.NoError(t, err)
	require.Equal(t, RefSpecTypeAllBranches, got.RefSpecType)

	failing := &JobReferenceSync{scheduler: svaNewScheduler(t, &svaJobStore{err: errSvaBoom})}
	require.ErrorIs(t, failing.Run(context.Background(), 1, 2, RefSpecTypeAllBranches, "", ""), errSvaBoom)
}

func TestSvaJobCreateRPCWriteParams(t *testing.T) {
	t.Parallel()

	principal := types.Principal{ID: 3, DisplayName: "System", Email: "system@example.com"}

	t.Run("reference sync", func(t *testing.T) {
		t.Parallel()

		sync := &JobReferenceSync{urlProvider: svaURLProvider{}}
		params, err := sync.createRPCWriteParams(context.Background(), principal, 7, "git-uid")
		require.NoError(t, err)
		require.Equal(t, "git-uid", params.RepoUID)
		require.Equal(t, "System", params.Actor.Name)
		require.Equal(t, "system@example.com", params.Actor.Email)
		require.NotEmpty(t, params.EnvVars)
	})

	t.Run("linked repositories sync", func(t *testing.T) {
		t.Parallel()

		syncLinked := NewJobSyncLinkedRepositories(
			"main", svaURLProvider{}, &svaGit{}, refcache.RepoFinder{}, nil, nil, nil, nil)
		params, err := syncLinked.createRPCWriteParams(context.Background(), principal, 7, "git-uid")
		require.NoError(t, err)
		require.Equal(t, "git-uid", params.RepoUID)
		require.Equal(t, "System", params.Actor.Name)
		require.NotEmpty(t, params.EnvVars)
	})
}
