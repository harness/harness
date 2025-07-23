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
	SpaceRef           string       `json:"space_ref"`
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
	isValid, markForHardReset, err := c.isResourceSpecChangeAllowed(existingResource, newResource)
	if !isValid {
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

// isResourceSpecChangeAllowed checks if the new resource specs are valid and determines if a hard reset is needed.
// Returns (isAllowed, markForHardReset, error) where error contains details about why the validation failed.
func (c *Controller) isResourceSpecChangeAllowed(
	existingResource *gitnessTypes.InfraProviderResource,
	newResource *gitnessTypes.InfraProviderResource,
) (bool, bool, error) {
	// If either resource is nil, we can't compare properly
	if existingResource == nil || newResource == nil {
		return false, false, fmt.Errorf("cannot validate resource change: missing resource information")
	}

	// Validate region is the same
	if existingResource.Region != newResource.Region {
		return false, false, usererror.BadRequestf(
			"region mismatch: current region '%s' does not match target region '%s'",
			existingResource.Region, newResource.Region)
	}

	// Check zone from metadata if available
	existingZone, existingHasZone := existingResource.Metadata["zone"]
	newZone, newHasZone := newResource.Metadata["zone"]

	// If both resources have zone info, they must match
	if existingHasZone && newHasZone && existingZone != newZone {
		return false, false, usererror.BadRequestf(
			"zone mismatch: current zone '%s' does not match target zone '%s'",
			existingZone, newZone,
		)
	}

	markForHardReset := false

	// Check boot disk changes
	isAllowed, needsHardReset, err := c.validateBootDiskChanges(existingResource.Metadata, newResource.Metadata)
	if !isAllowed {
		return false, false, err
	}
	if needsHardReset {
		markForHardReset = true
	}

	// Check persistent disk changes
	isAllowed, needsHardReset, err = c.validatePersistentDiskChanges(existingResource.Metadata, newResource.Metadata)
	if !isAllowed {
		return false, false, err
	}
	if needsHardReset {
		markForHardReset = true
	}

	// Check machine type changes
	machineTypeResetNeeded := c.validateMachineTypeChanges(existingResource.Metadata, newResource.Metadata)
	markForHardReset = markForHardReset || machineTypeResetNeeded

	// All checks passed
	return true, markForHardReset, nil
}

// validateBootDiskChanges checks if boot disk changes are valid and if they require a hard reset.
// Returns (isAllowed, needsHardReset, error).
func (c *Controller) validateBootDiskChanges(existingMeta, newMeta map[string]string) (bool, bool, error) {
	markForHardReset := false

	// Check boot disk size changes
	existingBoot, existingOK := existingMeta["boot_disk_size"]
	newBoot, newOK := newMeta["boot_disk_size"]
	if !existingOK || !newOK {
		return false, false, fmt.Errorf(
			"invalid boot disk size format: cannot parse boot disk sizes for comparison")
	}

	existingVal, eErr := strconv.Atoi(existingBoot)
	newVal, nErr := strconv.Atoi(newBoot)
	if eErr != nil || nErr != nil {
		return false, false, fmt.Errorf(
			"invalid boot disk size format: cannot parse boot disk sizes for comparison")
	}
	if newVal < existingVal {
		return false, false, usererror.BadRequestf(
			"reducing boot disk size is not allowed: from %d to %d",
			existingVal, newVal)
	}
	if newVal != existingVal {
		markForHardReset = true
	}

	// Check boot disk type changes
	existingBootType, existingOK := existingMeta["boot_disk_type"]
	newBootType, newOK := newMeta["boot_disk_type"]
	if !existingOK || !newOK {
		return false, false, fmt.Errorf(
			"invalid boot disk type format: cannot parse boot disk types for comparison")
	}
	if existingBootType != newBootType {
		markForHardReset = true
	}

	return true, markForHardReset, nil
}

// validatePersistentDiskChanges checks if persistent disk changes are valid and if they require a hard reset.
// Returns (isAllowed, needsHardReset, error).
func (c *Controller) validatePersistentDiskChanges(existingMeta, newMeta map[string]string) (bool, bool, error) {
	existingDisk, existingOK := existingMeta["persistent_disk_size"]
	newDisk, newOK := newMeta["persistent_disk_size"]
	if !existingOK || !newOK {
		return false, false, fmt.Errorf(
			"invalid persistent disk size format: cannot parse persistent disk sizes for comparison")
	}

	isAllowed, markForHardReset, err := c.checkPersistentDiskSizeChange(existingDisk, newDisk)
	if !isAllowed && err != nil {
		return false, false, usererror.BadRequestf(err.Error())
	}

	existingDiskType, existingOK := existingMeta["persistent_disk_type"]
	newDiskType, newOK := newMeta["persistent_disk_type"]
	if !existingOK || !newOK {
		return false, false, fmt.Errorf(
			"invalid persistent disk type format: cannot parse persistent disk types for comparison")
	}
	if existingDiskType != newDiskType {
		return false, false, usererror.BadRequestf(
			"persistent disk type change not allowed: from '%s' to '%s'",
			existingDiskType, newDiskType)
	}

	return true, markForHardReset, nil
}

// validateMachineTypeChanges checks if machine type changes require a hard reset.
// Returns needsHardReset.
func (c *Controller) validateMachineTypeChanges(existingMeta, newMeta map[string]string) bool {
	existingMachine, existingOK := existingMeta["machine_type"]
	newMachine, newOK := newMeta["machine_type"]
	if existingOK && newOK && existingMachine != newMachine {
		return true
	}

	return false
}

// checkPersistentDiskSizeChange compares existing and new persistent disk sizes.
// and determines if the change is allowed and if hard reset is needed.
// Returns (isAllowed, needsHardReset, error).
func (c *Controller) checkPersistentDiskSizeChange(existingDisk, newDisk string) (bool, bool, error) {
	existingVal, eErr := strconv.Atoi(existingDisk)
	newVal, nErr := strconv.Atoi(newDisk)

	if eErr != nil {
		return false, false, fmt.Errorf("invalid disk size format: cannot parse existing disk size: %w", eErr)
	}
	if nErr != nil {
		return false, false, fmt.Errorf("invalid disk size format: cannot parse new disk size: %w", nErr)
	}

	// If persistent disk size reduced, return error
	if newVal < existingVal {
		return false, false, fmt.Errorf(
			"reducing persistent disk size is not allowed: from %d to %d",
			existingVal, newVal)
	}

	// If persistent disk size increased, mark for hard reset
	if newVal > existingVal {
		return true, true, nil
	}

	// Equal sizes, no hard reset needed
	return true, false, nil
}
