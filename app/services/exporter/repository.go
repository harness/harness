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

package exporter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// TODO: take as optional input from api input to allow exporting to SMP.
	harnessCodeAPIURLRaw = "https://app.harness.io/gateway/code/api"
)

var (
	// ErrNotFound is returned if no export data was found.
	ErrNotFound = errors.New("export not found")
)

type Repository struct {
	urlProvider  gitnessurl.Provider
	git          git.Interface
	repoStore    store.RepoStore
	scheduler    *job.Scheduler
	encrypter    encrypt.Encrypter
	sseStreamer  sse.Streamer
	publicAccess publicaccess.Service
}

type Input struct {
	Identifier      string          `json:"identifier"`
	ID              int64           `json:"id"`
	Description     string          `json:"description"`
	IsPublic        bool            `json:"is_public"`
	HarnessCodeInfo HarnessCodeInfo `json:"harness_code_info"`
}

type HarnessCodeInfo struct {
	AccountID         string `json:"account_id"`
	ProjectIdentifier string `json:"project_identifier"`
	OrgIdentifier     string `json:"org_identifier"`
	Token             string `json:"token"`
}

var _ job.Handler = (*Repository)(nil)

const (
	exportJobMaxRetries  = 1
	exportJobMaxDuration = 45 * time.Minute
	exportRepoJobUID     = "export_repo_%d"
	exportSpaceJobUID    = "export_space_%d"
	jobType              = "repository_export"
)

var ErrJobRunning = errors.New("an export job is already running")

func (r *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, r)
}

func (r *Repository) RunManyForSpace(
	ctx context.Context,
	spaceID int64,
	repos []*types.Repository,
	harnessCodeInfo *HarnessCodeInfo,
) error {
	jobGroupID := getJobGroupID(spaceID)
	jobs, err := r.scheduler.GetJobProgressForGroup(ctx, jobGroupID)
	if err != nil {
		return fmt.Errorf("cannot get job progress before starting. %w", err)
	}

	if len(jobs) > 0 {
		err = checkJobAlreadyRunning(jobs)
		if err != nil {
			return err
		}

		n, err := r.scheduler.PurgeJobsByGroupID(ctx, jobGroupID)
		if err != nil {
			return err
		}
		log.Ctx(ctx).Info().Msgf("deleted %d old jobs", n)
	}

	jobDefinitions := make([]job.Definition, len(repos))
	for i, repository := range repos {
		isPublic, err := r.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repository.Path)
		if err != nil {
			return fmt.Errorf("failed to check repo public access: %w", err)
		}

		repoJobData := Input{
			Identifier:      repository.Identifier,
			ID:              repository.ID,
			Description:     repository.Description,
			IsPublic:        isPublic,
			HarnessCodeInfo: *harnessCodeInfo,
		}

		data, err := json.Marshal(repoJobData)
		if err != nil {
			return fmt.Errorf("failed to marshal job input json: %w", err)
		}
		strData := strings.TrimSpace(string(data))
		encryptedData, err := r.encrypter.Encrypt(strData)
		if err != nil {
			return fmt.Errorf("failed to encrypt job input: %w", err)
		}

		jobUID := fmt.Sprintf(exportRepoJobUID, repository.ID)

		jobDefinitions[i] = job.Definition{
			UID:        jobUID,
			Type:       jobType,
			MaxRetries: exportJobMaxRetries,
			Timeout:    exportJobMaxDuration,
			Data:       base64.StdEncoding.EncodeToString(encryptedData),
		}
	}

	return r.scheduler.RunJobs(ctx, jobGroupID, jobDefinitions)
}

func checkJobAlreadyRunning(jobs []job.Progress) error {
	if jobs == nil {
		return nil
	}
	for _, j := range jobs {
		if !j.State.IsCompleted() {
			return ErrJobRunning
		}
	}
	return nil
}

func getJobGroupID(spaceID int64) string {
	return fmt.Sprintf(exportSpaceJobUID, spaceID)
}

// Handle is repository export background job handler.
func (r *Repository) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	input, err := r.getJobInput(data)
	if err != nil {
		return "", err
	}
	harnessCodeInfo := input.HarnessCodeInfo
	client, err := newHarnessCodeClient(
		harnessCodeAPIURLRaw,
		harnessCodeInfo.AccountID,
		harnessCodeInfo.OrgIdentifier,
		harnessCodeInfo.ProjectIdentifier,
		harnessCodeInfo.Token,
	)
	if err != nil {
		return "", err
	}

	repository, err := r.repoStore.Find(ctx, input.ID)
	if err != nil {
		return "", err
	}

	remoteRepo, err := client.CreateRepo(ctx, repo.CreateInput{
		Identifier:    repository.Identifier,
		DefaultBranch: repository.DefaultBranch,
		Description:   repository.Description,
		IsPublic:      false, // TODO: replace with publicaccess service response once deployed on HC.
		Readme:        false,
		License:       "",
		GitIgnore:     "",
	})
	if err != nil {
		r.publishSSE(ctx, repository)
		return "", err
	}

	urlWithToken, err := modifyURL(remoteRepo.GitURL, harnessCodeInfo.Token)
	if err != nil {
		return "", err
	}

	err = r.git.PushRemote(ctx, &git.PushRemoteParams{
		ReadParams: git.ReadParams{RepoUID: repository.GitUID},
		RemoteURL:  urlWithToken,
	})
	if err != nil && !strings.Contains(err.Error(), "empty") {
		errDelete := client.DeleteRepo(ctx, remoteRepo.Identifier)
		if errDelete != nil {
			log.Ctx(ctx).Err(errDelete).Msgf("failed to delete repo '%s' on harness", remoteRepo.Identifier)
		}
		r.publishSSE(ctx, repository)
		return "", err
	}

	log.Ctx(ctx).Info().Msgf("completed exporting repository '%s' to harness", repository.Identifier)

	r.publishSSE(ctx, repository)

	return "", nil
}

func (r *Repository) publishSSE(ctx context.Context, repository *types.Repository) {
	err := r.sseStreamer.Publish(ctx, repository.ParentID, enum.SSETypeRepositoryExportCompleted, repository)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish export completion SSE")
	}
}

func (r *Repository) getJobInput(data string) (Input, error) {
	encrypted, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return Input{}, fmt.Errorf("failed to base64 decode job input: %w", err)
	}

	decrypted, err := r.encrypter.Decrypt(encrypted)
	if err != nil {
		return Input{}, fmt.Errorf("failed to decrypt job input: %w", err)
	}

	var input Input

	err = json.NewDecoder(strings.NewReader(decrypted)).Decode(&input)
	if err != nil {
		return Input{}, fmt.Errorf("failed to unmarshal job input json: %w", err)
	}

	return input, nil
}

func (r *Repository) GetProgressForSpace(ctx context.Context, spaceID int64) ([]job.Progress, error) {
	groupID := getJobGroupID(spaceID)
	progress, err := r.scheduler.GetJobProgressForGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job progress for group: %w", err)
	}

	if len(progress) == 0 {
		return nil, ErrNotFound
	}

	return progress, nil
}

func modifyURL(u string, token string) (string, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL '%s': %w", u, err)
	}

	// Set the username and password in the URL
	parsedURL.User = url.UserPassword("token", token)
	return parsedURL.String(), nil
}
