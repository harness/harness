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

	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/events"
)

// handleLastGITPush writes the push time after the githook returns, so githa is not holding the repo lock.
func (s *Service) handleLastGITPush(
	ctx context.Context,
	event *events.Event[*repoevents.PushedPayload],
) error {
	ctx, cancel := context.WithTimeout(ctx, lastGITPushTimeout)
	defer cancel()

	err := s.repoStore.UpdateLastGITPush(ctx, event.Payload.RepoID, event.Timestamp.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to update last git push time for repo %d: %w", event.Payload.RepoID, err)
	}

	return nil
}
