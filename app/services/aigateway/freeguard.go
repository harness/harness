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
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const (
	// FreeRouterModel is OpenRouter's router that only picks free models.
	FreeRouterModel = "openrouter/free"

	freeSuffix = ":free"
)

var (
	ErrModelNotFree    = errors.New("model is not free")
	ErrModelQuarantine = errors.New("model is quarantined")
	ErrPaidUsage       = errors.New("response reported a non-zero cost")
)

// Pricing mirrors the pricing object of the OpenRouter model catalog.
// OpenRouter encodes prices as decimal strings in USD per token.
type Pricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Request    string `json:"request,omitempty"`
	Image      string `json:"image,omitempty"`
}

// IsZero returns true only if prompt and completion prices are set and every price parses to zero.
// Any unparsable value is treated as paid.
func (p Pricing) IsZero() bool {
	if p.Prompt == "" || p.Completion == "" {
		return false
	}
	for _, v := range []string{p.Prompt, p.Completion, p.Request, p.Image} {
		if v == "" {
			continue
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f != 0 {
			return false
		}
	}
	return true
}

// IsFreeModelID returns true for model IDs that are free by naming convention.
func IsFreeModelID(id string) bool {
	return id == FreeRouterModel || strings.HasSuffix(id, freeSuffix)
}

// ProviderPreferences is sent with every request so OpenRouter never routes to a paid endpoint.
type ProviderPreferences struct {
	MaxPrice MaxPrice `json:"max_price"`
}

type MaxPrice struct {
	Prompt     float64 `json:"prompt"`
	Completion float64 `json:"completion"`
}

// FreeOnlyProvider returns provider preferences with a zero max price.
func FreeOnlyProvider() *ProviderPreferences {
	return &ProviderPreferences{MaxPrice: MaxPrice{Prompt: 0, Completion: 0}}
}

// Guard enforces the free-only rule on every model ID that leaves the gateway.
// Until the catalog is synced only IDs that are free by naming convention are accepted.
type Guard struct {
	mu          sync.RWMutex
	allowed     map[string]struct{}
	quarantined map[string]string
}

func NewGuard() *Guard {
	return &Guard{
		quarantined: map[string]string{},
	}
}

// SetAllowed replaces the allowlist with the free models found in the catalog.
func (g *Guard) SetAllowed(ids []string) {
	allowed := make(map[string]struct{}, len(ids)+1)
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	allowed[FreeRouterModel] = struct{}{}

	g.mu.Lock()
	g.allowed = allowed
	g.mu.Unlock()
}

// Check returns an error if the model must not be used.
func (g *Guard) Check(id string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if reason, ok := g.quarantined[id]; ok {
		return fmt.Errorf("%w: %s (%s)", ErrModelQuarantine, id, reason)
	}

	if g.allowed == nil {
		if !IsFreeModelID(id) {
			return fmt.Errorf("%w: %s", ErrModelNotFree, id)
		}
		return nil
	}

	if _, ok := g.allowed[id]; !ok {
		return fmt.Errorf("%w: %s is not in the free model allowlist", ErrModelNotFree, id)
	}
	return nil
}

// VerifyCost quarantines the model if a response reported any cost.
func (g *Guard) VerifyCost(id string, cost float64) error {
	if cost <= 0 {
		return nil
	}
	g.Quarantine(id, fmt.Sprintf("reported cost %g USD", cost))
	return fmt.Errorf("%w: %s reported %g USD", ErrPaidUsage, id, cost)
}

func (g *Guard) Quarantine(id, reason string) {
	g.mu.Lock()
	g.quarantined[id] = reason
	g.mu.Unlock()
}

func (g *Guard) IsQuarantined(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.quarantined[id]
	return ok
}
