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

package space

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/job"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	moveJobMaxRetries  = 3
	moveJobMaxDuration = 10 * time.Minute
	jobType            = "space_move"
)

var _ job.Handler = (*Service)(nil)

type Input struct {
	SourceSpacePath      string `json:"source_space_path"`
	DestinationSpacePath string `json:"destination_space_path"`
}

func (s *Service) Register(executor *job.Executor) error {
	return executor.Register(jobType, s)
}

func (s *Service) Run(
	ctx context.Context,
	srcIdentifier string,
	dstIdentifier string,
) error {
	jobDef, err := s.getJobDef(s.JobUIDFromSpacePath(srcIdentifier), Input{
		SourceSpacePath:      srcIdentifier,
		DestinationSpacePath: dstIdentifier,
	})
	if err != nil {
		return err
	}

	return s.scheduler.RunJob(ctx, jobDef)
}

// Handle is space move background job handler.
func (s *Service) Handle(
	ctx context.Context,
	data string,
	_ job.ProgressReporter,
) (string, error) {
	input, err := s.getJobInput(data)
	if err != nil {
		return "", err
	}

	if input.SourceSpacePath == "" {
		return "", fmt.Errorf("source space path is required")
	}

	if input.DestinationSpacePath == "" {
		return "", fmt.Errorf("destination space path is required")
	}

	log.Ctx(ctx).Debug().Msgf("space move job started for source space '%s' to destination space '%s'",
		input.SourceSpacePath, input.DestinationSpacePath)

	srcSpace, err := s.spaceStore.FindByRef(ctx, input.SourceSpacePath)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		log.Ctx(ctx).Info().Str("space.path", input.SourceSpacePath).
			Msg("source space not found, nothing to move")
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to find source space '%s': %w", input.SourceSpacePath, err)
	}

	dstSpace, err := s.spaceStore.FindByRef(ctx, input.DestinationSpacePath)
	// if dstSpace doesn't exist, update the srcSpace parent to match the dstSpace path
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		parentSpace, _, err := paths.DisectLeaf(input.DestinationSpacePath)
		if err != nil {
			return "", fmt.Errorf("failed to disect destination space path: %w", err)
		}

		log.Ctx(ctx).Info().Msgf("moving space %s by updating the parent space to %s", srcSpace.Identifier, parentSpace)
		err = s.MoveNoAuth(
			ctx,
			bootstrap.NewSystemServiceSession(),
			srcSpace,
			nil,
			parentSpace,
		)
		if err != nil {
			return "", fmt.Errorf("failed to move space: %w", err)
		}
		log.Ctx(ctx).Info().
			Msgf("space %s moved to %s", srcSpace.Identifier, parentSpace)

		s.spaceFinder.MarkChanged(ctx, srcSpace.Core())
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to find destination space for move: %w", err)
	}

	// when dstSpace exists, update the srcSpace resources parent to the dstSpace
	output, err := s.moveSpaceResourcesInTx(ctx, srcSpace, dstSpace)
	if err != nil {
		return "", fmt.Errorf("failed to move space resources: %w", err)
	}
	log.Ctx(ctx).Info().
		Int64("repo_count", output.RepoCount).
		Int64("label_count", output.LabelCount).
		Int64("rule_count", output.RuleCount).
		Int64("webhook_count", output.WebhookCount).
		Msgf("space resources moved from %s to %s",
			srcSpace.Identifier, dstSpace.Identifier)

	return "", nil
}

