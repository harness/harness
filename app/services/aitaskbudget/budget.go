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

// Package aitaskbudget accounts the model requests an AI task (fix agent) may issue.
package aitaskbudget

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"
)

// ErrBudgetExhausted is returned by ChargeRequest when the task has no requests left.
// The caller MUST stop the agent and move the work to human review (see section 4.4 of
// docs/design/openrouter-autonomous-agents.md); the charge is not applied in that case.
var ErrBudgetExhausted = store.ErrAITaskBudgetExhausted

// Service charges requests against the per-task budget.
type Service struct {
	aiTaskStore store.AITaskStore
}

// NewService creates a budget service.
func NewService(aiTaskStore store.AITaskStore) *Service {
	return &Service{aiTaskStore: aiTaskStore}
}

// ChargeRequest consumes one request of the budget of the given AI task and returns the
// number of requests remaining after the charge. The increment is a single conditional UPDATE,
// so concurrent callers can never exceed the budget. It returns ErrBudgetExhausted when the
// budget is used up.
func (s *Service) ChargeRequest(ctx context.Context, aiTaskID int64) (int, error) {
	remaining, err := s.aiTaskStore.ChargeRequest(ctx, aiTaskID)
	if err != nil {
		return 0, fmt.Errorf("failed to charge request for ai task %d: %w", aiTaskID, err)
	}
	return remaining, nil
}
