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

package manager

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestSanitizeSecretsForExecution(t *testing.T) {
	secrets := func() []*types.Secret {
		return []*types.Secret{
			{ID: 1, Identifier: "token", SpaceID: 10, Data: "super-secret"},
			{ID: 2, Identifier: "password", SpaceID: 10, Data: "hunter2"},
		}
	}

	tests := []struct {
		name       string
		execution  *types.Execution
		wantData   bool
		wantLength int
	}{
		{
			name:       "fork pull request strips secret values",
			execution:  &types.Execution{Event: enum.TriggerEventPullRequest, Fork: "attacker/fork"},
			wantData:   false,
			wantLength: 2,
		},
		{
			name:       "same-repo pull request strips secret values",
			execution:  &types.Execution{Event: enum.TriggerEventPullRequest, Fork: ""},
			wantData:   false,
			wantLength: 2,
		},
		{
			name:       "push keeps secret values",
			execution:  &types.Execution{Event: enum.TriggerEventPush},
			wantData:   true,
			wantLength: 2,
		},
		{
			name:       "tag keeps secret values",
			execution:  &types.Execution{Event: enum.TriggerEventTag},
			wantData:   true,
			wantLength: 2,
		},
		{
			name:       "manual keeps secret values",
			execution:  &types.Execution{Event: enum.TriggerEventManual},
			wantData:   true,
			wantLength: 2,
		},
		{
			name:       "cron keeps secret values",
			execution:  &types.Execution{Event: enum.TriggerEventCron},
			wantData:   true,
			wantLength: 2,
		},
		{
			name:       "nil execution keeps secret values",
			execution:  nil,
			wantData:   true,
			wantLength: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSecretsForExecution(secrets(), tt.execution)

			if len(got) != tt.wantLength {
				t.Fatalf("expected %d secrets, got %d", tt.wantLength, len(got))
			}

			for _, s := range got {
				hasData := s.Data != ""
				if hasData != tt.wantData {
					t.Errorf("secret %q: expected data present=%v, got data=%q",
						s.Identifier, tt.wantData, s.Data)
				}
				// Identifiers must always survive so expression lookups resolve.
				if s.Identifier == "" {
					t.Errorf("secret identifier was dropped")
				}
			}
		})
	}
}

func TestSanitizeSecretsForExecutionEmpty(t *testing.T) {
	got := sanitizeSecretsForExecution(nil, &types.Execution{Event: enum.TriggerEventPullRequest})
	if len(got) != 0 {
		t.Fatalf("expected no secrets, got %d", len(got))
	}
}
