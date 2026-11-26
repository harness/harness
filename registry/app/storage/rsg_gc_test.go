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

	"github.com/stretchr/testify/require"
)

func TestRsg_GcStorageClientRemoveBlob(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	store := rsgNewOciBlobStore(t, drv)
	dgst := rsgHelloDigest(t)
	_, err := store.Put(ctx, rsgAcctPrefix, []byte(rsgPayload))
	require.NoError(t, err)

	gc := NewGcStorageClient(drv)
	require.NoError(t, gc.RemoveBlob(ctx, dgst, rsgAcctPrefix))

	_, err = store.Stat(ctx, rsgAcctPrefix, dgst)
	require.ErrorIs(t, err, ErrBlobUnknown)
}
