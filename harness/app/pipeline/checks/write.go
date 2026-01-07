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

package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Write is a util function which writes execution and pipeline state to the
// check store.
func Write(
	ctx context.Context,
	checkStore store.CheckStore,
	execution *types.Execution,
	pipeline *types.Pipeline,
) error {
	payload := types.CheckPayloadInternal{
		Number:     execution.Number,
		RepoID:     execution.RepoID,
		PipelineID: execution.PipelineID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal check payload: %w", err)
	}
	now := time.Now().UnixMilli()
	summary := pipeline.Description
	if summary == "" {
		summary = pipeline.Identifier
	}
	check := &types.Check{
		RepoID:     execution.RepoID,
		Identifier: pipeline.Identifier,
		Summary:    summary,
		Created:    now,
		Updated:    now,
		CreatedBy:  execution.CreatedBy,
		Status:     execution.Status.ConvertToCheckStatus(),
		CommitSHA:  execution.After,
		Metadata:   []byte("{}"),
		Payload: types.CheckPayload{
			Version: "1",
			Kind:    enum.CheckPayloadKindPipeline,
			Data:    data,
		},
	}
	err = checkStore.Upsert(ctx, check)
	if err != nil {
		return fmt.Errorf("could not upsert to check store: %w", err)
	}
	return nil
}
