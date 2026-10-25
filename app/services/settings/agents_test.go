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

package settings

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/refcache"

	"github.com/stretchr/testify/require"
)

func TestRepoGetAgentSettingsDefaults(t *testing.T) {
	t.Parallel()

	service := NewService(newInMemorySettingsStore(), refcache.SpaceFinder{})

	out, err := RepoGetAgentSettings(context.Background(), service, 1)
	require.NoError(t, err)
	require.False(t, *out.KillSwitch)
	require.Equal(t, AutonomyDraftPR, *out.Autonomy)
	require.Equal(t, 2, *out.MaxActive)
}

func TestRepoGetAgentSettingsOverridesAndClamping(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(newInMemorySettingsStore(), refcache.SpaceFinder{})

	require.NoError(t, service.RepoSetMany(ctx, 1,
		KeyValue{Key: KeyAIAgentsKillSwitch, Value: true},
		KeyValue{Key: KeyAIAgentsAutonomy, Value: 9},
		KeyValue{Key: KeyAIAgentsMaxActive, Value: -4},
	))

	out, err := RepoGetAgentSettings(ctx, service, 1)
	require.NoError(t, err)
	require.True(t, *out.KillSwitch)
	require.Equal(t, AutonomyAutoMerge, *out.Autonomy)
	require.Equal(t, 0, *out.MaxActive)

	kill, err := RepoGet(ctx, service, 2, KeyAIAgentsKillSwitch, DefaultAIAgentsKillSwitch)
	require.NoError(t, err)
	require.False(t, kill)
}

func TestAgentSettingsKeyValues(t *testing.T) {
	t.Parallel()

	require.Len(t, GetAgentSettingsAsKeyValues(GetDefaultAgentSettings()), 3)
	require.Empty(t, GetAgentSettingsAsKeyValues(&AgentSettings{}))
}
