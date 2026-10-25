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

// Package verify closes the loop of Insights tasks: a resolved task linked to an Insights
// issue becomes verified once the error stopped appearing, and is reopened if it reappears.
package verify

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// DefaultVerifyWindow is how long the release with the fix must be live without the error.
	DefaultVerifyWindow = 24 * time.Hour
	// DefaultQuietPeriod is how long an issue must stay unseen after the task was resolved.
	DefaultQuietPeriod = 72 * time.Hour

	pageSize = 200
)

// Config configures the verification.
type Config struct {
	VerifyWindow time.Duration
	QuietPeriod  time.Duration
}

func (c Config) withDefaults() Config {
	if c.VerifyWindow <= 0 {
		c.VerifyWindow = DefaultVerifyWindow
	}
	if c.QuietPeriod <= 0 {
		c.QuietPeriod = DefaultQuietPeriod
	}
	return c
}

// IssueRef returns the task link reference used for an Insights issue (its numeric id).
func IssueRef(issueID int64) string { return strconv.FormatInt(issueID, 10) }

// Transitioner changes the state of a task; it is implemented by *task.Service.
type Transitioner interface {
	Transition(ctx context.Context, principalID, taskID int64, to enum.TaskState, reason string) (*types.Task, error)
}

// Verifier decides which resolved tasks are verified.
type Verifier struct {
	cfg         Config
	links       store.TaskLinkedRefStore
	issues      store.InsightsIssueStore
	releases    store.InsightsReleaseStore
	transitions Transitioner
	principal   func() int64
	now         func() time.Time
}

// NewVerifier creates a Verifier. principal returns the principal that performs the transitions.
func NewVerifier(
	cfg Config,
	links store.TaskLinkedRefStore,
	issues store.InsightsIssueStore,
	releases store.InsightsReleaseStore,
	transitions Transitioner,
	principal func() int64,
) *Verifier {
	return &Verifier{
		cfg:         cfg.withDefaults(),
		links:       links,
		issues:      issues,
		releases:    releases,
		transitions: transitions,
		principal:   principal,
		now:         time.Now,
	}
}

// Result summarizes a verification pass.
type Result struct {
	Checked  int
	Verified int
}

func (r Result) String() string {
	return fmt.Sprintf("checked %d resolved tasks, verified %d", r.Checked, r.Verified)
}

type taskRefs struct {
	taskID   int64
	resolved int64
	refs     []string
}

// Run verifies all resolved tasks linked to Insights issues.
func (v *Verifier) Run(ctx context.Context) (Result, error) {
	var res Result
	cache := map[int64][]*types.InsightsReleaseInfo{}
	var after int64

	for {
		rows, err := v.links.ListByStateAndLinkKind(
			ctx, enum.TaskStateResolved, enum.TaskLinkKindInsightsIssue, after, pageSize)
		if err != nil {
			return res, fmt.Errorf("failed to list resolved insights tasks: %w", err)
		}
		if len(rows) == 0 {
			return res, nil
		}

		full := len(rows) == pageSize
		groups := groupByTask(rows)
		if full && len(groups) > 1 {
			// the last task may continue on the next page: handle it there.
			groups = groups[:len(groups)-1]
		}
		for _, g := range groups {
			res.Checked++
			ok, reason, cerr := v.check(ctx, g, cache)
			if cerr != nil {
				log.Ctx(ctx).Warn().Err(cerr).Int64("task_id", g.taskID).Msg("insights verify: check failed")
				continue
			}
			if !ok {
				continue
			}
			_, terr := v.transitions.Transition(ctx, v.principal(), g.taskID, enum.TaskStateVerified, reason)
			if terr != nil {
				if errors.Is(terr, task.ErrInvalidTransition) {
					continue // changed concurrently
				}
				log.Ctx(ctx).Warn().Err(terr).Int64("task_id", g.taskID).Msg("insights verify: transition failed")
				continue
			}
			res.Verified++
		}

		if !full {
			return res, nil
		}
		after = groups[len(groups)-1].taskID
	}
}

