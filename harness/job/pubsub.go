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

package job

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"github.com/harness/gitness/pubsub"
)

const (
	PubSubTopicCancelJob   = "gitness:job:cancel_job"
	PubSubTopicStateChange = "gitness:job:state_change"
)

func encodeStateChange(job *Job) ([]byte, error) {
	stateChange := &StateChange{
		UID:      job.UID,
		Type:     job.Type,
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

func DecodeStateChange(payload []byte) (*StateChange, error) {
	stateChange := &StateChange{}
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(stateChange); err != nil {
		return nil, err
	}

	return stateChange, nil
}

func publishStateChange(ctx context.Context, publisher pubsub.Publisher, job *Job) error {
	payload, err := encodeStateChange(job)
	if err != nil {
		return fmt.Errorf("failed to gob encode StateChange: %w", err)
	}

	err = publisher.Publish(ctx, PubSubTopicStateChange, payload)
	if err != nil {
		return fmt.Errorf("failed to publish StateChange: %w", err)
	}

	return nil
}
