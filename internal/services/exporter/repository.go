// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package exporter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/harness/gitness/encrypt"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/store"
	gitnessurl "github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
)

type Repository struct {
	urlProvider *gitnessurl.Provider
	git         gitrpc.Interface
	repoStore   store.RepoStore
	scheduler   *job.Scheduler
	encrypter   encrypt.Encrypter
}

type Input struct {
	UID             string          `json:"uid"`
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
)

const jobType = "repository_export"

func (e *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, e)
}

func (e *Repository) RunMany(ctx context.Context, spaceId int64, harnessCodeInfo *HarnessCodeInfo, repos []*types.Repository) error {
	jobGroupId := fmt.Sprintf(exportSpaceJobUid, spaceId)
	jobDefinitions := make([]job.Definition, len(repos))
	for i, repo := range repos {
		repoJobData := Input{
			UID:             repo.UID,
			Description:     repo.Description,
			IsPublic:        repo.IsPublic,
			HarnessCodeInfo: *harnessCodeInfo,
		}

		data, err := json.Marshal(repoJobData)
		if err != nil {
			return fmt.Errorf("failed to marshal job input json: %w", err)
		}
		strData := strings.TrimSpace(string(data))
		encryptedData, err := e.encrypter.Encrypt(strData)
		if err != nil {
			return fmt.Errorf("failed to encrypt job input: %w", err)
		}

		jobUID := fmt.Sprintf(exportRepoJobUid, repo.ID)

		jobDefinitions[i] = job.Definition{
			UID:        jobUID,
			Type:       jobType,
			MaxRetries: exportJobMaxRetries,
			Timeout:    exportJobMaxDuration,
			Data:       base64.StdEncoding.EncodeToString(encryptedData),
		}
	}

	return e.scheduler.RunJobs(ctx, jobGroupId, jobDefinitions)
}

// Handle is repository export background job handler.
func (e *Repository) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	// create repo via api call and then do git push
	return "", nil
}

func (e *Repository) GetProgress(ctx context.Context, repo *types.Space) (types.JobProgress, error) {
	// todo(abhinav): implement
	return types.JobProgress{}, nil
}
