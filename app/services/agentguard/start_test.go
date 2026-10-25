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
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/harness/gitness/app/services/settings"

	"github.com/stretchr/testify/require"
)

func TestGuardStartAttemptConcurrentMaxActive(t *testing.T) {
	t.Parallel()

	const workers, maxActive = 25, 4
	maxActiveVal := maxActive
	killSwitch, autonomy := false, settings.AutonomyDraftPR
	s := &settings.AgentSettings{KillSwitch: &killSwitch, Autonomy: &autonomy, MaxActive: &maxActiveVal}
	g := New(Config{GatewayEnabled: true}, fakeSettings{s: s}, NewMemoryAttemptStore(), nil, nil)

	var ok, denied atomic.Int32
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := g.StartAttempt(context.Background(), Request{RepoID: 1, TaskKey: "t" + strconv.Itoa(i)})
			if err == nil {
				ok.Add(1)
			} else if errors.Is(err, ErrAttemptDenied) {
				denied.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, maxActive, ok.Load())
	require.EqualValues(t, workers-maxActive, denied.Load())
}

func TestGuardStartAttemptSkipCooldown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	g := New(Config{GatewayEnabled: true}, nil, NewMemoryAttemptStore(), nil, nil)
	a, err := g.StartAttempt(ctx, Request{RepoID: 1, TaskKey: "t"})
	require.NoError(t, err)
	require.NoError(t, g.FinishAttempt(ctx, a.ID, AttemptFailed))

	_, err = g.StartAttempt(ctx, Request{RepoID: 1, TaskKey: "t"})
	require.ErrorIs(t, err, ErrAttemptDenied)
	require.ErrorContains(t, err, string(ReasonCooldown))

	_, err = g.TryStartAttempt(ctx, Request{RepoID: 1, TaskKey: "t"}, StartOptions{SkipCooldown: true})
	require.NoError(t, err)
}
