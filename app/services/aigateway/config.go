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
	"strings"
	"time"

	"github.com/harness/gitness/types"
)

const (
	DefaultBaseURL               = "https://openrouter.ai/api"
	DefaultDecisionModel         = "baseten/mammoth-1:free"
	DefaultDecisionFallbackModel = "baseten/elephant-1:free"

	// freeTierRPM is the per-account limit of OpenRouter free models; we stay below it.
	freeTierRPM = 20
)

type Config struct {
	Enabled               bool
	BaseURL               string
	APIKey                string
	DecisionModel         string
	DecisionFallbackModel string
	DailyRequestLimit     int
	RequestsPerMinute     int
	Replicas              int
	CatalogSyncInterval   time.Duration
	HTTPTimeout           time.Duration
}

func (c *Config) Sanitize() {
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.DecisionModel == "" {
		c.DecisionModel = DefaultDecisionModel
	}
	if c.DecisionFallbackModel == "" {
		c.DecisionFallbackModel = DefaultDecisionFallbackModel
	}
	if c.DailyRequestLimit <= 0 {
		c.DailyRequestLimit = 50
	}
	if c.RequestsPerMinute <= 0 || c.RequestsPerMinute > freeTierRPM {
		c.RequestsPerMinute = freeTierRPM - 2
	}
	if c.Replicas < 1 {
		c.Replicas = 1
	}
	if c.CatalogSyncInterval < time.Hour {
		c.CatalogSyncInterval = 6 * time.Hour
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = time.Minute
	}
}

// perReplicaRPM is the share of the account-wide per-minute limit one replica may use (at least 1).
// The per-minute window is kept in memory, so every replica enforces its own share.
func (c *Config) perReplicaRPM() int {
	return max(1, c.RequestsPerMinute/max(1, c.Replicas))
}

func NewConfig(global *types.Config) Config {
	cfg := Config{
		Enabled:               global.AIGateway.Enabled,
		BaseURL:               global.AIGateway.BaseURL,
		APIKey:                global.AIGateway.APIKey,
		DecisionModel:         global.AIGateway.DecisionModel,
		DecisionFallbackModel: global.AIGateway.DecisionFallbackModel,
		DailyRequestLimit:     global.AIGateway.DailyRequestLimit,
		RequestsPerMinute:     global.AIGateway.RequestsPerMinute,
		Replicas:              global.AIGateway.Replicas,
		CatalogSyncInterval:   global.AIGateway.CatalogSyncInterval,
		HTTPTimeout:           global.AIGateway.HTTPTimeout,
	}

	cfg.Sanitize()

	return cfg
}