func groupByTask(rows []*types.TaskLinkedRef) []*taskRefs {
	var out []*taskRefs
	for _, r := range rows {
		if n := len(out); n == 0 || out[n-1].taskID != r.TaskID {
			out = append(out, &taskRefs{taskID: r.TaskID, resolved: r.Resolved})
		}
		last := out[len(out)-1]
		last.refs = append(last.refs, r.RefID)
	}
	return out
}

// check reports whether every issue linked to the task shows evidence of the fix.
func (v *Verifier) check(
	ctx context.Context,
	g *taskRefs,
	cache map[int64][]*types.InsightsReleaseInfo,
) (bool, string, error) {
	var reasons []string
	for _, ref := range g.refs {
		id, err := strconv.ParseInt(strings.TrimSpace(ref), 10, 64)
		if err != nil {
			return false, "", nil // not an issue id we understand
		}
		issue, err := v.issues.Find(ctx, id)
		if err != nil {
			return false, "", fmt.Errorf("failed to find insights issue %d: %w", id, err)
		}
		if issue.Status != enum.InsightsIssueStatusResolved {
			return false, "", nil
		}
		rels, ok := cache[issue.ProjectID]
		if !ok {
			rels, err = v.releases.ListReleases(ctx, issue.ProjectID)
			if err != nil {
				return false, "", fmt.Errorf("failed to list releases: %w", err)
			}
			cache[issue.ProjectID] = rels
		}
		reason, ok := v.evidence(issue, g.resolved, rels)
		if !ok {
			return false, "", nil
		}
		reasons = append(reasons, reason)
	}
	if len(reasons) == 0 {
		return false, "", nil
	}
	return true, strings.Join(reasons, "; "), nil
}

// evidence returns why the resolved issue counts as verified, if it does.
func (v *Verifier) evidence(issue *types.InsightsIssue, taskResolved int64, rels []*types.InsightsReleaseInfo) (string, bool) {
	now := v.now().UnixMilli()

	// Rule 1: a release with the fix is live (seen in project events) for the verify window
	// and the issue has not been seen since it went live.
	if fix := issue.ResolvedInRelease; fix != "" {
		if live, first, ok := firstLive(fix, rels); ok &&
			now-first >= v.cfg.VerifyWindow.Milliseconds() && issue.LastSeen < first {
			return fmt.Sprintf(
				"release %s has been live since %s (%s) and the error was not seen again in %s",
				live, formatTime(first), liveNote(live, fix), v.cfg.VerifyWindow), true
		}
	}

	// Rule 2: not seen at all for the quiet period since the task was resolved.
	base := max(issue.LastSeen, taskResolved)
	if base > 0 && now-base >= v.cfg.QuietPeriod.Milliseconds() {
		return fmt.Sprintf("the error was not seen for %s since %s", v.cfg.QuietPeriod, formatTime(base)), true
	}

	return "", false
}

func liveNote(live, fix string) string {
	if live == fix {
		return "release with the fix"
	}
	return "newer than the fix release " + fix
}

func formatTime(ms int64) string { return time.UnixMilli(ms).UTC().Format(time.RFC3339) }

// firstLive finds the earliest first-event timestamp among the releases equal to or newer than fix.
func firstLive(fix string, rels []*types.InsightsReleaseInfo) (release string, first int64, ok bool) {
	for _, r := range rels {
		if !releaseAtLeast(r.Release, fix) {
			continue
		}
		if !ok || r.FirstSeen < first {
			release, first, ok = r.Release, r.FirstSeen, true
		}
	}
	return release, first, ok
}

// releaseAtLeast reports whether release is the fix release or newer. Releases that are not
// semantic versions only match when they are equal.
func releaseAtLeast(release, fix string) bool {
	if release == fix {
		return true
	}
	c, ok := insights.CompareReleases(release, fix)
	return ok && c > 0
}
