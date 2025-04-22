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

	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideMediator,
)

func ProvideMediator(
	ctx context.Context,
	config *types.Config,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	metricsStore store.UsageMetricStore,
	repoEvReaderFactory *events.ReaderFactory[*repoevents.Reader],
) (Sender, error) {
	if !config.UsageMetrics.Enabled {
		return &Noop{}, nil
	}

	m := newMediator(
		ctx,
		spaceFinder,
		metricsStore,
		NewConfig(config),
	)

	if err := registerEventListeners(ctx, config.InstanceID, m, repoEvReaderFactory, repoFinder); err != nil {
		return nil, fmt.Errorf("failed to register event listeners: %w", err)
	}

	return m, nil
}
