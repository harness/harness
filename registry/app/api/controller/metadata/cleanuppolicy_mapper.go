//  Copyright 2023 Harness, Inc.
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

package metadata

import (
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
)

func CreateCleanupPolicyEntity(
	config *artifact.ModifyRegistryJSONRequestBody,
	repoID int64,
) (*[]types.CleanupPolicy, error) {
	if config == nil || config.CleanupPolicy == nil {
		emptySlice := make([]types.CleanupPolicy, 0)
		return &emptySlice, nil
	}

	var cleanupPolicyEntities []types.CleanupPolicy
	cleanupPolicyDto := *config.CleanupPolicy

	for _, value := range cleanupPolicyDto {
		cleanupPolicyEntity, err := getCleanupPolicyEntity(value, repoID)
		if err != nil {
			return nil, err
		}
		cleanupPolicyEntities = append(cleanupPolicyEntities, *cleanupPolicyEntity)
	}
	return &cleanupPolicyEntities, nil
}

func CreateCleanupPolicyResponse(
	cleanupPolicyEntities *[]types.CleanupPolicy,
) *[]artifact.CleanupPolicy {
	var cleanupPolicyDtos []artifact.CleanupPolicy

	for _, value := range *cleanupPolicyEntities {
		cleanupPolicy := getCleanupPolicyDto(value)
		cleanupPolicyDtos = append(cleanupPolicyDtos, *cleanupPolicy)
	}
	return &cleanupPolicyDtos
}

func getCleanupPolicyEntity(
	cleanupPolicy artifact.CleanupPolicy,
	repoID int64,
) (*types.CleanupPolicy, error) {
	// Validate required fields
	if cleanupPolicy.ExpireDays == nil {
		return nil, usererror.BadRequest("expireDays is required for cleanup policy")
	}
	if cleanupPolicy.Name == nil {
		return nil, usererror.BadRequest("name is required for cleanup policy")
	}
	if cleanupPolicy.VersionPrefix == nil {
		return nil, usererror.BadRequest("versionPrefix is required for cleanup policy")
	}
	if cleanupPolicy.PackagePrefix == nil {
		return nil, usererror.BadRequest("packagePrefix is required for cleanup policy")
	}

	expireTime := time.Duration(*cleanupPolicy.ExpireDays) * 24 * time.Hour
	return &types.CleanupPolicy{
		Name:          *cleanupPolicy.Name,
		VersionPrefix: *cleanupPolicy.VersionPrefix,
		PackagePrefix: *cleanupPolicy.PackagePrefix,
		ExpiryTime:    expireTime.Milliseconds(),
		RegistryID:    repoID,
	}, nil
}

func getCleanupPolicyDto(
	cleanupPolicy types.CleanupPolicy,
) *artifact.CleanupPolicy {
	packagePrefix := cleanupPolicy.PackagePrefix
	versionPrefix := cleanupPolicy.VersionPrefix
	expiryDays := int(time.Duration(cleanupPolicy.ExpiryTime).Hours() / 24)

	return &artifact.CleanupPolicy{
		Name:          &cleanupPolicy.Name,
		VersionPrefix: &versionPrefix,
		PackagePrefix: &packagePrefix,
		ExpireDays:    &expiryDays,
	}
}
