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

	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

const (
	rsgAcctPrefix   = "acct"
	rsgRepoName     = "repo"
	rsgPayload      = "rsg blob payload"
	rsgGenericRoot  = "root1"
	rsgGenericSHA   = "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	rsgBucketCustom = "custom-bucket"
)

func rsgNewFilesystemDriver(t *testing.T) driver.StorageDriver {
	t.Helper()
	root := t.TempDir()
	d, err := filesystem.FromParameters(map[string]any{
		"rootdirectory": root,
		"maxthreads":    uint64(25),
	})
	require.NoError(t, err)
	return d
}

func rsgNewOciBlobStore(t *testing.T, drv driver.StorageDriver) *ociBlobStore {
	t.Helper()
	return &ociBlobStore{
		repoKey:                rsgRepoName,
		driver:                 drv,
		ctx:                    context.Background(),
		pathFn:                 PathFn,
		rootParentRef:          rsgAcctPrefix,
		resumableDigestEnabled: false,
	}
}

func rsgNewGenericBlobStore(t *testing.T, drv driver.StorageDriver) *genericBlobStore {
	t.Helper()
	return &genericBlobStore{
		driver:        drv,
		rootParentRef: rsgAcctPrefix,
	}
}

func rsgNewGlobalBlobStore(t *testing.T, drv driver.StorageDriver) *globalBlobStore {
	t.Helper()
	return &globalBlobStore{
		bucketKey:              rsgBucketCustom,
		driver:                 drv,
		ctx:                    context.Background(),
		resumableDigestEnabled: false,
	}
}

func rsgHelloDigest(t *testing.T) digest.Digest {
	t.Helper()
	return digest.FromBytes([]byte(rsgPayload))
}

func rsgStorageServiceWithDriver(t *testing.T, drv driver.StorageDriver) *Service {
	t.Helper()
	resolver := NewStaticStorageResolver(drv)
	svc, err := NewStorageService(resolver, EnableDelete, EnableRedirect)
	require.NoError(t, err)
	return svc
}

func rsgNonDefaultTarget(drv driver.StorageDriver) StorageTarget {
	return StorageTarget{
		Driver:    drv,
		BucketKey: types.BucketKey(rsgBucketCustom),
	}
}
