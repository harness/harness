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
	"errors"
	"testing"
)

func TestPricingIsZero(t *testing.T) {
	tests := []struct {
		p    Pricing
		want bool
	}{
		{Pricing{Prompt: "0", Completion: "0"}, true},
		{Pricing{Prompt: "0.0", Completion: "0", Request: "0", Image: "0"}, true},
		{Pricing{Prompt: "0", Completion: "0.000001"}, false},
		{Pricing{Prompt: "0", Completion: "0", Request: "0.01"}, false},
		{Pricing{Prompt: "0", Completion: "0", Image: "0.002"}, false},
		{Pricing{Prompt: "", Completion: "0"}, false},
		{Pricing{Prompt: "free", Completion: "0"}, false},
		{Pricing{Prompt: "-1", Completion: "0"}, false},
	}
	for _, tt := range tests {
		if got := tt.p.IsZero(); got != tt.want {
			t.Errorf("%+v: expected %v, got %v", tt.p, tt.want, got)
		}
	}
}

func TestIsFreeModelID(t *testing.T) {
	for id, want := range map[string]bool{
		"baseten/mammoth-1:free": true,
		"openrouter/free":        true,
		"openrouter/auto":        false,
		"vendor/model":           false,
		"vendor/model:freebie":   false,
	} {
		if got := IsFreeModelID(id); got != want {
			t.Errorf("%s: expected %v, got %v", id, want, got)
		}
	}
}

func TestGuardVerifyCost(t *testing.T) {
	g := NewGuard()
	if err := g.VerifyCost("a/b:free", 0); err != nil {
		t.Errorf("zero cost must pass: %v", err)
	}
	if err := g.VerifyCost("a/b:free", 0.0001); !errors.Is(err, ErrPaidUsage) {
		t.Errorf("expected ErrPaidUsage, got %v", err)
	}
	if err := g.Check("a/b:free"); !errors.Is(err, ErrModelQuarantine) {
		t.Errorf("expected quarantine, got %v", err)
	}
	// a catalog sync must not lift a quarantine
	g.SetAllowed([]string{"a/b:free"})
	if err := g.Check("a/b:free"); !errors.Is(err, ErrModelQuarantine) {
		t.Errorf("quarantine must survive catalog sync, got %v", err)
	}
}
