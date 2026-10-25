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

package agentlauncher

import (
	"context"

	"github.com/harness/gitness/app/services/agentpolicy"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// AutofixCandidate implements triage.AutofixObserver: an Insights task that the triage marked as
// autofix_candidate is offered to the policy as a fix_error launch.
func (l *Launcher) AutofixCandidate(ctx context.Context, repo *types.RepositoryCore, t *types.Task, confidence float64) {
	l.ConsiderTrigger(ctx, repo, t, KindFixError, agentpolicy.TriggerDecision{Value: "fix", Confidence: confidence})
}

// ConsiderTrigger is the fire-and-forget entry point for trigger hooks (for example Shield findings whose
// fix_strategy is agent: pass KindFixDependency or KindFixSAST and the strategy decision). Errors are logged.
func (l *Launcher) ConsiderTrigger(
	ctx context.Context,
	repo *types.RepositoryCore,
	t *types.Task,
	kind Kind,
	trigger agentpolicy.TriggerDecision,
) {
	res, err := l.Consider(ctx, ConsiderInput{Task: t, Repo: repo, Kind: kind, TriggerDecision: trigger})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task_id", t.ID).Msg("agent launcher: failed to consider task")
		return
	}
	log.Ctx(ctx).Debug().Int64("task_id", t.ID).Str("outcome", string(res.Outcome)).Msg("agent launcher: considered task")
}
