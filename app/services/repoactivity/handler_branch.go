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

package repoactivity

import (
	"context"
	"fmt"
	"strings"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/events"
	gitapi "github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func extractBranchName(ref string) string {
	return strings.TrimPrefix(ref, gitapi.BranchPrefix)
}

func (s *Service) handleEventBranchCreated(
	ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload],
) error {
	branchName := extractBranchName(event.Payload.Ref)

	err := s.repoActivityStore.Create(ctx, &types.RepoActivity{
		RepoID:      event.Payload.RepoID,
		PrincipalID: event.Payload.PrincipalID,
		Key:         event.ID,
		Type:        enum.RepoActivityTypeBranchCreated,
		Payload: &types.RepoActivityPayloadBranchCreated{
			Name: branchName,
			New:  event.Payload.SHA,
		},
		Timestamp: event.Timestamp.UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("failed to persist repository activity for branch creation: %w", err)
	}

	return nil
}

func (s *Service) handleEventBranchUpdated(
	ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	branchName := extractBranchName(event.Payload.Ref)

	if err := s.repoActivityStore.Create(ctx, &types.RepoActivity{
		RepoID:      event.Payload.RepoID,
		PrincipalID: event.Payload.PrincipalID,
		Key:         event.ID,
		Type:        enum.RepoActivityTypeBranchUpdated,
		Payload: &types.RepoActivityPayloadBranchUpdated{
			Name:   branchName,
			Old:    event.Payload.OldSHA,
			New:    event.Payload.NewSHA,
			Forced: event.Payload.Forced,
		},
		Timestamp: event.Timestamp.UnixMilli(),
	}); err != nil {
		return fmt.Errorf("failed to persist repository activity for branch update: %w", err)
	}

	return nil
}

func (s *Service) handleEventBranchDeleted(
	ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload],
) error {
	branchName := extractBranchName(event.Payload.Ref)

	err := s.repoActivityStore.Create(ctx, &types.RepoActivity{
		RepoID:      event.Payload.RepoID,
		PrincipalID: event.Payload.PrincipalID,
		Key:         event.ID,
		Type:        enum.RepoActivityTypeBranchDeleted,
		Payload: &types.RepoActivityPayloadBranchDeleted{
			Name: branchName,
			Old:  event.Payload.SHA,
		},
		Timestamp: event.Timestamp.UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("failed to persist repository activity for branch deletion: %w", err)
	}

	return nil
}
