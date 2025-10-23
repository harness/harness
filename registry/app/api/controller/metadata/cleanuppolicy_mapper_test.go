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
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
)

func TestCreateCleanupPolicyEntity_FunctionExists(t *testing.T) {
	t.Run("test_create_cleanup_policy_entity_exists", func(t *testing.T) {
		assert.NotNil(t, CreateCleanupPolicyEntity)

		// Check it can be called with nil safely
		result := CreateCleanupPolicyEntity(nil, 123)
		assert.Nil(t, result)
	})
}

func TestCreateCleanupPolicyResponse_FunctionExists(t *testing.T) {
	t.Run("test_create_cleanup_policy_response_exists", func(t *testing.T) {
		assert.NotNil(t, CreateCleanupPolicyResponse)

		// Check it can be called with nil slice safely
		var entities []types.CleanupPolicy
		result := CreateCleanupPolicyResponse(&entities)
		assert.NotNil(t, result)
		assert.Len(t, *result, 0)
	})
}

func TestGetCleanupPolicyEntity_FunctionExists(t *testing.T) {
	t.Run("test_get_cleanup_policy_entity_exists", func(t *testing.T) {
		assert.NotNil(t, getCleanupPolicyEntity)

		name := "policy1"
		versionPrefixSlice := []string{"v1."}
		packagePrefixSlice := []string{"pkg-"}
		expireDays := 7

		input := artifact.CleanupPolicy{
			Name:          &name,
			VersionPrefix: &versionPrefixSlice,
			PackagePrefix: &packagePrefixSlice,
			ExpireDays:    &expireDays,
		}

		entity := getCleanupPolicyEntity(input, 42)
		assert.NotNil(t, entity)
		assert.Equal(t, name, entity.Name)
		assert.Equal(t, []string{"v1."}, entity.VersionPrefix)
		assert.Equal(t, []string{"pkg-"}, entity.PackagePrefix)
		assert.Equal(t, int64(expireDays*24*60*60*1000), entity.ExpiryTime)
		assert.Equal(t, int64(42), entity.RegistryID)
	})
}

func TestGetCleanupPolicyDto_FunctionExists(t *testing.T) {
	t.Run("test_get_cleanup_policy_dto_exists", func(t *testing.T) {
		assert.NotNil(t, getCleanupPolicyDto)

		input := types.CleanupPolicy{
			Name:          "policy2",
			VersionPrefix: []string{"v1."},
			PackagePrefix: []string{"pkg-"},
			ExpiryTime:    int64(3 * 24 * 60 * 60 * 1000),
			RegistryID:    99,
		}

		dto := getCleanupPolicyDto(input)
		assert.NotNil(t, dto)
		assert.Equal(t, &input.Name, dto.Name)
		assert.Equal(t, &input.VersionPrefix, dto.VersionPrefix)
		assert.Equal(t, &input.PackagePrefix, dto.PackagePrefix)

		expireDays := int(time.Duration(input.ExpiryTime).Hours() / 24)
		assert.Equal(t, &expireDays, dto.ExpireDays)
	})
}
