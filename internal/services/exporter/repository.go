// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package exporter

import (
	"context"
	"encoding/json"
	"fmt"
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
}

type RepoImportData struct {
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
)

const jobType = "repository_export"

func (e *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, e)
}

func (e *Repository) Run(ctx context.Context, jobGroupId string, harnessCodeInfo *HarnessCodeInfo, repos []*types.Repository) error {
	jobDefinitions := make([]job.Definition, len(repos))
	for i, repo := range repos {
		repoJobData := RepoImportData{
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
		jobUID, err := job.UID()

		jobDefinitions[i] = job.Definition{
			UID:        jobUID,
			Type:       jobType,
			MaxRetries: exportJobMaxRetries,
			Timeout:    exportJobMaxDuration,
			Data:       strData,
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