func (s *Service) MoveNoAuth(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
	inIdentifier *string,
	inParentRef string,
) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := s.spaceStore.FindForUpdate(ctx, space.ID)
		if err != nil {
			return fmt.Errorf("failed to lock the space for update: %w", err)
		}

		parentSpace, err := s.spaceStore.FindByRef(ctx, inParentRef)
		if err != nil {
			return fmt.Errorf("failed to find space by ID: %w", err)
		}

		// delete old primary segment
		err = s.spacePathStore.DeletePrimarySegment(ctx, space.ID)
		if err != nil {
			return fmt.Errorf("failed to delete primary path segment: %w", err)
		}

		// update space with move inputs
		if inIdentifier != nil {
			space.Identifier = *inIdentifier
		}

		space.ParentID = parentSpace.ID

		// add new primary segment using updated space data
		now := time.Now().UnixMilli()
		newPrimarySegment := &types.SpacePathSegment{
			ParentID:   parentSpace.ID,
			Identifier: space.Identifier,
			SpaceID:    space.ID,
			IsPrimary:  true,
			CreatedBy:  session.Principal.ID,
			Created:    now,
			Updated:    now,
		}
		err = s.spacePathStore.InsertSegment(ctx, newPrimarySegment)
		if err != nil {
			return fmt.Errorf("failed to create new primary path segment: %w", err)
		}

		if err := s.cleanUpStaleSpaceResources(ctx, space); err != nil {
			return fmt.Errorf("failed to clean up stale space resources: %w", err)
		}

		// update space itself
		err = s.spaceStore.Update(ctx, space)
		if err != nil {
			return fmt.Errorf("failed to update the space in the db: %w", err)
		}

		return nil
	})
}

type MoveResourcesOutput struct {
	RepoCount    int64 `json:"repo_count"`
	LabelCount   int64 `json:"label_count"`
	RuleCount    int64 `json:"rule_count"`
	WebhookCount int64 `json:"webhook_count"`
}

// MoveResources moves space resources to a new parent space individually and soft delete the source space.
func (s *Service) moveSpaceResourcesInTx(
	ctx context.Context,
	sourceSpace *types.Space,
	targetSpace *types.Space,
) (MoveResourcesOutput, error) {
	log.Ctx(ctx).Info().
		Msgf("moving space resources individually as target space %s exists", targetSpace.Identifier)

	var output MoveResourcesOutput
	if sourceSpace.ID == targetSpace.ID {
		return output, fmt.Errorf("source and target spaces cannot be the same")
	}

	if err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		_, err = s.spaceStore.FindForUpdate(ctx, sourceSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to lock the space for update: %w", err)
		}

		_, err = s.spaceStore.FindForUpdate(ctx, targetSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to lock the space for update: %w", err)
		}

		output.RepoCount, err = s.repoStore.UpdateParent(ctx, sourceSpace.ID, targetSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to move repos: %w", err)
		}

		output.LabelCount, err = s.labelStore.UpdateParentSpace(ctx, sourceSpace.ID, targetSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to update labels: %w", err)
		}

		output.RuleCount, err = s.rulesStore.UpdateParentSpace(ctx, sourceSpace.ID, targetSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to update rules: %w", err)
		}

		output.WebhookCount, err = s.webhookStore.UpdateParentSpace(ctx, sourceSpace.ID, targetSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to update webhooks: %w", err)
		}

		if err := s.cleanUpStaleSpaceResources(ctx, sourceSpace); err != nil {
			return fmt.Errorf("failed to clean up parent space resources: %w", err)
		}

		if err := s.SoftDeleteInner(
			ctx,
			bootstrap.NewSystemServiceSession(),
			sourceSpace,
			time.Now().Unix(),
		); err != nil {
			return fmt.Errorf("failed to soft delete source space: %w", err)
		}

		return nil
	}); err != nil {
		return output, err
	}

	s.spaceFinder.MarkChanged(ctx, sourceSpace.Core())
	s.repoFinder.Flush(ctx)

	return output, nil
}

