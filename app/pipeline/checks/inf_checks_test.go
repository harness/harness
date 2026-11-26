// Copyright 2026 Harness, Inc.
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
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type infCheckStore struct {
	store.CheckStore

	upserted *types.Check
	err      error
}

func (s *infCheckStore) Upsert(_ context.Context, check *types.Check) error {
	if s.err != nil {
		return s.err
	}
	s.upserted = check
	return nil
}

func TestInfWrite(t *testing.T) {
	ctx := context.Background()
	execution := &types.Execution{
		Number:     7,
		RepoID:     3,
		PipelineID: 5,
		Status:     enum.CIStatusRunning,
		Title:      "title",
		Started:    10,
		Finished:   20,
		After:      "abc123",
		CreatedBy:  9,
	}
	pipeline := &types.Pipeline{Identifier: "build"}

	cs := &infCheckStore{}
	require.NoError(t, Write(ctx, cs, execution, pipeline))
	require.NotNil(t, cs.upserted)
	require.Equal(t, int64(3), cs.upserted.RepoID)
	require.Equal(t, "abc123", cs.upserted.CommitSHA)
	require.Equal(t, enum.CheckPayloadKindPipeline, cs.upserted.Payload.Kind)
	require.Equal(t, "1", cs.upserted.Payload.Version)
	require.Equal(t, "build", cs.upserted.Identifier)
	require.Equal(t, "build", cs.upserted.Summary, "identifier is the summary fallback")
	require.Equal(t, int64(9), cs.upserted.CreatedBy)
	require.Equal(t, enum.CIStatusRunning.ConvertToCheckStatus(), cs.upserted.Status)
	require.JSONEq(t, "{}", string(cs.upserted.Metadata))

	pipeline.Description = "a description"
	require.NoError(t, Write(ctx, cs, execution, pipeline))
	require.Equal(t, "a description", cs.upserted.Summary)

	var payload types.CheckPayloadInternal
	require.NoError(t, json.Unmarshal(cs.upserted.Payload.Data, &payload))
	require.Equal(t, types.CheckPayloadInternal{Number: 7, RepoID: 3, PipelineID: 5}, payload)

	cs.err = errors.New("db down")
	require.ErrorContains(t, Write(ctx, cs, execution, pipeline), "could not upsert to check store")
}
