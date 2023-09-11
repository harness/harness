// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package exporter

import (
	"context"
	"encoding/json"
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

var _ job.Handler = (*Repository)(nil)

type Input struct {
}

const jobType = "repository_export"

func (i *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, i)
}

func (i *Repository) Run(ctx context.Context, jobUID string, input Input) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	strData := strings.TrimSpace(string(data))

	return i.scheduler.RunJob(ctx, job.Definition{
		UID:        jobUID,
		Type:       jobType,
		MaxRetries: 1,
		Timeout:    30 * time.Minute,
		Data:       strData,
	})
}

// Handle is repository import background job handler.
func (i *Repository) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	return "", nil
}

func (i *Repository) GetProgress(ctx context.Context, repo *types.Repository) (types.JobProgress, error) {
	// todo(abhinav): implement
	return types.JobProgress{}, nil
}
