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

package webhook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Listreturns the webhooks from the provided scope.
func (s *Service) List(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	inherited bool,
	filter *types.WebhookFilter,
) ([]*types.Webhook, int64, error) {
	var parents []types.WebhookParentInfo
	var err error

	switch parentType {
	case enum.WebhookParentRepo:
		parents, err = s.getParentInfoRepo(ctx, parentID, inherited)
		if err != nil {
			return nil, 0, err
		}
	case enum.WebhookParentSpace:
		parents, err = s.getParentInfoSpace(ctx, parentID, inherited)
		if err != nil {
			return nil, 0, err
		}
	default:
		return nil, 0, fmt.Errorf("webhook type %s is not supported", parentType)
	}

	count, err := s.webhookStore.Count(ctx, parents, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count webhooks for scope with id %d: %w", parentID, err)
	}

	webhooks, err := s.webhookStore.List(ctx, parents, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list webhooks for scope with id %d: %w", parentID, err)
	}

	return webhooks, count, nil
}

func (s *Service) getParentInfoRepo(
	ctx context.Context,
	repoID int64,
	inherited bool,
) ([]types.WebhookParentInfo, error) {
	var parents []types.WebhookParentInfo

	parents = append(parents, types.WebhookParentInfo{
		ID:   repoID,
		Type: enum.WebhookParentRepo,
	})

	if inherited {
		repo, err := s.repoStore.Find(ctx, repoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get repo: %w", err)
		}

		ids, err := s.spaceStore.GetAncestorIDs(ctx, repo.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent space ids: %w", err)
		}

		for _, id := range ids {
			parents = append(parents, types.WebhookParentInfo{
				Type: enum.WebhookParentSpace,
				ID:   id,
			})
		}
	}

	return parents, nil
}

func (s *Service) getParentInfoSpace(
	ctx context.Context,
	spaceID int64,
	inherited bool,
) ([]types.WebhookParentInfo, error) {
	var parents []types.WebhookParentInfo

	if inherited {
		ids, err := s.spaceStore.GetAncestorIDs(ctx, spaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent space ids: %w", err)
		}

		for _, id := range ids {
			parents = append(parents, types.WebhookParentInfo{
				Type: enum.WebhookParentSpace,
				ID:   id,
			})
		}
	} else {
		parents = append(parents, types.WebhookParentInfo{
			Type: enum.WebhookParentSpace,
			ID:   spaceID,
		})
	}

	return parents, nil
}
