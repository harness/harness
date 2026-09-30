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
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/types/enum"
)

// GeneralFindSpace returns the general settings of a space.
func (c *Controller) GeneralFindSpace(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	inherited bool,
) (*settings.GeneralSettingsSpace, error) {
	space, err := c.getSpaceCheckAccess(ctx, session, spaceRef, enum.PermissionSpaceView)
	if err != nil {
		return nil, err
	}

	out := settings.GetDefaultGeneralSettingsSpace()

	// The non-inherited path returns only the space-local value (no provenance to report).
	if !inherited {
		defaultBranch, err := c.settings.SpaceGetDefaultBranch(ctx, space.ID, false)
		if err != nil {
			return nil, err
		}

		out.DefaultBranch = &defaultBranch

		return out, nil
	}

	// The inherited path resolves the effective value and reports which space configures it so the
	// caller can distinguish "set here" from "inherited from <scope>".
	defaultBranch, sourceSpaceID, found, err := c.settings.SpaceGetDefaultBranchWithSource(ctx, space.ID)
	if err != nil {
		return nil, err
	}

	out.DefaultBranch = &defaultBranch

	if found {
		sourceSpace, err := c.spaceFinder.FindByID(ctx, sourceSpaceID)
		if err != nil {
			return nil, err
		}

		out.DefaultBranchScope = &sourceSpace.Path
	}

	return out, nil
}
