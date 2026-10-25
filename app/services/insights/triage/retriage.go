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
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/types/enum"
)

// Retriage runs the triage of an issue again, synchronously. It is used when a task waited for the AI
// quota (esperando_cuota): the task already exists, so the model answer is recorded on it as decisions
// and a comment. The task priority chosen by the rules is left as it is: a person decides whether the
// model severity changes it.
//
// It returns an error wrapping ErrQuotaExhausted when the quota is exhausted again, so the caller can
// put the task back to waiting, and one wrapping ErrAIUnavailable when the AI can't answer. Nothing is
// changed in both cases. An issue without a task, without events or in a repository with the AI off
// is a no-op.
func (o *Observer) Retriage(ctx context.Context, issueID int64) error {
	issue, err := o.deps.Issues.Find(ctx, issueID)
	if err != nil {
		return fmt.Errorf("failed to find issue: %w", err)
	}
	if issue.TaskID == nil {
		return nil
	}
	repo, err := o.deps.Repos.FindByID(ctx, issue.RepoID)
	if err != nil {
		return fmt.Errorf("failed to find repo: %w", err)
	}
	mode, err := o.deps.Modes.Resolve(ctx, repo)
	if err != nil || !aiAllowed(mode) || o.deps.Decider == nil {
		return nil
	}
	events, err := o.deps.Events.ListByIssue(ctx, issue.ID, 1)
	if err != nil {
		return fmt.Errorf("failed to list issue events: %w", err)
	}
	if len(events) == 0 {
		return nil
	}
	ev := events[0]
	info := parseEvent(ev)

	dups, releases, err := o.scan(ctx, issue, info)
	if err != nil {
		dups, releases = nil, nil
	}
	releases = append(releases, issue.FirstRelease)
	prev := previousRelease(ev.Release, releases...)
	suspects := o.findSuspects(ctx, repo, ev.Release, prev, info, ev.Timestamp)

	state := questions.InsightsState{
		Error: questions.ErrorInfo{
			IssueID: strconv.FormatInt(issue.ID, 10), Type: info.Type, Message: info.Message,
			Platform: issue.Platform, Release: ev.Release, Environment: ev.Environment,
			Frames: frameLabels(info.Frames), EventCount: int(issue.TimesSeen), UsersAffected: int(issue.UsersSeen),
		},
		SuspectCommits: suspects,
	}
	if mode == enum.AIModeFull && len(info.Frames) > 0 {
		state.Error.Code = info.Frames[0].ContextLine
	}
	for _, d := range dups {
		state.Candidates = append(state.Candidates, questions.Candidate{
			IssueID: strconv.FormatInt(d.issue.ID, 10), Type: d.info.Type, Message: d.issue.Title,
			FingerprintDistance: 1 - d.overlap, Frames: frameLabels(d.info.Frames),
			EventCount: int(d.issue.TimesSeen),
		})
	}

	out := o.decide(ctx, mode, issue, &state)
	if out.quota {
		return fmt.Errorf("%w: %s", ErrQuotaExhausted, out.aiFailure)
	}
	if out.aiFailure != "" || out.triage == nil {
		return fmt.Errorf("%w: %s", ErrAIUnavailable, out.aiFailure)
	}

	pid := o.principal()
	taskID := *issue.TaskID
	o.recordDecisions(ctx, pid, taskID, out)
	return o.deps.Tasks.AddComment(ctx, pid, taskID, "Triage re-run after the AI quota was restored: severity "+
		string(out.severity)+", action "+string(out.action)+".")
}
