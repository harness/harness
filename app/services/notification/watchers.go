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

package notification

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// listWatchers returns the users watching the pull request, skipping the provided principal IDs.
// Service accounts and services are never returned, as they can't receive email notifications.
func (s *Service) listWatchers(
	ctx context.Context,
	prID int64,
	skipIDs ...int64,
) ([]*types.PrincipalInfo, error) {
	watchers, err := s.pullReqWatcherStore.List(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to list watchers of pullReqID %d: %w", prID, err)
	}

	skip := make(map[int64]struct{}, len(skipIDs))
	for _, id := range skipIDs {
		skip[id] = struct{}{}
	}

	principals := make([]*types.PrincipalInfo, 0, len(watchers))
	for _, w := range watchers {
		info := w.Watcher
		// The principal info is resolved via cache - an unresolved watcher has no ID or type set.
		if info.ID != w.PrincipalID || info.Type != enum.PrincipalTypeUser {
			continue
		}
		if _, ok := skip[info.ID]; ok {
			continue
		}
		principals = append(principals, &info)
	}

	return principals, nil
}
