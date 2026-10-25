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

package system

import (
	"context"
	"time"

	"github.com/harness/gitness/app/services/aigateway"
)

// AIModel is a free model of the AI gateway catalog.
type AIModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
	SupportsTools bool   `json:"supports_tools"`
	Quarantined   bool   `json:"quarantined"`
}

// AIModelsOutput is the response of the AI models endpoint.
type AIModelsOutput struct {
	Enabled  bool       `json:"enabled"`
	SyncedAt *time.Time `json:"synced_at,omitempty"`
	Models   []AIModel  `json:"models"`
}

// AIQuotaOutput is the response of the AI quota endpoint.
type AIQuotaOutput struct {
	Enabled        bool           `json:"enabled"`
	Day            string         `json:"day,omitempty"`
	Limit          int            `json:"limit"`
	Used           int            `json:"used"`
	UsedByClass    map[string]int `json:"used_by_class"`
	RemainingClass map[string]int `json:"remaining_by_class"`
}

var aiClasses = []aigateway.Class{
	aigateway.ClassDecision,
	aigateway.ClassTriage,
	aigateway.ClassAgent,
	aigateway.ClassManual,
}

// AIModels returns the free model catalog. When the gateway is disabled it returns enabled=false and no data.
func (c *Controller) AIModels(_ context.Context) AIModelsOutput {
	if !c.aiGateway.Enabled() {
		return AIModelsOutput{Models: []AIModel{}}
	}
	return mapAIModels(c.aiGateway.Catalog(), c.aiGateway.Guard())
}

// AIQuota returns the quota planner snapshot for today. When the gateway is disabled it returns enabled=false.
func (c *Controller) AIQuota(_ context.Context) AIQuotaOutput {
	if !c.aiGateway.Enabled() {
		return AIQuotaOutput{UsedByClass: map[string]int{}, RemainingClass: map[string]int{}}
	}
	return mapAIQuota(c.aiGateway.Quota(), c.aiGateway.Planner().Remaining)
}

func mapAIModels(catalog *aigateway.Catalog, guard *aigateway.Guard) AIModelsOutput {
	models := catalog.Models()
	out := AIModelsOutput{Enabled: true, Models: make([]AIModel, 0, len(models))}
	if t := catalog.SyncedAt(); !t.IsZero() {
		out.SyncedAt = &t
	}
	for _, m := range models {
		out.Models = append(out.Models, AIModel{
			ID:            m.ID,
			Name:          m.Name,
			ContextLength: m.ContextLength,
			SupportsTools: m.SupportsTools(),
			Quarantined:   guard.IsQuarantined(m.ID),
		})
	}
	return out
}

func mapAIQuota(snap aigateway.QuotaSnapshot, remaining func(aigateway.Class) int) AIQuotaOutput {
	out := AIQuotaOutput{
		Enabled:        true,
		Day:            snap.Day,
		Limit:          snap.Limit,
		Used:           snap.Used,
		UsedByClass:    make(map[string]int, len(aiClasses)),
		RemainingClass: make(map[string]int, len(aiClasses)),
	}
	for _, class := range aiClasses {
		out.UsedByClass[string(class)] = snap.UsedClass[class]
		out.RemainingClass[string(class)] = remaining(class)
	}
	return out
}
