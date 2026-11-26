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

package storage

import (
	"context"
	"testing"

	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_StaticStorageResolver(t *testing.T) {
	drv := rsgNewFilesystemDriver(t)
	resolver := NewStaticStorageResolver(drv)

	target, err := resolver.Resolve(context.Background(), types.StorageLookup{})
	require.NoError(t, err)
	assert.Equal(t, drv, target.Driver)
	assert.Equal(t, DefaultBucketKey, target.BucketKey)
	assert.True(t, target.IsDefault())

	nonDefault := StorageTarget{BucketKey: rsgBucketCustom}
	assert.False(t, nonDefault.IsDefault())
}

func TestRsg_GetRegistryOptions(t *testing.T) {
	opts := GetRegistryOptions()
	assert.Len(t, opts, 0)
}
