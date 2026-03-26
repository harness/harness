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

package reposettings

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// GeneralUpdateSpace updates the general settings of the space.
func (c *Controller) GeneralUpdateSpace(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *settings.GeneralSettingsSpace,
) (*settings.GeneralSettingsSpace, error) {
	space, err := c.getSpaceCheckAccess(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, err
	}

	old, out, err := settings.SpaceUpdateGeneralSettings(ctx, c.settings, space.ID, in)
	if err != nil {
		return nil, err
	}

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeSpaceSettings, space.Identifier),
		audit.ActionUpdated,
		paths.Parent(space.Path),
		audit.WithOldObject(old),
		audit.WithNewObject(out),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf(
			"failed to insert audit log for update space settings operation: %s", err,
		)
	}

	return out, nil
}
