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

package usage

import (
	"context"
	"fmt"
	"time"

	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
)

type RepoFinder interface {
	FindByID(ctx context.Context, id int64) (*types.RepositoryCore, error)
}

func RegisterEventListeners(
	ctx context.Context,
	instanceID string,
	sender Sender,
	repoEvReaderFactory *events.ReaderFactory[*repoevents.Reader],
	repoFinder RepoFinder,
) error {
	// repo events
	const groupRepo = "gitness:usage:repo"
	_, err := repoEvReaderFactory.Launch(ctx, groupRepo, instanceID,
		func(r *repoevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))
			_ = r.RegisterCreated(repoCreateHandler(sender, repoFinder))
			_ = r.RegisterPushed(repoPushHandler(sender, repoFinder))

			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to launch repo event reader: %w", err)
	}

	return nil
}

func repoCreateHandler(sender Sender, repoFinder RepoFinder) events.HandlerFunc[*repoevents.CreatedPayload] {
	return func(ctx context.Context, event *events.Event[*repoevents.CreatedPayload]) error {
		return sendRepoPushUsage(ctx, sender, repoFinder, event.Payload.RepoID)
	}
}

func repoPushHandler(sender Sender, repoFinder RepoFinder) events.HandlerFunc[*repoevents.PushedPayload] {
	return func(ctx context.Context, event *events.Event[*repoevents.PushedPayload]) error {
		return sendRepoPushUsage(ctx, sender, repoFinder, event.Payload.RepoID)
	}
}

func sendRepoPushUsage(ctx context.Context, sender Sender, repoFinder RepoFinder, repoID int64) error {
	repo, err := repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repo with id %d: %w", repoID, err)
	}

	rootSpace, _, err := paths.DisectRoot(repo.Path)
	if err != nil {
		return fmt.Errorf("failed to disect repo path %q: %w", repo.Path, err)
	}

	m := Metric{
		SpaceRef: rootSpace,
		Pushes:   1,
	}
	if err := sender.Send(ctx, m); err != nil {
		return fmt.Errorf("failed to send usage metric: %w", err)
	}

	return nil
}
