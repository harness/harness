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

package pkg

import (
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func TestRrmCargoPackageTypeBasics(t *testing.T) {
	cargo := NewCargoPackageType(nil, nil)
	require.Equal(t, string(artifact.PackageTypeCARGO), cargo.GetPackageType())
	require.Equal(t, string(types.PathPackageTypeCargo), cargo.GetPathPackageType())
	require.False(t, cargo.IsFileOperationSupported())
}

func TestRrmCargoPackageTypeValidation(t *testing.T) {
	cargo := NewCargoPackageType(nil, nil)
	require.True(t, cargo.IsValidRepoType(string(artifact.RegistryTypeVIRTUAL)))
	require.True(t, cargo.IsValidUpstreamSource(string(artifact.UpstreamConfigSourceCrates)))
	require.False(t, cargo.IsURLRequiredForUpstreamSource(string(artifact.UpstreamConfigSourceCrates)))
	require.True(t, cargo.IsURLRequiredForUpstreamSource("unknown-source"))
}
