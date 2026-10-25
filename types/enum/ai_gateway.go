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

package enum

// AIModelKind defines what a catalog model is used for.
type AIModelKind string

const (
	AIModelKindDecision AIModelKind = "decision"
	AIModelKindChat     AIModelKind = "chat"
	AIModelKindEmbed    AIModelKind = "embed"
)

// AIModelStatus defines the health state of a catalog model.
type AIModelStatus string

const (
	AIModelStatusActive      AIModelStatus = "active"
	AIModelStatusDegraded    AIModelStatus = "degraded"
	AIModelStatusQuarantined AIModelStatus = "quarantined"
	AIModelStatusRetired     AIModelStatus = "retired"
)

// AIDecisionOutcome defines the observed result of an AI decision.
type AIDecisionOutcome string

const (
	AIDecisionOutcomeSuccess    AIDecisionOutcome = "success"
	AIDecisionOutcomeFailure    AIDecisionOutcome = "failure"
	AIDecisionOutcomeOverridden AIDecisionOutcome = "overridden"
)
