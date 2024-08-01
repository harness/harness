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
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// UnassignLabel removes a label from a pull request.
func (c *Controller) UnassignLabel(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	labelID int64,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pullreq, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return fmt.Errorf("failed to find pullreq: %w", err)
	}

	label, labelValue, err := c.labelSvc.UnassignFromPullReq(
		ctx, repo.ID, repo.ParentID, pullreq.ID, labelID)
	if err != nil {
		return fmt.Errorf("failed to delete pullreq label: %w", err)
	}

	pullreq, err = c.pullreqStore.UpdateOptLock(ctx, pullreq, func(pullreq *types.PullReq) error {
		pullreq.Edited = time.Now().UnixMilli()
		pullreq.ActivitySeq++
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update pull request: %w", err)
	}

	var value *string
	var color *enum.LabelColor
	if labelValue != nil {
		value = &labelValue.Value
		color = &labelValue.Color
	}
	payload := &types.PullRequestActivityLabel{
		Label:      label.Key,
		LabelColor: label.Color,
		Value:      value,
		ValueColor: color,
		Type:       enum.LabelActivityUnassign,
	}
	if _, err := c.activityStore.CreateWithPayload(
		ctx, pullreq, session.Principal.ID, payload, nil); err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to write pull request activity after label unassign")
	}

	return nil
}
