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

package events

import (
	"context"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const ExecutedEvent events.EventType = "executed"

type ExecutedPayload struct {
	PipelineID   int64         `json:"pipeline_id"`
	RepoID       int64         `json:"repo_id"`
	ExecutionNum int64         `json:"execution_number"`
	Status       enum.CIStatus `json:"status"`
}

func (r *Reporter) Executed(ctx context.Context, payload *ExecutedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ExecutedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pipeline executed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pipeline executed event with id '%s'", eventID)
}

func (r *Reader) RegisterExecuted(fn events.HandlerFunc[*ExecutedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, ExecutedEvent, fn, opts...)
}
