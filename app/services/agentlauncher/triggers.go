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
	"github.com/harness/gitness/types/enum"

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

// ConsiderFinding is the entry point for Shield: a finding whose fix strategy is the agent (or whose
// deterministic bump needs it) is offered to the policy as a fix_dependency launch, or fix_sast for SAST
// findings. decisionID and confidence describe the fix_strategy decision (or the rules decision) that proposes
// the agent. It does nothing when the finding has no task.
func (l *Launcher) ConsiderFinding(
	ctx context.Context,
	repo *types.RepositoryCore,
	f *types.ShieldFinding,
	decisionID int64,
	confidence float64,
) {
	if f == nil || f.TaskID == nil {
		return
	}
	t, err := l.deps.TaskFinder.Find(ctx, *f.TaskID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("finding_id", f.ID).Msg("agent launcher: failed to find the task of the finding")
		return
	}
	kind := KindFixDependency
	if f.Kind == enum.ShieldFindingKindSAST {
		kind = KindFixSAST
	}
	l.ConsiderTrigger(ctx, repo, t, kind,
		agentpolicy.TriggerDecision{ID: decisionID, Value: "agent", Confidence: confidence})
}
