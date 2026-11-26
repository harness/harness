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

package githook

import (
	"testing"

	"github.com/harness/gitness/app/services/refcache"

	"github.com/stretchr/testify/require"
)

func TestCt5ProvideFactoryAndController(t *testing.T) {
	factory := ProvideFactory()
	require.NotNil(t, factory)

	ctrl := ProvideController(
		nil, nil, nil, refcache.RepoFinder{}, nil, nil, nil, nil, nil, nil, factory, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
	)
	require.NotNil(t, ctrl)

	concrete, ok := factory.(*ControllerClientFactory)
	require.True(t, ok)
	require.Same(t, ctrl, concrete.githookCtrl)
}

func TestCt5ProvideExtenders(t *testing.T) {
	pre, err := ProvidePreReceiveExtender()
	require.NoError(t, err)
	require.NotNil(t, pre)

	update, err := ProvideUpdateExtender()
	require.NoError(t, err)
	require.NotNil(t, update)

	post, err := ProvidePostReceiveExtender()
	require.NoError(t, err)
	require.NotNil(t, post)
}
