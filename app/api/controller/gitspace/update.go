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

package gitspace

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/gitspace/common"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	gitnessTypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a gitspace.
type UpdateInput struct {
	IDE                enum.IDEType `json:"ide"`
	ResourceIdentifier string       `json:"resource_identifier"`
	ResourceSpaceRef   string       `json:"resource_space_ref"`
	Name               string       `json:"name"`
	SSHTokenIdentifier string       `json:"ssh_token_identifier"`
	Identifier         string       `json:"-"`
	SpaceRef           string       `json:"-"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	in *UpdateInput,
) (*gitnessTypes.GitspaceConfig, error) {
	in.SpaceRef = spaceRef
	in.Identifier = identifier
	if err := c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}
	err := apiauth.CheckGitspace(ctx, c.authorizer, session, spaceRef, identifier, enum.PermissionGitspaceEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	gitspaceConfig, err := c.gitspaceSvc.FindWithLatestInstanceWithSpacePath(ctx, spaceRef, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find gitspace config: %w", err)
	}

	// Check the gitspace state. Update can be done only in stopped, error or uninitialized state
	currentState := gitspaceConfig.State
	if currentState != enum.GitspaceStateStopped &&
		currentState != enum.GitspaceStateError &&
		currentState != enum.GitspaceStateUninitialized {
		return nil, usererror.BadRequest(
			"Gitspace update can only be performed when gitspace is stopped, in error state, or uninitialized",
		)
	}

	c.updateIDE(in, gitspaceConfig)
	if err := c.handleSSHToken(in, gitspaceConfig); err != nil {
		return nil, err
	}
	if err := c.updateResourceIdentifier(ctx, in, gitspaceConfig); err != nil {
		return nil, err
	}

	// TODO Update with proper locks
	err = c.gitspaceSvc.UpdateConfig(ctx, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to update gitspace config: %w", err)
	}
	return gitspaceConfig, nil
}

func (c *Controller) updateIDE(in *UpdateInput, gitspaceConfig *gitnessTypes.GitspaceConfig) {
	if in.IDE != "" && in.IDE != gitspaceConfig.IDE {
		gitspaceConfig.IDE = in.IDE
		gitspaceConfig.IsMarkedForSoftReset = true
	}

	// Always clear SSH token if IDE is VS Code Web
	if gitspaceConfig.IDE == enum.IDETypeVSCodeWeb {
		gitspaceConfig.SSHTokenIdentifier = ""
	}
}

func (c *Controller) handleSSHToken(in *UpdateInput, gitspaceConfig *gitnessTypes.GitspaceConfig) error {
	if in.SSHTokenIdentifier != "" {
		if gitspaceConfig.IDE == enum.IDETypeVSCodeWeb {
			return usererror.BadRequest("SSH token should not be sent with VS Code Web IDE")
		}

		// For other IDEs, update the token
		if in.SSHTokenIdentifier != gitspaceConfig.SSHTokenIdentifier {
			gitspaceConfig.SSHTokenIdentifier = in.SSHTokenIdentifier
			gitspaceConfig.IsMarkedForSoftReset = true
		}
	}

	return nil
}

func (c *Controller) updateResourceIdentifier(
	ctx context.Context,
	in *UpdateInput,
	gitspaceConfig *gitnessTypes.GitspaceConfig,
) error {
	// Handle resource identifier update similar to create, but only if provided
	if in.ResourceIdentifier == "" || in.ResourceIdentifier == gitspaceConfig.InfraProviderResource.UID {
		return nil
	}

	if gitspaceConfig.InfraProviderResource.UID == "default" {
		return usererror.BadRequest("The default resource cannot be updated in harness open source")
	}

	// Set resource space reference if not provided
	if in.ResourceSpaceRef == "" {
		rootSpaceRef, _, err := paths.DisectRoot(in.SpaceRef)
		if err != nil {
			return fmt.Errorf("unable to find root space path for %s: %w", in.SpaceRef, err)
		}
		in.ResourceSpaceRef = rootSpaceRef
	}

	// Find spaces and resources
	existingResource, newResource, err := c.getResources(ctx, in, gitspaceConfig)
	if err != nil {
		return err
	}

	// Validate the resource spec change
	markForHardReset, err := common.IsResourceSpecChangeAllowed(existingResource, newResource)
	if err != nil {
		return err
	}

	gitspaceConfig.IsMarkedForReset = gitspaceConfig.IsMarkedForReset || markForHardReset
	gitspaceConfig.InfraProviderResource = *newResource

	return nil
}

func (c *Controller) getResources(
	ctx context.Context,
	in *UpdateInput,
	gitspaceConfig *gitnessTypes.GitspaceConfig,
) (*gitnessTypes.InfraProviderResource, *gitnessTypes.InfraProviderResource, error) {
	// Get existing resource space and resource
	existingSpace, err := c.spaceFinder.FindByRef(
		ctx,
		gitspaceConfig.InfraProviderResource.SpacePath,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find resource space: %w", err)
	}

	existingResource, err := c.infraProviderSvc.FindResourceByConfigAndIdentifier(
		ctx,
		existingSpace.ID,
		gitspaceConfig.InfraProviderResource.InfraProviderConfigIdentifier,
		gitspaceConfig.InfraProviderResource.UID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not find existing infra provider resource: %w",
			err,
		)
	}

	// Get new resource space and resource
	newSpace, err := c.spaceFinder.FindByRef(
		ctx,
		in.ResourceSpaceRef,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find resource space: %w", err)
	}

	newResource, err := c.infraProviderSvc.FindResourceByConfigAndIdentifier(
		ctx,
		newSpace.ID,
		gitspaceConfig.InfraProviderResource.InfraProviderConfigIdentifier,
		in.ResourceIdentifier,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not find infra provider resource %q: %w",
			in.ResourceIdentifier,
			err,
		)
	}

	return existingResource, newResource, nil
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return ErrGitspaceRequiresParent
	}

	//nolint:revive
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	return nil
}
