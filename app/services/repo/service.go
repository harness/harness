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

package repo

import (
	"context"
	"fmt"
	"time"

	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
)

const groupRepo = "gitness:repo"

type Service struct {
	repoEvReporter *repoevents.Reporter
	repoStore      store.RepoStore
	urlProvider    url.Provider
	git            git.Interface
	locker         *locker.Locker
}

func NewService(
	ctx context.Context,
	config *types.Config,
	repoEvReporter *repoevents.Reporter,
	repoReaderFactory *events.ReaderFactory[*repoevents.Reader],
	repoStore store.RepoStore,
	urlProvider url.Provider,
	git git.Interface,
	locker *locker.Locker,
) (*Service, error) {
	service := &Service{
		repoEvReporter: repoEvReporter,
		repoStore:      repoStore,
		urlProvider:    urlProvider,
		git:            git,
		locker:         locker,
	}

	_, err := repoReaderFactory.Launch(ctx, groupRepo, config.InstanceID,
		func(r *repoevents.Reader) error {
			const idleTimeout = 15 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterDefaultBranchUpdated(service.handleUpdateDefaultBranch)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch reader factory for repo git group: %w", err)
	}

	return service, nil
}
