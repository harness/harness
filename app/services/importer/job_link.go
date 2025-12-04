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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/job"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	jobLinkRepoMaxRetries  = 0
	jobLinkRepoMaxDuration = 45 * time.Minute
)

type JobRepositoryLink struct {
	scheduler *job.Scheduler
	encrypter encrypt.Encrypter
	importer  *Importer
}

var _ job.Handler = (*JobRepositoryLink)(nil)

type JobLinkRepoInput struct {
	Input
}

const jobTypeRepositoryLink = "link_repository_import"

func (r *JobRepositoryLink) Register(executor *job.Executor) error {
	return executor.Register(jobTypeRepositoryLink, r)
}

// Run starts a background job that imports the provided repository from the provided clone URL.
func (r *JobRepositoryLink) Run(
	ctx context.Context,
	provider Provider,
	repo *types.Repository,
	public bool,
	cloneURL string,
) error {
	jobID := r.jobIDFromRepoID(repo.ID)
	jobDef, err := r.getJobDef(jobID, JobLinkRepoInput{
		Input: Input{
			RepoID:    repo.ID,
			Public:    public,
			GitUser:   provider.Username,
			GitPass:   provider.Password,
			CloneURL:  cloneURL,
			Pipelines: PipelineOptionIgnore,
		},
	})
	if err != nil {
		return err
	}

	return r.scheduler.RunJob(ctx, jobDef)
}

func (*JobRepositoryLink) jobIDFromRepoID(repoID int64) string {
	const jobIDPrefix = "link-repo-"
	return jobIDPrefix + strconv.FormatInt(repoID, 10)
}

func (r *JobRepositoryLink) getJobDef(jobUID string, input JobLinkRepoInput) (job.Definition, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to marshal job input json: %w", err)
	}

	strData := strings.TrimSpace(string(data))

	encryptedData, err := r.encrypter.Encrypt(strData)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to encrypt job input: %w", err)
	}

	return job.Definition{
		UID:        jobUID,
		Type:       jobTypeRepositoryLink,
		MaxRetries: jobLinkRepoMaxRetries,
		Timeout:    jobLinkRepoMaxDuration,
		Data:       base64.StdEncoding.EncodeToString(encryptedData),
	}, nil
}

func (r *JobRepositoryLink) getJobInput(data string) (JobLinkRepoInput, error) {
	encrypted, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return JobLinkRepoInput{}, fmt.Errorf("failed to base64 decode job input: %w", err)
	}

	decrypted, err := r.encrypter.Decrypt(encrypted)
	if err != nil {
		return JobLinkRepoInput{}, fmt.Errorf("failed to decrypt job input: %w", err)
	}

	var input JobLinkRepoInput

	err = json.NewDecoder(strings.NewReader(decrypted)).Decode(&input)
	if err != nil {
		return JobLinkRepoInput{}, fmt.Errorf("failed to unmarshal job input json: %w", err)
	}

	return input, nil
}

// Handle is repository import background job handler.
//
//nolint:gocognit // refactor if needed.
func (r *JobRepositoryLink) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	input, err := r.getJobInput(data)
	if err != nil {
		return "", err
	}

	err = r.importer.Import(ctx, input.Input)
	if err != nil {
		return "", fmt.Errorf("failed to import repository: %w", err)
	}

	return "", nil
}

func (r *JobRepositoryLink) GetProgress(ctx context.Context, repo *types.RepositoryCore) (job.Progress, error) {
	progress, err := r.scheduler.GetJobProgress(ctx, r.jobIDFromRepoID(repo.ID))
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		if repo.State == enum.RepoStateGitImport {
			// if the job is not found but repo is marked as importing, return state=failed
			return job.FailProgress(), nil
		}

		// if repo is importing through the migrator cli there is no job created for it, return state=progress
		if repo.State == enum.RepoStateMigrateDataImport ||
			repo.State == enum.RepoStateMigrateGitPush {
			return job.Progress{
				State:    job.JobStateRunning,
				Progress: job.ProgressMin,
			}, nil
		}

		// otherwise there either was no import, or it completed a long time ago (job cleaned up by now)
		return job.Progress{}, ErrNotFound
	}
	if err != nil {
		return job.Progress{}, fmt.Errorf("failed to get job progress: %w", err)
	}

	return progress, nil
}

func (r *JobRepositoryLink) Cancel(ctx context.Context, repo *types.Repository) error {
	if repo.State != enum.RepoStateGitImport {
		return nil
	}

	err := r.scheduler.CancelJob(ctx, r.jobIDFromRepoID(repo.ID))
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	return nil
}
