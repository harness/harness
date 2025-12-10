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

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/harness/gitness/job"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/types"
)

const JobType = "rpm_registry_index"

type JobRpmRegistryIndex struct {
	postProcessingReporter *registrypostprocessingevents.Reporter
}

func NewJobRpmRegistryIndex(
	postProcessingReporter *registrypostprocessingevents.Reporter,
	executor *job.Executor,
) (*JobRpmRegistryIndex, error) {
	j := JobRpmRegistryIndex{
		postProcessingReporter: postProcessingReporter,
	}
	err := executor.Register(JobType, &j)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (j *JobRpmRegistryIndex) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	input, err := j.getJobInput(data)
	if err != nil {
		return "", err
	}
	for _, id := range input.RegistryIDs {
		j.postProcessingReporter.BuildRegistryIndexWithPrincipal(
			ctx, id, make([]types.SourceRef, 0), 0,
		)
	}

	return "", nil
}

func (j *JobRpmRegistryIndex) getJobInput(data string) (RegistrySyncInput, error) {
	var input RegistrySyncInput

	err := json.NewDecoder(strings.NewReader(data)).Decode(&input)
	if err != nil {
		return RegistrySyncInput{}, fmt.Errorf("failed to unmarshal rpm registry sync job input json: %w", err)
	}

	return input, nil
}

type RegistrySyncInput struct {
	RegistryIDs []int64 `json:"registry_ids"`
}
