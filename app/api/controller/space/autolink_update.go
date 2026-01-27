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
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) AutolinkUpdate(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	autolinkIdentifier int64,
	in *autolink.AutoLinkUpdateInput,
) (*types.AutoLink, error) {
	_, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, err
	}

	autolink, err := c.autolinkSvc.Update(ctx, autolinkIdentifier, session.Principal.ID, in)
	if err != nil {
		return nil, fmt.Errorf("failed to update autolink: %w", err)
	}

	return autolink, nil
}
