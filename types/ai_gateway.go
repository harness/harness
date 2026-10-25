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

package types

import (
	"encoding/json"

	"github.com/harness/gitness/types/enum"
)

// AIModel is a model of the free OpenRouter catalog together with its observed performance.
type AIModel struct {
	ID                string             `json:"id"` // OpenRouter model id
	Kind              enum.AIModelKind   `json:"kind"`
	ContextLength     int                `json:"context_length"`
	SupportsTools     bool               `json:"supports_tools"`
	Status            enum.AIModelStatus `json:"status"`
	SuccessRateByTask map[string]float64 `json:"success_rate_by_task"`
	P95LatencyMS      int                `json:"p95_latency_ms"`
	LastSeen          int64              `json:"last_seen"`
	QuarantinedReason string             `json:"quarantined_reason,omitempty"`
	Created           int64              `json:"created"`
	Updated           int64              `json:"updated"`
}

// AIModelFilter specifies the filter for listing models. Empty fields are not filtered.
type AIModelFilter struct {
	Kind   enum.AIModelKind
	Status enum.AIModelStatus
}

// AIDecisionAnswer is the answer payload of a decision.
type AIDecisionAnswer struct {
	Value         json.RawMessage    `json:"value"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    float64            `json:"confidence"`
}

// AIDecision is the audit record of a decision made by the AI gateway.
type AIDecision struct {
	ID         int64                   `json:"id"`
	TaskID     *int64                  `json:"task_id,omitempty"`
	QuestionID string                  `json:"question_id"`
	ModelBuild string                  `json:"model_build"`
	StateHash  string                  `json:"state_hash"`
	Answer     AIDecisionAnswer        `json:"answer"`
	Fallback   bool                    `json:"fallback"`
	Outcome    *enum.AIDecisionOutcome `json:"outcome,omitempty"`
	Created    int64                   `json:"created"`
}

// AIQuota is the daily request counter of a quota class.
type AIQuota struct {
	Day      string `json:"day"` // YYYY-MM-DD
	Class    string `json:"class"`
	Used     int    `json:"used"`
	Reserved int    `json:"reserved"`
}

// AIUsage is a usage record of the AI gateway.
type AIUsage struct {
	ID        int64   `json:"id"`
	SpaceID   *int64  `json:"space_id,omitempty"`
	RepoID    *int64  `json:"repo_id,omitempty"`
	Feature   string  `json:"feature"`
	Model     string  `json:"model"`
	Requests  int     `json:"requests"`
	TokensIn  int64   `json:"tokens_in"`
	TokensOut int64   `json:"tokens_out"`
	CostUSD   float64 `json:"cost_usd"`
	Created   int64   `json:"created"`
}
