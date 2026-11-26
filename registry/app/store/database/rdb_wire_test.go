// Copyright 2026 Harness, Inc.
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

package database

import (
	"testing"

	"github.com/harness/gitness/store/database/dbtx"

	"github.com/stretchr/testify/require"
)

func TestRdbWireProviders(t *testing.T) {
	e := rdbNewEnv(t)
	tx := dbtx.New(e.db)
	spaces := rdbNewSpaceFinder()

	registries := ProvideRegistryDao(e.db, e.mediaTypes, e.dcf)
	require.NotNil(t, registries)
	require.NotNil(t, ProvideUpstreamDao(e.db, registries, spaces))
	require.NotNil(t, ProvideMediaTypeDao(e.db))
	require.NotNil(t, ProvideBlobDao(e.db, e.mediaTypes))
	require.NotNil(t, ProvideRegistryBlobDao(e.db))
	require.NotNil(t, ProvideImageDao(e.db))
	require.NotNil(t, ProvideArtifactDao(e.db, e.dcf))
	require.NotNil(t, ProvideDownloadStatDao(e.db))
	require.NotNil(t, ProvideBandwidthStatDao(e.db))
	require.NotNil(t, ProvideTagDao(e.db, e.dcf))
	require.NotNil(t, ProvideManifestDao(e.db, e.mediaTypes))
	require.NotNil(t, ProvideWebhookDao(e.db))
	require.NotNil(t, ProvideWebhookExecutionDao(e.db))
	require.NotNil(t, ProvideManifestRefDao(e.db))
	require.NotNil(t, ProvideOCIImageIndexMappingDao(e.db))
	require.NotNil(t, ProvideLayerDao(e.db, e.mediaTypes))
	require.NotNil(t, ProvideCleanupPolicyDao(e.db, tx))
	require.NotNil(t, ProvideNodeDao(e.db))
	require.NotNil(t, ProvideQuarantineArtifactDao(e.db))
	require.NotNil(t, ProvidePackageTagDao(e.db))
	require.NotNil(t, ProvideGenericBlobDao(e.db))
	require.NotNil(t, ProvideTaskRepository(e.db, tx))
	require.NotNil(t, ProvideTaskSourceRepository(e.db, tx))
	require.NotNil(t, ProvideTaskEventRepository(e.db))
}
