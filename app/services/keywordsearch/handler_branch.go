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

package keywordsearch

import (
	"context"
	"fmt"
	"strings"

	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/events"
)

func (s *Service) handleEventBranchCreated(ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload]) error {
	return s.indexRepo(ctx, event.Payload.RepoID, event.Payload.Ref)
}

func (s *Service) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload]) error {
	return s.indexRepo(ctx, event.Payload.RepoID, event.Payload.Ref)
}

func (s *Service) handleUpdateDefaultBranch(ctx context.Context,
	event *events.Event[*repoevents.DefaultBranchUpdatedPayload]) error {
	repo, err := s.repoStore.Find(ctx, event.Payload.RepoID)
	if err != nil {
		return fmt.Errorf("failed to find repository in db: %w", err)
	}

	err = s.indexer.Index(ctx, repo)
	if err != nil {
		return fmt.Errorf("index update failed for repo %d: %w", repo.ID, err)
	}

	return nil
}

func (s *Service) indexRepo(
	ctx context.Context,
	repoID int64,
	ref string,
) error {
	repo, err := s.repoStore.Find(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repository in db: %w", err)
	}

	branch, err := getBranchFromRef(ref)
	if err != nil {
		return events.NewDiscardEventError(
			fmt.Errorf("failed to parse branch name from ref: %w", err))
	}

	// we only maintain the index on the default branch
	if repo.DefaultBranch != branch {
		return nil
	}

	err = s.indexer.Index(ctx, repo)
	if err != nil {
		return fmt.Errorf("index update failed for repo %d: %w", repo.ID, err)
	}

	return nil
}

func getBranchFromRef(ref string) (string, error) {
	const refPrefix = "refs/heads/"

	if !strings.HasPrefix(ref, refPrefix) {
		return "", fmt.Errorf("failed to get branch name from branch ref %s", ref)
	}

	branch := ref[len(refPrefix):]
	if len(branch) == 0 {
		return "", fmt.Errorf("got an empty branch name from branch ref %s", ref)
	}
	return branch, nil
}
