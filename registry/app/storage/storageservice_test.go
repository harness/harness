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
	"errors"
	"testing"

	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errResolve = errors.New("storage backend unavailable")

// mockStorageResolver is a test implementation of StorageResolver.
type mockStorageResolver struct {
	target StorageTarget
	err    error
}

func (m *mockStorageResolver) Resolve(_ context.Context, _ types.StorageLookup) (StorageTarget, error) {
	return m.target, m.err
}

func TestOciBlobsStore_ResolveError(t *testing.T) {
	resolver := &mockStorageResolver{err: errResolve}
	svc, err := NewStorageService(resolver)
	require.NoError(t, err)

	store, err := svc.OciBlobsStore(context.Background(), "repo", "root", types.BlobLocator{})
	assert.Nil(t, store)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errResolve)
	assert.Contains(t, err.Error(), "failed to resolve storage target")
}

func TestOciBlobsStore_DefaultBucket(t *testing.T) {
	resolver := &mockStorageResolver{
		target: StorageTarget{
			Driver:    nil,
			BucketKey: DefaultBucketKey,
		},
	}
	svc, err := NewStorageService(resolver)
	require.NoError(t, err)

	store, err := svc.OciBlobsStore(context.Background(), "repo", "root", types.BlobLocator{})
	require.NoError(t, err)
	assert.NotNil(t, store)

	ociStore, ok := store.(*ociBlobStore)
	require.True(t, ok)
	assert.Equal(t, "repo", ociStore.repoKey)
	assert.Equal(t, "root", ociStore.rootParentRef)
}

func TestOciBlobsStore_NonDefaultBucket(t *testing.T) {
	resolver := &mockStorageResolver{
		target: StorageTarget{
			Driver:    nil,
			BucketKey: "custom-bucket",
		},
	}
	svc, err := NewStorageService(resolver)
	require.NoError(t, err)

	store, err := svc.OciBlobsStore(context.Background(), "repo", "root", types.BlobLocator{})
	require.NoError(t, err)
	assert.NotNil(t, store)

	_, ok := store.(*globalBlobStore)
	assert.True(t, ok, "expected GlobalBlobStore for non-default bucket")
}

func TestGenericBlobsStore_ResolveError(t *testing.T) {
	resolver := &mockStorageResolver{err: errResolve}
	svc, err := NewStorageService(resolver)
	require.NoError(t, err)

	store, err := svc.GenericBlobsStore(context.Background(), "root", types.BlobLocator{})
	assert.Nil(t, store)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errResolve)
	assert.Contains(t, err.Error(), "failed to resolve storage target")
}

func TestGenericBlobsStore_DefaultBucket(t *testing.T) {
	resolver := &mockStorageResolver{
		target: StorageTarget{
			Driver:    nil,
			BucketKey: DefaultBucketKey,
		},
	}
	svc, err := NewStorageService(resolver)
	require.NoError(t, err)

	store, err := svc.GenericBlobsStore(context.Background(), "root", types.BlobLocator{})
	require.NoError(t, err)
	assert.NotNil(t, store)

	genericStore, ok := store.(*genericBlobStore)
	require.True(t, ok)
	assert.Equal(t, "root", genericStore.rootParentRef)
}

func TestGenericBlobsStore_NonDefaultBucket(t *testing.T) {
	resolver := &mockStorageResolver{
		target: StorageTarget{
			Driver:    nil,
			BucketKey: "custom-bucket",
		},
	}
	svc, err := NewStorageService(resolver)
	require.NoError(t, err)

	store, err := svc.GenericBlobsStore(context.Background(), "root", types.BlobLocator{})
	require.NoError(t, err)
	assert.NotNil(t, store)

	_, ok := store.(*globalBlobStore)
	assert.True(t, ok, "expected GlobalBlobStore for non-default bucket")
}

func TestOciBlobsStore_ServiceOptions(t *testing.T) {
	resolver := &mockStorageResolver{
		target: StorageTarget{
			Driver:    nil,
			BucketKey: DefaultBucketKey,
		},
	}
	svc, err := NewStorageService(resolver, EnableDelete, EnableRedirect)
	require.NoError(t, err)

	store, err := svc.OciBlobsStore(context.Background(), "repo", "root", types.BlobLocator{})
	require.NoError(t, err)

	ociStore, ok := store.(*ociBlobStore)
	require.True(t, ok)
	assert.True(t, ociStore.deleteEnabled)
	assert.True(t, ociStore.redirect)
	assert.True(t, ociStore.resumableDigestEnabled)
}
