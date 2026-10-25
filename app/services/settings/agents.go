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
	"fmt"

	"github.com/gotidy/ptr"
)

// Autonomy levels of autonomous agents.
const (
	AutonomyObserve   = 0
	AutonomySuggest   = 1
	AutonomyDraftPR   = 2
	AutonomyAutoMerge = 3
)

// AgentSettings represents the autonomous agent related part of repository settings as exposed externally.
type AgentSettings struct {
	KillSwitch *bool `json:"ai_agents_kill_switch" yaml:"ai_agents_kill_switch"`
	Autonomy   *int  `json:"ai_agents_autonomy" yaml:"ai_agents_autonomy" description:"0 observe, 1 suggest, 2 draft PR, 3 auto-merge"` //nolint:lll
	MaxActive  *int  `json:"ai_agents_max_active" yaml:"ai_agents_max_active"`
}

func GetDefaultAgentSettings() *AgentSettings {
	return &AgentSettings{
		KillSwitch: ptr.Bool(DefaultAIAgentsKillSwitch),
		Autonomy:   ptr.Int(DefaultAIAgentsAutonomy),
		MaxActive:  ptr.Int(DefaultAIAgentsMaxActive),
	}
}

func GetAgentSettingsMappings(s *AgentSettings) []SettingHandler {
	return []SettingHandler{
		Mapping(KeyAIAgentsKillSwitch, s.KillSwitch),
		Mapping(KeyAIAgentsAutonomy, s.Autonomy),
		Mapping(KeyAIAgentsMaxActive, s.MaxActive),
	}
}

func GetAgentSettingsAsKeyValues(s *AgentSettings) []KeyValue {
	kvs := make([]KeyValue, 0, 3)

	if s.KillSwitch != nil {
		kvs = append(kvs, KeyValue{Key: KeyAIAgentsKillSwitch, Value: *s.KillSwitch})
	}
	if s.Autonomy != nil {
		kvs = append(kvs, KeyValue{Key: KeyAIAgentsAutonomy, Value: *s.Autonomy})
	}
	if s.MaxActive != nil {
		kvs = append(kvs, KeyValue{Key: KeyAIAgentsMaxActive, Value: *s.MaxActive})
	}

	return kvs
}

// RepoGetAgentSettings returns the agent settings of a repo with defaults applied.
// Out of range values are normalized: autonomy is clamped to [0,3], negative max active becomes 0.
func RepoGetAgentSettings(ctx context.Context, s *Service, repoID int64) (*AgentSettings, error) {
	out, err := RepoMapWithDefaults(ctx, s, repoID, GetDefaultAgentSettings, GetAgentSettingsMappings)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent settings: %w", err)
	}
	*out.Autonomy = min(max(*out.Autonomy, AutonomyObserve), AutonomyAutoMerge)
	*out.MaxActive = max(*out.MaxActive, 0)
	return out, nil
}
