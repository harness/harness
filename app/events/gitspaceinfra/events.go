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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// List all Gitspace Infra events below.

	GitspaceInfraEvent events.EventType = "gitspace_infra_event"
)

type (
	GitspaceInfraEventPayload struct {
		Infra types.Infrastructure `json:"infra,omitempty"`
		Type  enum.InfraEvent      `json:"type"`
	}
)

func (r *Reporter) EmitGitspaceInfraEvent(
	ctx context.Context,
	event events.EventType,
	payload *GitspaceInfraEventPayload,
) {
	if payload == nil {
		return
	}
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, event, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send %v event", event)
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported %v event with id '%s'", event, eventID)
}

func (r *Reader) RegisterGitspaceInfraEvent(
	fn events.HandlerFunc[*GitspaceInfraEventPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, GitspaceInfraEvent, fn, opts...)
}
