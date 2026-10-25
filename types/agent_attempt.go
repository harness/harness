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

package types

import "time"

// AgentAttemptState is the state of an agent attempt.
type AgentAttemptState string

const (
	AgentAttemptRunning   AgentAttemptState = "running"
	AgentAttemptSucceeded AgentAttemptState = "succeeded"
	AgentAttemptFailed    AgentAttemptState = "failed"
)

// AgentAttempt is a single run of an agent for a task key.
type AgentAttempt struct {
	ID       int64             `json:"id"`
	RepoID   int64             `json:"repo_id"`
	TaskKey  string            `json:"task_key"`
	Model    string            `json:"model,omitempty"`
	State    AgentAttemptState `json:"state"`
	Started  int64             `json:"started"`
	Finished *int64            `json:"finished,omitempty"`
	Error    string            `json:"error,omitempty"`
}

// AgentAttemptStart are the inputs of an atomic attempt reservation.
type AgentAttemptStart struct {
	RepoID  int64
	TaskKey string
	Model   string
	// MaxActive is the maximum number of running attempts of the repo.
	MaxActive int
	// Cooldown is the minimum time between the starts of attempts of the same task key.
	Cooldown time.Duration
	// MaxFailed is the number of failed attempts of the task key after which it is refused.
	MaxFailed int
	// At is the start time of the attempt.
	At time.Time
}

// Deny reasons returned by AgentAttemptStore.TryStart. They match the agentguard reasons.
const (
	AgentAttemptDenyTooManyFailures = "too_many_failed_attempts"
	AgentAttemptDenyCooldown        = "cooldown_active"
	AgentAttemptDenyMaxActive       = "max_active_agents_reached"
)
