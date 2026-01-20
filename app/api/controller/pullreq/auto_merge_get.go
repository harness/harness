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

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *Controller) AutoMergeGet(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (*types.AutoMergeResponse, error) {
	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if pr.SubState != enum.PullReqSubStateAutoMerge {
		return nil, errors.NotFound("Auto-merge isn't enabled.")
	}

	autoMerge, err := c.autoMergeStore.Find(ctx, pr.ID)
	if errors.Is(err, store.ErrResourceNotFound) {
		log.Ctx(ctx).Warn().Msg("pull request marked for auto merging, but no auto merge entry found in DB")
		return nil, errors.NotFound("Auto-merge isn't enabled.")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find auto merge state: %w", err)
	}

	return &types.AutoMergeResponse{
		Requested:    autoMerge.Requested,
		RequestedBy:  session.Principal.ToPrincipalInfo(),
		MergeMethod:  autoMerge.MergeMethod,
		Title:        autoMerge.Title,
		Message:      autoMerge.Message,
		DeleteBranch: autoMerge.DeleteBranch,
	}, nil
}
