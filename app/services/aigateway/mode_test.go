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

package aigateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeSettings struct {
	value *string
}

func (f fakeSettings) RepoGet(_ context.Context, _ int64, _ settings.Key, out any) (bool, error) {
	if f.value == nil {
		return false, nil
	}
	raw, err := json.Marshal(*f.value)
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(raw, out)
}

type fakePublicAccess struct {
	publicaccess.Service
	public bool
}

func (f fakePublicAccess) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return f.public, nil
}

func strp(s string) *string { return &s }

func TestModeResolverResolve(t *testing.T) {
	tests := []struct {
		name    string
		stored  *string
		public  bool
		want    enum.AIMode
		wantErr bool
	}{
		{"private default", nil, false, enum.AIModeDecisionsOnly, false},
		{"public default", nil, true, enum.AIModeFull, false},
		{"empty stored private", strp(""), false, enum.AIModeDecisionsOnly, false},
		{"empty stored public", strp(""), true, enum.AIModeFull, false},
		{"private raised to full", strp("full"), false, enum.AIModeFull, false},
		{"public lowered to off", strp("off"), true, enum.AIModeOff, false},
		{"public decisions-only", strp("decisions-only"), true, enum.AIModeDecisionsOnly, false},
		{"invalid stored", strp("everything"), true, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ModeResolver{settings: fakeSettings{tt.stored}, publicAccess: fakePublicAccess{public: tt.public}}
			got, err := r.Resolve(context.Background(), &types.RepositoryCore{ID: 1, Path: "a/b"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestModeHelpers(t *testing.T) {
	tests := []struct {
		mode                  enum.AIMode
		decisions, gen, agent bool
	}{
		{enum.AIModeFull, true, true, true},
		{enum.AIModeDecisionsOnly, true, false, false},
		{enum.AIModeOff, false, false, false},
		{"", false, false, false},
		{"bogus", false, false, false},
	}
	for _, tt := range tests {
		if AllowsDecisions(tt.mode) != tt.decisions || AllowsGeneration(tt.mode) != tt.gen ||
			AllowsAgents(tt.mode) != tt.agent {
			t.Errorf("mode %q: unexpected truth table", tt.mode)
		}
	}
}
