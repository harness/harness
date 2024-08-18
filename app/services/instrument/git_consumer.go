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

package instrument

import (
	"context"
	"fmt"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

type Consumer struct {
	repoStore          store.RepoStore
	principalInfoCache store.PrincipalInfoCache
	instrumentation    Service
}

func NewConsumer(ctx context.Context,
	config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	repoStore store.RepoStore,
	principalInfoCache store.PrincipalInfoCache,
	instrumentation Service,
) (Consumer, error) {
	c := Consumer{
		repoStore:          repoStore,
		principalInfoCache: principalInfoCache,
		instrumentation:    instrumentation,
	}

	const groupCommitInstrument = "gitness:git:instrumentation"
	_, err := gitReaderFactory.Launch(ctx, groupCommitInstrument, config.InstanceID,
		func(r *gitevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(3),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			_ = r.RegisterBranchUpdated(c.instrumentTrackOnBranchUpdate)

			return nil
		})
	if err != nil {
		return Consumer{}, fmt.Errorf("failed to launch git consumer: %w", err)
	}
	return c, nil
}

func (c Consumer) instrumentTrackOnBranchUpdate(
	ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	repo, err := c.repoStore.Find(ctx, event.Payload.RepoID)
	if err != nil {
		return fmt.Errorf("failed to get repo git info: %w", err)
	}

	principal, err := c.principalInfoCache.Get(ctx, event.Payload.PrincipalID)
	if err != nil {
		return fmt.Errorf("failed to get principal info: %w", err)
	}

	err = c.instrumentation.Track(ctx, Event{
		Type:      EventTypeCreateCommit,
		Principal: principal,
		Path:      repo.Path,
		Properties: map[Property]any{
			PropertyRepositoryID:    repo.ID,
			PropertyRepositoryName:  repo.Identifier,
			PropertyIsDefaultBranch: event.Payload.Ref == repo.DefaultBranch,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for create commit operation: %s", err)
	}
	return nil
}
