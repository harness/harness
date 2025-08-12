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

package common

import (
	"context"
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// FilterResourcesByCompatibility filters resources based on compatibility with a reference resource.
// It removes any resources that are not compatible according to the IsResourceSpecChangeAllowed criteria.
func FilterResourcesByCompatibility(
	ctx context.Context,
	filteredResources []*types.InfraProviderResource,
	referenceResource *types.InfraProviderResource,
) ([]*types.InfraProviderResource, error) {
	if referenceResource == nil {
		return nil, fmt.Errorf("referenceResource cannot be nil")
	}

	compatibleResources := make([]*types.InfraProviderResource, 0)

	// Now filter based on compatibility
	for _, resource := range filteredResources {
		// Skip the current resource itself
		if resource.UID == referenceResource.UID {
			continue
		}

		_, err := IsResourceSpecChangeAllowed(referenceResource, resource)
		if err != nil {
			log.Ctx(ctx).Debug().
				Err(err).
				Str("resource_id", resource.UID).
				Str("reference_id", referenceResource.UID).
				Msg("resource compatibility check failed")
		} else {
			compatibleResources = append(compatibleResources, resource)
		}
	}

	return compatibleResources, nil
}

// IsResourceSpecChangeAllowed checks if the new resource specs are valid and determines if a hard reset is needed.
// Returns (markForHardReset, error) where error contains details about why the validation failed.
func IsResourceSpecChangeAllowed(
	existingResource *types.InfraProviderResource,
	newResource *types.InfraProviderResource,
) (bool, error) {
	// If either resource is nil, we can't compare properly
	if existingResource == nil || newResource == nil {
		return false, fmt.Errorf("cannot validate resource change: missing resource information")
	}

	// Validate region is the same
	if existingResource.Region != newResource.Region {
		return false, usererror.BadRequestf(
			"region mismatch: current region '%s' does not match target region '%s'",
			existingResource.Region, newResource.Region)
	}

	// Check zone from metadata if available
	existingZone, existingHasZone := existingResource.Metadata["zone"]
	newZone, newHasZone := newResource.Metadata["zone"]

	// If both resources have zone info, they must match
	if existingHasZone && newHasZone && existingZone != newZone {
		return false, usererror.BadRequestf(
			"zone mismatch: current zone '%s' does not match target zone '%s'",
			existingZone, newZone,
		)
	}

	markForInfraReset := false

	// Check boot disk changes
	needsHardReset, err := validateBootDiskChanges(existingResource.Metadata, newResource.Metadata)
	if err != nil {
		return false, err
	}
	if needsHardReset {
		markForInfraReset = true
	}

	// Check persistent disk changes
	needsHardReset, err = validatePersistentDiskChanges(existingResource.Metadata, newResource.Metadata)
	if err != nil {
		return false, err
	}
	if needsHardReset {
		markForInfraReset = true
	}

	// Check machine type changes
	machineTypeResetNeeded := validateMachineTypeChanges(existingResource.Metadata, newResource.Metadata)
	markForInfraReset = markForInfraReset || machineTypeResetNeeded

	// All checks passed
	return markForInfraReset, nil
}

// validatePersistentDiskChanges checks if persistent disk changes are valid and if they require a hard reset.
// Returns (needsHardReset, error).
func validatePersistentDiskChanges(existingMeta, newMeta map[string]string) (bool, error) {
	existingDisk, existingOK := existingMeta["persistent_disk_size"]
	newDisk, newOK := newMeta["persistent_disk_size"]
	if !existingOK || !newOK {
		return false, fmt.Errorf(
			"invalid persistent disk size format: cannot parse persistent disk sizes for comparison")
	}

	markForHardReset, err := checkPersistentDiskSizeChange(existingDisk, newDisk)
	if err != nil {
		return false, err
	}

	existingDiskType, existingOK := existingMeta["persistent_disk_type"]
	newDiskType, newOK := newMeta["persistent_disk_type"]
	if !existingOK || !newOK {
		return false, fmt.Errorf(
			"invalid persistent disk type format: cannot parse persistent disk types for comparison")
	}
	if existingDiskType != newDiskType {
		return false, usererror.BadRequestf(
			"persistent disk type change not allowed: from '%s' to '%s'",
			existingDiskType, newDiskType)
	}

	return markForHardReset, nil
}

// validateMachineTypeChanges checks if machine type changes require a hard reset.
// Returns needsHardReset.
func validateMachineTypeChanges(existingMeta, newMeta map[string]string) bool {
	existingMachine, existingOK := existingMeta["machine_type"]
	newMachine, newOK := newMeta["machine_type"]
	if existingOK && newOK && existingMachine != newMachine {
		return true
	}

	return false
}

// validateBootDiskChanges checks if boot disk changes are valid and if they require a hard reset.
// Returns (needsHardReset, error).
func validateBootDiskChanges(existingMeta, newMeta map[string]string) (bool, error) {
	markForHardReset := false

	// Check boot disk size changes
	existingBoot, existingOK := existingMeta["boot_disk_size"]
	newBoot, newOK := newMeta["boot_disk_size"]
	if !existingOK || !newOK {
		return false, fmt.Errorf(
			"invalid boot disk size format: cannot parse boot disk sizes for comparison")
	}

	existingVal, eErr := strconv.Atoi(existingBoot)
	newVal, nErr := strconv.Atoi(newBoot)
	if eErr != nil || nErr != nil {
		return false, fmt.Errorf(
			"invalid boot disk size format: cannot parse boot disk sizes for comparison")
	}
	if newVal != existingVal {
		markForHardReset = true
	}

	// Check boot disk type changes
	existingBootType, existingOK := existingMeta["boot_disk_type"]
	newBootType, newOK := newMeta["boot_disk_type"]
	if !existingOK || !newOK {
		return false, fmt.Errorf(
			"invalid boot disk type format: cannot parse boot disk types for comparison")
	}
	if existingBootType != newBootType {
		markForHardReset = true
	}

	return markForHardReset, nil
}

// checkPersistentDiskSizeChange compares existing and new persistent disk sizes.
// and determines if the change is allowed and if hard reset is needed.
// Returns (needsHardReset, error).
//
//nolint:unparam // the bool return value is kept for future extension
func checkPersistentDiskSizeChange(existingDisk, newDisk string) (bool, error) {
	existingVal, eErr := strconv.Atoi(existingDisk)
	if eErr != nil {
		return false, fmt.Errorf("invalid disk size format: cannot parse existing disk size: %w", eErr)
	}

	newVal, nErr := strconv.Atoi(newDisk)
	if nErr != nil {
		return false, fmt.Errorf("invalid disk size format: cannot parse new disk size: %w", nErr)
	}

	// Disallow any changes to persistent disk size
	if newVal != existingVal {
		return false, fmt.Errorf(
			"changing persistent disk size is not allowed: from %d to %d",
			existingVal, newVal)
	}

	// Equal sizes, no hard reset needed
	return false, nil
}
