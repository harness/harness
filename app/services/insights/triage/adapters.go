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

package triage

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/insights/suspect"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

// GatewayDecider adapts the AI gateway to Decider using confidence escalation on the key question.
type GatewayDecider struct {
	Gateway *aigateway.Gateway
}

var _ Decider = GatewayDecider{}

// Decide implements Decider.
func (d GatewayDecider) Decide(ctx context.Context, req DecideRequest) (*DecideResult, error) {
	if !d.Gateway.Enabled() {
		return nil, ErrAIUnavailable
	}
	res, err := d.Gateway.DecideWithEscalation(ctx, aigateway.EscalationRequest{
		SessionID: req.SessionID, State: req.State, Questions: req.Questions, KeyQuestion: req.KeyQuestion,
	})
	if err != nil {
		if errors.Is(err, aigateway.ErrQuotaExhausted) || aigateway.IsRateLimited(err) {
			return nil, fmt.Errorf("%w: %w", ErrQuotaExhausted, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrAIUnavailable, err)
	}
	final := res.First
	if res.Second != nil {
		final = res.Second
	}
	return &DecideResult{
		Decisions: final.Decisions, Model: final.Model, Fallback: final.Fallback,
		Human: res.Outcome == aigateway.OutcomeHuman,
	}, nil
}

// GitSuspectFinder finds suspect commits through the git service.
type GitSuspectFinder struct {
	Git git.Interface
}

var _ SuspectFinder = GitSuspectFinder{}

// Find implements SuspectFinder.
func (f GitSuspectFinder) Find(
	ctx context.Context,
	repo *types.RepositoryCore,
	in suspect.Input,
) ([]suspect.Candidate, error) {
	return suspect.Find(ctx, suspect.NewGitSource(f.Git, repo.GitUID), in, suspect.Options{})
}
