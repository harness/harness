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

package branch

import (
	"context"
	"fmt"
	"strings"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
)

func ExtractBranchName(ref string) string {
	return strings.TrimPrefix(ref, refsBranchPrefix)
}

func (s *Service) handleEventBranchCreated(
	ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload],
) error {
	branchName := ExtractBranchName(event.Payload.Ref)

	branchSHA, err := sha.New(event.Payload.SHA)
	if err != nil {
		return fmt.Errorf("branch sha format invalid: %w", err)
	}

	now := time.Now().UnixMilli()
	branch := &types.BranchTable{
		Name:      branchName,
		SHA:       branchSHA,
		CreatedBy: event.Payload.PrincipalID,
		Created:   now,
		UpdatedBy: event.Payload.PrincipalID,
		Updated:   now,
	}

	err = s.branchStore.Upsert(ctx, event.Payload.RepoID, branch)
	if err != nil {
		return fmt.Errorf("failed to create branch in database: %w", err)
	}

	return nil
}

// handleEventBranchUpdated handles the branch updated event.
func (s *Service) handleEventBranchUpdated(
	ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	branchName := ExtractBranchName(event.Payload.Ref)

	branchSHA, err := sha.New(event.Payload.NewSHA)
	if err != nil {
		return fmt.Errorf("branch sha format invalid: %w", err)
	}

	now := time.Now().UnixMilli()
	branch := &types.BranchTable{
		Name:      branchName,
		SHA:       branchSHA,
		CreatedBy: event.Payload.PrincipalID,
		Created:   now,
		UpdatedBy: event.Payload.PrincipalID,
		Updated:   now,
	}

	if err := s.branchStore.Upsert(ctx, event.Payload.RepoID, branch); err != nil {
		return fmt.Errorf("failed to upsert branch in database: %w", err)
	}
	return nil
}

// handleEventBranchDeleted handles the branch deleted event.
func (s *Service) handleEventBranchDeleted(
	ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload],
) error {
	branchName := ExtractBranchName(event.Payload.Ref)

	err := s.branchStore.Delete(
		ctx,
		event.Payload.RepoID,
		branchName,
	)
	if err != nil {
		return fmt.Errorf("failed to delete branch from database: %w", err)
	}

	return nil
}