// cleanUpStaleSpaceResources removes the resources of the parent space that will be moved.
func (s *Service) cleanUpStaleSpaceResources(ctx context.Context, space *types.Space) error {
	ancestors, err := s.spaceStore.GetAncestors(ctx, space.ID)
	if err != nil {
		return fmt.Errorf("failed to get ancestors: %w", err)
	}

	// exclude the root space from cleanup
	rootSpace, err := s.spaceStore.GetRootSpace(ctx, space.ID)
	if err != nil {
		return fmt.Errorf("failed to get root space: %w", err)
	}

	descendantSpaceIDs, err := s.spaceStore.GetDescendantsIDs(ctx, space.ID)
	if err != nil {
		return fmt.Errorf("failed to get descendant space IDs: %w", err)
	}

	descendantSpaceIDs = append(descendantSpaceIDs, space.ID)

	descendantSpaceIDSet := make(map[int64]struct{}, len(descendantSpaceIDs))
	for _, id := range descendantSpaceIDs {
		descendantSpaceIDSet[id] = struct{}{}
	}

	for _, ancestor := range ancestors {
		if ancestor.ID == rootSpace.ID || ancestor.ID == space.ID {
			continue
		}

		rules, err := s.rulesStore.List(ctx, []types.RuleParentInfo{
			{
				Type: enum.RuleParentSpace,
				ID:   ancestor.ID,
			},
		}, &types.RuleFilter{})
		if err != nil {
			return fmt.Errorf("failed to list rules for space %d: %w", ancestor.ID, err)
		}

		for _, rule := range rules {
			modified, err := s.cleanUpRuleRepoTargets(ctx, &rule, descendantSpaceIDSet)
			if err != nil {
				return fmt.Errorf("failed to clean up rule %d: %w", rule.ID, err)
			}

			if modified {
				log.Ctx(ctx).Info().Msgf("cleaning up rule %d target repos due to moving space %d", rule.ID, space.ID)
				if err := s.rulesStore.Update(ctx, &rule); err != nil {
					return fmt.Errorf("failed to update rule %d: %w", rule.ID, err)
				}
				log.Ctx(ctx).Info().Msgf("updated rule %d target repos due to moving space %d", rule.ID, space.ID)
			}
		}
	}

	return nil
}

// cleanUpRuleRepoTargets removes repository IDs from a rule's RepoTarget if they belong to descendant spaces.
func (s *Service) cleanUpRuleRepoTargets(
	ctx context.Context,
	rule *types.Rule,
	descendantSpaceIDSet map[int64]struct{},
) (bool, error) {
	if len(rule.RepoTarget) == 0 {
		return false, nil
	}

	var repoTarget protection.RepoTarget
	if err := json.Unmarshal(rule.RepoTarget, &repoTarget); err != nil {
		return false, fmt.Errorf("failed to unmarshal repo target: %w", err)
	}

	modified := false

	if len(repoTarget.Include.IDs) > 0 {
		filteredIncludeIDs, err := s.filterRepoIDs(ctx, repoTarget.Include.IDs, descendantSpaceIDSet)
		if err != nil {
			return false, fmt.Errorf("failed to filter include repo IDs: %w", err)
		}
		if len(filteredIncludeIDs) != len(repoTarget.Include.IDs) {
			repoTarget.Include.IDs = filteredIncludeIDs
			modified = true
		}
	}

	if len(repoTarget.Exclude.IDs) > 0 {
		filteredExcludeIDs, err := s.filterRepoIDs(ctx, repoTarget.Exclude.IDs, descendantSpaceIDSet)
		if err != nil {
			return false, fmt.Errorf("failed to filter exclude repo IDs: %w", err)
		}
		if len(filteredExcludeIDs) != len(repoTarget.Exclude.IDs) {
			repoTarget.Exclude.IDs = filteredExcludeIDs
			modified = true
		}
	}

	if modified {
		newRepoTarget, err := json.Marshal(repoTarget)
		if err != nil {
			return false, fmt.Errorf("failed to marshal updated repo target: %w", err)
		}
		rule.RepoTarget = newRepoTarget
	}

	return modified, nil
}

// filterRepoIDs filters out repository IDs that belong to descendant spaces.
func (s *Service) filterRepoIDs(
	ctx context.Context,
	repoIDs []int64,
	descendantSpaceIDSet map[int64]struct{},
) ([]int64, error) {
	filtered := make([]int64, 0, len(repoIDs))

	for _, repoID := range repoIDs {
		repo, err := s.repoStore.Find(ctx, repoID)
		if err != nil {
			if errors.Is(err, gitness_store.ErrResourceNotFound) {
				continue
			}
			return nil, fmt.Errorf("failed to find repository %d: %w", repoID, err)
		}

		if _, isDescendant := descendantSpaceIDSet[repo.ParentID]; !isDescendant {
			filtered = append(filtered, repoID)
		}
	}

	return filtered, nil
}
