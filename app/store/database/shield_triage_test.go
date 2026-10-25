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

package database_test

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestShieldFindingStore_UpdateTriageSurvivesRescan(t *testing.T) {
	ctx, env, teardown := setupShieldEnv(t)
	defer teardown()

	scan1 := newShieldScan(t, ctx, env, 1)
	f := newShieldFinding(1, scan1.ID, "a", 90, 1000)
	if err := env.findings.Upsert(ctx, f); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	strategy := enum.ShieldFixStrategyPatch
	if err := env.findings.UpdateTriage(ctx, f.ID, &types.ShieldFindingTriage{
		RiskScore: 20, Reachable: enum.ShieldReachabilityNotReachable, FixStrategy: &strategy,
		AIDecisionIDs: []int64{4, 5},
	}, 1500); err != nil {
		t.Fatalf("UpdateTriage() error = %v", err)
	}

	// a rescan refreshes the deterministic risk but keeps the triaged reachability and strategy
	scan2 := newShieldScan(t, ctx, env, 1)
	again := newShieldFinding(1, scan2.ID, "a", 90, 2000)
	if err := env.findings.Upsert(ctx, again); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if again.RiskScore != 90 {
		t.Errorf("risk = %d, want refreshed 90", again.RiskScore)
	}
	if again.Reachable != enum.ShieldReachabilityNotReachable {
		t.Errorf("reachable = %s, want kept not_reachable", again.Reachable)
	}
	if again.FixStrategy == nil || *again.FixStrategy != enum.ShieldFixStrategyPatch {
		t.Errorf("fix strategy = %v, want kept patch", again.FixStrategy)
	}
	if len(again.AIDecisionIDs) != 2 || again.AIDecisionIDs[0] != 4 {
		t.Errorf("ai decision ids = %v", again.AIDecisionIDs)
	}
}
