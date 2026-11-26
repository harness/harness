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

func TestRsg_StorageServiceGlobalBlobsStore(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	svc := rsgStorageServiceWithDriver(t, drv)

	global := svc.GlobalBlobsStore(ctx, rsgNonDefaultTarget(drv), true)
	require.NotNil(t, global)
	assert.Equal(t, types.BucketKey(rsgBucketCustom), global.BucketKey())

	oci := svc.GlobalBlobsStore(ctx, rsgNonDefaultTarget(drv), false)
	require.NotNil(t, oci)
}

func TestRsg_StorageServiceWithRealDriver(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	resolver := NewStaticStorageResolver(drv)
	svc, err := NewStorageService(resolver, EnableDelete, EnableRedirect)
	require.NoError(t, err)

	oci, err := svc.OciBlobsStore(ctx, rsgRepoName, rsgAcctPrefix, types.BlobLocator{})
	require.NoError(t, err)
	require.NotNil(t, oci)

	generic, err := svc.GenericBlobsStore(ctx, rsgAcctPrefix, types.BlobLocator{})
	require.NoError(t, err)
	require.NotNil(t, generic)
}

func TestRsg_StorageServiceOptions(t *testing.T) {
	svc := &Service{}
	require.NoError(t, EnableRedirect(svc))
	require.NoError(t, EnableDelete(svc))
	assert.True(t, svc.redirect)
	assert.True(t, svc.deleteEnabled)
}
