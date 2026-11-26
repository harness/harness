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

package adapter

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/stretchr/testify/require"
)

type rrmStubFactory struct{}

func (rrmStubFactory) Create(
	_ context.Context, _ refcache.SpaceFinder, _ types.UpstreamProxy, _ secret.Service,
) (Adapter, error) {
	return rrmStubAdapter{}, nil
}

type rrmStubAdapter struct{}

func (rrmStubAdapter) HealthCheck() (string, error) { return "ok", nil }
func (rrmStubAdapter) GetImageName(imageName string) (string, error) {
	return imageName, nil
}

func TestRrmRegisterFactory(t *testing.T) {
	require.Error(t, RegisterFactory("", rrmStubFactory{}))
	require.Error(t, RegisterFactory("rrm-test-type", nil))

	const rrmType = "rrm-test-adapter-type"
	require.NoError(t, RegisterFactory(rrmType, rrmStubFactory{}))
	require.Error(t, RegisterFactory(rrmType, rrmStubFactory{}))

	factory, err := GetFactory(rrmType)
	require.NoError(t, err)
	require.NotNil(t, factory)

	_, err = GetFactory("rrm-missing-adapter-type")
	require.Error(t, err)

	types := ListRegisteredAdapterTypes()
	require.Contains(t, types, rrmType)
}
