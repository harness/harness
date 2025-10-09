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

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
)

func CreateCleanupPolicyEntity(
	config *artifact.ModifyRegistryJSONRequestBody,
	repoID int64,
) *[]types.CleanupPolicy {
	if config == nil || config.CleanupPolicy == nil {
		return nil
	}

	var cleanupPolicyEntities []types.CleanupPolicy
	cleanupPolicyDto := *config.CleanupPolicy

	for _, value := range cleanupPolicyDto {
		cleanupPolicyEntity := getCleanupPolicyEntity(value, repoID)
		cleanupPolicyEntities = append(cleanupPolicyEntities, *cleanupPolicyEntity)
	}
	return &cleanupPolicyEntities
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
) *types.CleanupPolicy {
	expireTime := time.Duration(*cleanupPolicy.ExpireDays) * 24 * time.Hour
	return &types.CleanupPolicy{
		Name:          *cleanupPolicy.Name,
		VersionPrefix: *cleanupPolicy.VersionPrefix,
		PackagePrefix: *cleanupPolicy.PackagePrefix,
		ExpiryTime:    expireTime.Milliseconds(),
		RegistryID:    repoID,
	}
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
