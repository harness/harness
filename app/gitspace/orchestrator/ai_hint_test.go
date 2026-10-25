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

package orchestrator

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/gitspace/agentmodel"
)

func TestHintedModelSelector(t *testing.T) {
	reg := agentmodel.NewRegistry()
	reg.Set(7, "vendor/chosen:free")
	s := hintedModelSelector{hints: reg, fallback: fixedSelector{model: "vendor/default:free"}}

	got, err := s.SelectModel(agentmodel.WithConfigID(context.Background(), 7))
	if err != nil || got != "vendor/chosen:free" {
		t.Fatalf("got %q, %v", got, err)
	}
	// no hint for the config, or no config in the context: fall back
	for _, ctx := range []context.Context{agentmodel.WithConfigID(context.Background(), 8), context.Background()} {
		got, err = s.SelectModel(ctx)
		if err != nil || got != "vendor/default:free" {
			t.Fatalf("got %q, %v", got, err)
		}
	}
}
