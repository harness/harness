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

package agentguard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/require"
)

type fakeSettings struct {
	s   *settings.AgentSettings
	err error
}

func (f fakeSettings) AgentSettings(context.Context, int64) (*settings.AgentSettings, error) {
	return f.s, f.err
}

type auditEntry struct {
	action   audit.Action
	resource audit.Resource
	data     map[string]string
}

type fakeAudit struct {
	entries []auditEntry
}

func (f *fakeAudit) Log(
	_ context.Context,
	_ types.Principal,
	r audit.Resource,
	a audit.Action,
	_ string,
	opts ...audit.Option,
) error {
	e := &audit.Event{}
	for _, o := range opts {
		o.Apply(e)
	}
	f.entries = append(f.entries, auditEntry{action: a, resource: r, data: e.Data})
	return nil
}

func TestGuardCheck(t *testing.T) {
	t.Parallel()

	const (
		agentID = int64(99)
		repoID  = int64(1)
		task    = "task-1"
	)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	agent := &types.Principal{ID: agentID, UID: "harness-agent"}

	tests := []struct {
		name     string
		cfg      Config
		repo     *settings.AgentSettings
		settErr  error
		noAgent  bool
		req      Request
		seed     func(ctx context.Context, s *MemoryAttemptStore)
		wantOK   bool
		wantWhy  Reason
		wantAudt audit.Action
	}{
		{name: "allowed by defaults", wantOK: true, wantWhy: ReasonAllowed},
		{name: "gateway disabled", cfg: Config{MaxFailedAttempts: 3}, wantWhy: ReasonGatewayDisabled},
		{name: "global kill switch", cfg: Config{GatewayEnabled: true, KillSwitch: true}, wantWhy: ReasonGlobalKillSwitch},
		{
			name:    "repo kill switch",
			repo:    &settings.AgentSettings{KillSwitch: ptr.Bool(true)},
			wantWhy: ReasonRepoKillSwitch,
		},
		{
			name:    "autonomy too low",
			repo:    &settings.AgentSettings{Autonomy: ptr.Int(settings.AutonomySuggest)},
			wantWhy: ReasonAutonomyTooLow,
		},
		{
			name:   "autonomy equal to required",
			repo:   &settings.AgentSettings{Autonomy: ptr.Int(settings.AutonomyDraftPR)},
			wantOK: true, wantWhy: ReasonAllowed,
		},
		{name: "triggered by agent", req: Request{TriggeredByPrincipalID: agentID}, wantWhy: ReasonTriggeredByAgent},
		{name: "triggered by other", req: Request{TriggeredByPrincipalID: 5}, wantOK: true, wantWhy: ReasonAllowed},
		{name: "settings error fails closed", settErr: errors.New("boom"), wantWhy: ReasonInternalError},
		{name: "no agent principal", noAgent: true, wantWhy: ReasonAgentNotConfigured},
		{
			name: "too many failed attempts",
			seed: func(ctx context.Context, s *MemoryAttemptStore) {
				for range 3 {
					a, _ := s.Start(ctx, task, repoID, now.Add(-24*time.Hour))
					_ = s.Finish(ctx, a.ID, AttemptFailed)
				}
			},
			wantWhy: ReasonTooManyFailures,
		},
		{
			name: "two failures long ago is fine",
			seed: func(ctx context.Context, s *MemoryAttemptStore) {
				for range 2 {
					a, _ := s.Start(ctx, task, repoID, now.Add(-24*time.Hour))
					_ = s.Finish(ctx, a.ID, AttemptFailed)
				}
			},
			wantOK: true, wantWhy: ReasonAllowed,
		},
		{
			name: "cooldown active",
			seed: func(ctx context.Context, s *MemoryAttemptStore) {
				a, _ := s.Start(ctx, task, repoID, now.Add(-10*time.Minute))
				_ = s.Finish(ctx, a.ID, AttemptSucceeded)
			},
			wantWhy: ReasonCooldown,
		},
		{
			name: "cooldown elapsed",
			seed: func(ctx context.Context, s *MemoryAttemptStore) {
				a, _ := s.Start(ctx, task, repoID, now.Add(-31*time.Minute))
				_ = s.Finish(ctx, a.ID, AttemptSucceeded)
			},
			wantOK: true, wantWhy: ReasonAllowed,
		},
		{
			name: "other task attempts ignored",
			seed: func(ctx context.Context, s *MemoryAttemptStore) {
				_, _ = s.Start(ctx, "other", 2, now.Add(-time.Minute))
			},
			wantOK: true, wantWhy: ReasonAllowed,
		},
		{
			name: "max active agents",
			repo: &settings.AgentSettings{MaxActive: ptr.Int(1)},
			seed: func(ctx context.Context, s *MemoryAttemptStore) {
				_, _ = s.Start(ctx, "other", repoID, now.Add(-time.Hour))
			},
			wantWhy: ReasonMaxActiveAgents,
		},
		{
			name:    "max active zero blocks",
			repo:    &settings.AgentSettings{MaxActive: ptr.Int(0)},
			wantWhy: ReasonMaxActiveAgents,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			cfg := tt.cfg
			if cfg == (Config{}) {
				cfg = Config{GatewayEnabled: true}
			}

			s := settings.GetDefaultAgentSettings()
			if tt.repo != nil {
				if tt.repo.KillSwitch != nil {
					s.KillSwitch = tt.repo.KillSwitch
				}
				if tt.repo.Autonomy != nil {
					s.Autonomy = tt.repo.Autonomy
				}
				if tt.repo.MaxActive != nil {
					s.MaxActive = tt.repo.MaxActive
				}
			}

			store := NewMemoryAttemptStore()
			if tt.seed != nil {
				tt.seed(ctx, store)
			}
			al := &fakeAudit{}
			principal := func() *types.Principal {
				if tt.noAgent {
					return nil
				}
				return agent
			}

			g := New(cfg, fakeSettings{s: s, err: tt.settErr}, store, al, principal)
			g.now = func() time.Time { return now }

			req := tt.req
			req.RepoID = repoID
			req.TaskKey = task
			req.MinAutonomy = settings.AutonomyDraftPR
			req.SpacePath = "space"

			d := g.Check(ctx, req)
			require.Equal(t, tt.wantOK, d.Allowed)
			require.Equal(t, tt.wantWhy, d.Reason)

			if tt.noAgent {
				require.Empty(t, al.entries)
				return
			}
			require.Len(t, al.entries, 1)
			require.Equal(t, audit.ResourceTypeAgentTask, al.entries[0].resource.Type)
			require.Equal(t, string(tt.wantWhy), al.entries[0].data["reason"])
			if tt.wantOK {
				require.Equal(t, audit.ActionAllowed, al.entries[0].action)
			} else {
				require.Equal(t, audit.ActionDenied, al.entries[0].action)
			}
		})
	}
}

func TestGuardConfigDefaults(t *testing.T) {
	t.Parallel()

	g := New(Config{}, nil, nil, nil, nil)
	require.Equal(t, 3, g.config.MaxFailedAttempts)
	require.Equal(t, 30*time.Minute, g.config.Cooldown)
}

func TestGuardAttemptLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMemoryAttemptStore()
	g := New(Config{GatewayEnabled: true}, nil, store, nil, nil)

	a, err := g.StartAttempt(ctx, Request{RepoID: 1, TaskKey: "t"})
	require.NoError(t, err)
	n, _ := store.CountActive(ctx, 1)
	require.Equal(t, 1, n)

	require.NoError(t, g.FinishAttempt(ctx, a.ID, AttemptFailed))
	n, _ = store.CountActive(ctx, 1)
	require.Equal(t, 0, n)
	list, _ := store.ListByTask(ctx, "t")
	require.Len(t, list, 1)
	require.Equal(t, AttemptFailed, list[0].Status)
}
