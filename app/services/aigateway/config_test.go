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

package aigateway

import "testing"

func TestPerReplicaRPM(t *testing.T) {
	tests := []struct {
		name     string
		rpm      int
		replicas int
		want     int
	}{
		{"default single replica", 18, 0, 18},
		{"explicit one", 18, 1, 18},
		{"split", 18, 3, 6},
		{"rounds down", 18, 4, 4},
		{"never below one", 18, 100, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{RequestsPerMinute: tt.rpm, Replicas: tt.replicas}
			c.Sanitize()
			if got := c.perReplicaRPM(); got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestPlannerUsesPerReplicaShare(t *testing.T) {
	g := New(Config{Enabled: true, RequestsPerMinute: 18, Replicas: 6}, &Redactor{})
	if g.planner.rpm != 3 {
		t.Fatalf("expected planner rpm 3, got %d", g.planner.rpm)
	}
}
