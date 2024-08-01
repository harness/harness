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
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// AssignLabel assigns a label to a pull request .
func (c *Controller) AssignLabel(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *types.PullReqCreateInput,
) (*types.PullReqLabel, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate input: %w", err)
	}

	pullreq, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pullreq: %w", err)
	}

	out, err := c.labelSvc.AssignToPullReq(
		ctx, session.Principal.ID, pullreq.ID, repo.ID, repo.ParentID, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create pullreq label: %w", err)
	}

	if out.ActivityType == enum.LabelActivityNoop {
		return out.PullReqLabel, nil
	}

	pullreq, err = c.pullreqStore.UpdateOptLock(ctx, pullreq, func(pullreq *types.PullReq) error {
		pullreq.Edited = time.Now().UnixMilli()
		pullreq.ActivitySeq++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	payload := activityPayload(out)
	if _, err := c.activityStore.CreateWithPayload(
		ctx, pullreq, session.Principal.ID, payload, nil); err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to write pull request activity after label unassign")
	}

	return out.PullReqLabel, nil
}

func activityPayload(out *label.AssignToPullReqOut) *types.PullRequestActivityLabel {
	var oldValue *string
	var oldValueColor *enum.LabelColor
	if out.OldLabelValue != nil {
		oldValue = &out.OldLabelValue.Value
		oldValueColor = &out.OldLabelValue.Color
	}

	var value *string
	var valueColor *enum.LabelColor
	if out.NewLabelValue != nil {
		value = &out.NewLabelValue.Value
		valueColor = &out.NewLabelValue.Color
	}

	return &types.PullRequestActivityLabel{
		Label:         out.Label.Key,
		LabelColor:    out.Label.Color,
		Value:         value,
		ValueColor:    valueColor,
		OldValue:      oldValue,
		OldValueColor: oldValueColor,
		Type:          out.ActivityType,
	}
}
