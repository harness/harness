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
	"fmt"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"
)

// Model is a free model from the OpenRouter catalog.
type Model struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	ContextLength       int          `json:"context_length"`
	Pricing             Pricing      `json:"pricing"`
	SupportedParameters []string     `json:"supported_parameters"`
	Architecture        Architecture `json:"architecture"`
}

type Architecture struct {
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
}

func (m Model) IsFree() bool {
	return m.Pricing.IsZero()
}

func (m Model) SupportsTools() bool {
	return slices.Contains(m.SupportedParameters, "tools")
}

func (m Model) SupportsStructuredOutputs() bool {
	return slices.Contains(m.SupportedParameters, "structured_outputs")
}

// Requirements are the hard filters applied before Mammoth chooses a model.
type Requirements struct {
	MinContext int
	Tools      bool
}

// Catalog keeps the set of free models and feeds the guard allowlist.
type Catalog struct {
	client *httpClient
	guard  *Guard

	mu       sync.RWMutex
	models   map[string]Model
	syncedAt time.Time
}

func NewCatalog(client *httpClient, guard *Guard) *Catalog {
	return &Catalog{
		client: client,
		guard:  guard,
		models: map[string]Model{},
	}
}

// Sync fetches the catalog and keeps only models whose pricing is zero.
// Listing models is not a model request, so it is not charged to the quota planner.
func (c *Catalog) Sync(ctx context.Context) (int, error) {
	var out struct {
		Data []Model `json:"data"`
	}
	if err := c.client.doJSON(ctx, http.MethodGet, "/v1/models", nil, &out); err != nil {
		return 0, fmt.Errorf("failed to fetch model catalog: %w", err)
	}

	models := make(map[string]Model)
	ids := make([]string, 0)
	for _, m := range out.Data {
		if !m.IsFree() {
			continue
		}
		models[m.ID] = m
		ids = append(ids, m.ID)
	}

	c.mu.Lock()
	c.models = models
	c.syncedAt = time.Now()
	c.mu.Unlock()

	c.guard.SetAllowed(ids)

	return len(models), nil
}

func (c *Catalog) Get(id string) (Model, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.models[id]
	return m, ok
}

func (c *Catalog) SyncedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.syncedAt
}

// Candidates returns the free, non-quarantined models that satisfy the requirements, sorted by ID.
func (c *Catalog) Candidates(req Requirements) []Model {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]Model, 0, len(c.models))
	for _, m := range c.models {
		if m.ID == FreeRouterModel {
			continue
		}
		if req.MinContext > 0 && m.ContextLength < req.MinContext {
			continue
		}
		if req.Tools && !m.SupportsTools() {
			continue
		}
		if c.guard.IsQuarantined(m.ID) {
			continue
		}
		res = append(res, m)
	}

	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}
