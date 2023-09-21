package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/store"
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
		summary = pipeline.UID
	}
	check := &types.Check{
		RepoID:    execution.RepoID,
		UID:       pipeline.UID,
		Summary:   summary,
		Created:   now,
		Updated:   now,
		CreatedBy: execution.CreatedBy,
		Status:    execution.Status.ConvertToCheckStatus(),
		CommitSHA: execution.After,
		Metadata:  []byte("{}"),
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
