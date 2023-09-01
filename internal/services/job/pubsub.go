// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package job

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"
)

const (
	PubSubTopicCancelJob   = "gitness:job:cancel_job"
	PubSubTopicStateChange = "gitness:job:state_change"
)

func encodeStateChange(job *types.Job) ([]byte, error) {
	stateChange := &types.JobStateChange{
		UID:      job.UID,
		State:    job.State,
		Progress: job.RunProgress,
		Result:   job.Result,
		Failure:  job.LastFailureError,
	}

	buffer := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buffer).Encode(stateChange); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func DecodeStateChange(payload []byte) (*types.JobStateChange, error) {
	stateChange := &types.JobStateChange{}
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(stateChange); err != nil {
		return nil, err
	}

	return stateChange, nil
}

func publishStateChange(ctx context.Context, publisher pubsub.Publisher, job *types.Job) error {
	payload, err := encodeStateChange(job)
	if err != nil {
		return fmt.Errorf("failed to gob encode JobStateChange: %w", err)
	}

	err = publisher.Publish(ctx, PubSubTopicStateChange, payload)
	if err != nil {
		return fmt.Errorf("failed to publish JobStateChange: %w", err)
	}

	return nil
}
