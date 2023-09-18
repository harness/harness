// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package exporter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/sse"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
	"net/url"
	"strings"
	"time"

	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/sse"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/store"
	gitnessurl "github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
)

var (
	// ErrNotFound is returned if no export data was found.
	ErrNotFound = errors.New("export not found")
)

type Repository struct {
	urlProvider *gitnessurl.Provider
	git         gitrpc.Interface
	repoStore   store.RepoStore
	scheduler   *job.Scheduler
	encrypter   encrypt.Encrypter
	sseStreamer sse.Streamer
}

type Input struct {
	UID             string          `json:"uid"`
	ID              int64           `json:"id"`
	Description     string          `json:"description"`
	IsPublic        bool            `json:"is_public"`
	HarnessCodeInfo HarnessCodeInfo `json:"harness_code_info"`
}

type HarnessCodeInfo struct {
	AccountId         string `json:"account_id"`
	ProjectIdentifier string `json:"project_identifier"`
	OrgIdentifier     string `json:"org_identifier"`
	Token             string `json:"token"`
}

var _ job.Handler = (*Repository)(nil)

const (
	exportJobMaxRetries  = 1
	exportJobMaxDuration = 45 * time.Minute
	exportRepoJobUid     = "export_repo_%d"
	exportSpaceJobUid    = "export_space_%d"
	jobType              = "repository_export"
)

var ErrJobRunning = errors.New("an export job is already running")


func (r *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, r)
}
}
	func (r *Repository) RunManyForSpace(
		ctx context.Context,
		spaceId int64,
		repos []*types.Repository,
		harnessCodeInfo *HarnessCodeInfo,
) error {		jobGroupId := getJobGroupId(spaceId)

		jobs, err := r.scheduler.GetJobProgressForGroup(ctx, jobGroupId)
		if err != nil {
		return fmt.Errorf("cannot get job progress before starting. %w", err)
	}

		err = checkJobAlreadyRunning(jobs)
		if err != nil {
		return err
	}

		n, err := r.scheduler.PurgeJobsByGroupId(ctx, jobGroupId)
		if err != nil {
		return err
	}
		log.Ctx(ctx).Info().Msgf("deleted %d old jobs", n)
		jobDefinitions := make([]job.Definition, len(repos))
	for i, repository := range repos {
		repoJobData := Input{
			UID:             repository.UID,
			ID:              repository.ID,
			Description:     repository.Description,
			IsPublic:        repository.IsPublic,
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

		jobUID := fmt.Sprintf(exportRepoJobUid, repository.ID)

		jobDefinitions[i] = job.Definition{
			UID:        jobUID,
			Type:       jobType,
			MaxRetries: exportJobMaxRetries,
			Timeout:    exportJobMaxDuration,
			Data:       base64.StdEncoding.EncodeToString(encryptedData),
		}
	}

	return r.scheduler.RunJobs(ctx, jobGroupId, jobDefinitions)
}

func checkJobAlreadyRunning(jobs []types.JobProgress) error {
	if jobs != nil {
		for _, j := range jobs {
			if !slices.Contains(enum.GetCompletedJobState(), j.State) {
				return ErrJobRunning
			}
		}
	}
	return nil
}

func getJobGroupId(spaceId int64) string {
	return fmt.Sprintf(exportSpaceJobUid, spaceId)
}

// Handle is repository export background job handler.
func (r *Repository) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	input, err := r.getJobInput(data)
	if err != nil {
		return "", err
	}
	harnessCodeInfo := input.HarnessCodeInfo
	client, err := newHarnessCodeClient(r.urlProvider.GetHarnessCodeInternalUrl(), harnessCodeInfo.AccountId, harnessCodeInfo.OrgIdentifier, harnessCodeInfo.ProjectIdentifier, harnessCodeInfo.Token)
	if err != nil {
		return "", err
	}

	repository, err := r.repoStore.Find(ctx, input.ID)
	if err != nil {
		return "", err
	}
	remoteRepo, err := client.CreateRepo(ctx, repo.CreateInput{
		UID:           repository.UID,
		DefaultBranch: repository.DefaultBranch,
		Description:   repository.Description,
		IsPublic:      repository.IsPublic,
		Readme:        false,
		License:       "",
		GitIgnore:     "",
	})
	if err != nil {
		r.publishSSE(ctx, repository)
		return "", err
	}

	urlWithToken, err := modifyUrl(remoteRepo.GitURL, harnessCodeInfo.Token)
	if err != nil {
		return "", err
	}

	err = r.git.PushRemote(ctx, &gitrpc.PushRemoteParams{
		ReadParams: gitrpc.ReadParams{RepoUID: repository.GitUID},
		RemoteUrl:  urlWithToken,
	})
	if err != nil && !strings.Contains(err.Error(), "empty") {
		errDelete := client.DeleteRepo(ctx, remoteRepo.UID)
		if errDelete != nil {
			log.Ctx(ctx).Err(errDelete).Msgf("failed to delete repo '%s' on harness", remoteRepo.UID)
		}
		r.publishSSE(ctx, repository)
		return "", err
	}

	log.Ctx(ctx).Info().Msgf("completed exporting repository '%s' to harness", repository.UID)

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

func (r *Repository) GetProgressForSpace(ctx context.Context, spaceID int64) ([]types.JobProgress, error) {
	spaceId := getJobGroupId(spaceID)
	progress, err := r.scheduler.GetJobProgressForGroup(ctx, spaceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get job progress for group: %w", err)
	}

	if len(progress) == 0 {
		return nil, ErrNotFound
	}

	return progress, nil
}

func modifyUrl(u string, token string) (string, error) {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return "", err
	}

	// Set the username and password in the URL
	parsedUrl.User = url.UserPassword("token", token)
	return parsedUrl.String(), nil
}
