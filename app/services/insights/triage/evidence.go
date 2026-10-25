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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/insights/ingest"
	"github.com/harness/gitness/app/services/insights/suspect"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	topFramesForOverlap = 5
	minFrameOverlap     = 0.34
	latestEventsLimit   = 1
)

// errorInfo is what triage needs from a stored event payload.
type errorInfo struct {
	Type    string
	Message string
	// Frames are innermost first; only in-app frames when the SDK flagged any.
	Frames []ingest.Frame
}

// parseEvent extracts the outermost exception with a stacktrace from the stored event JSON.
// A payload that doesn't parse yields an empty info (rules and message-only triage still work).
func parseEvent(ev *types.InsightsEvent) errorInfo {
	info := errorInfo{Message: ev.Message}
	if ev.Payload == "" {
		return info
	}
	var parsed ingest.Event
	if err := json.Unmarshal([]byte(ev.Payload), &parsed); err != nil {
		return info
	}
	vals := parsed.ExceptionValues()
	if len(vals) == 0 {
		return info
	}
	pick := vals[len(vals)-1]
	for i := len(vals) - 1; i >= 0; i-- {
		if vals[i].Stacktrace != nil && len(vals[i].Stacktrace.Frames) > 0 {
			pick = vals[i]
			break
		}
	}
	info.Type = pick.Type
	if info.Message == "" {
		info.Message = pick.Value
	}
	if pick.Stacktrace == nil {
		return info
	}
	frames := pick.Stacktrace.Frames
	anyInApp := false
	for _, f := range frames {
		if f.InApp != nil && *f.InApp {
			anyInApp = true
			break
		}
	}
	for i := len(frames) - 1; i >= 0; i-- {
		f := frames[i]
		if anyInApp && (f.InApp == nil || !*f.InApp) {
			continue
		}
		info.Frames = append(info.Frames, f)
	}
	return info
}

func frameLoc(f ingest.Frame) string {
	for _, s := range []string{f.Module, f.Filename, f.AbsPath} {
		if s != "" {
			return s
		}
	}
	return ""
}

func frameKey(f ingest.Frame) string { return frameLoc(f) + "|" + f.Function }

func frameLabel(f ingest.Frame) string {
	s := f.Function
	if loc := frameLoc(f); loc != "" {
		if f.Lineno > 0 {
			loc = fmt.Sprintf("%s:%d", loc, int(f.Lineno))
		}
		if s != "" {
			return s + " (" + loc + ")"
		}
		return loc
	}
	return s
}

func frameLabels(frames []ingest.Frame) []string {
	out := make([]string, 0, len(frames))
	for _, f := range frames {
		if l := frameLabel(f); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func topKeys(frames []ingest.Frame) map[string]struct{} {
	keys := map[string]struct{}{}
	for i, f := range frames {
		if i >= topFramesForOverlap {
			break
		}
		if k := frameKey(f); k != "|" {
			keys[k] = struct{}{}
		}
	}
	return keys
}

// overlap is the Jaccard index of the top in-app frames.
func overlap(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}

type dupCandidate struct {
	issue   *types.InsightsIssue
	info    errorInfo
	overlap float64
}

// scan walks the open issues of the project once. It returns up to questions.MaxDuplicateCandidates
// duplicate candidates (same exception type and overlapping in-app frames, best overlap first) and
// the release names it saw, used to find the previous release.
func (o *Observer) scan(
	ctx context.Context,
	issue *types.InsightsIssue,
	info errorInfo,
) (dups []dupCandidate, releases []string, err error) {
	list, err := o.deps.Issues.List(ctx, issue.RepoID, &types.InsightsIssueFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: o.cfg.MaxScanIssues}},
		Statuses:        []enum.InsightsIssueStatus{enum.InsightsIssueStatusUnresolved},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list issues: %w", err)
	}
	mine := topKeys(info.Frames)
	for _, other := range list {
		if other.ProjectID != issue.ProjectID {
			continue
		}
		releases = append(releases, other.FirstRelease, other.LastRelease)
		if other.ID == issue.ID || info.Type == "" || len(mine) == 0 {
			continue
		}
		evs, lerr := o.deps.Events.ListByIssue(ctx, other.ID, latestEventsLimit)
		if lerr != nil || len(evs) == 0 {
			continue
		}
		oi := parseEvent(evs[0])
		if oi.Type != info.Type {
			continue
		}
		if ov := overlap(mine, topKeys(oi.Frames)); ov >= minFrameOverlap {
			dups = append(dups, dupCandidate{issue: other, info: oi, overlap: ov})
		}
	}
	sort.SliceStable(dups, func(a, b int) bool {
		if dups[a].overlap != dups[b].overlap {
			return dups[a].overlap > dups[b].overlap
		}
		return dups[a].issue.ID < dups[b].issue.ID
	})
	if len(dups) > questions.MaxDuplicateCandidates {
		dups = dups[:questions.MaxDuplicateCandidates]
	}
	return dups, releases, nil
}

// previousRelease returns the highest known release below current, empty when it can't be told.
func previousRelease(current string, known ...string) string {
	best := ""
	for _, r := range known {
		if r == "" || r == current {
			continue
		}
		if c, ok := insights.CompareReleases(r, current); !ok || c >= 0 {
			continue
		}
		if best == "" {
			best = r
			continue
		}
		if c, ok := insights.CompareReleases(r, best); ok && c > 0 {
			best = r
		}
	}
	return best
}

// findSuspects returns at most questions.MaxSuspectCommits commits. Without a known release the
// default branch is blamed. Failures are not fatal: triage continues without commits.
func (o *Observer) findSuspects(
	ctx context.Context,
	repo *types.RepositoryCore,
	release, previous string,
	info errorInfo,
	eventTS int64,
) []questions.SuspectCommit {
	if o.deps.Suspects == nil || len(info.Frames) == 0 {
		return nil
	}
	ref := release
	if ref == "" {
		ref, previous = repo.DefaultBranch, ""
	}
	if ref == "" {
		return nil
	}
	in := suspect.Input{ReleaseSHA: ref, PreviousSHA: previous}
	for _, f := range info.Frames {
		in.Frames = append(in.Frames, suspect.Frame{
			Filename: f.Filename, AbsPath: f.AbsPath, Module: f.Module, Function: f.Function, Lineno: int(f.Lineno),
		})
	}
	cands, err := o.deps.Suspects.Find(ctx, repo, in)
	if err != nil && previous != "" {
		// release range unknown to git: blame only
		in.PreviousSHA = ""
		cands, err = o.deps.Suspects.Find(ctx, repo, in)
	}
	if err != nil {
		return nil
	}
	if len(cands) > questions.MaxSuspectCommits {
		cands = cands[:questions.MaxSuspectCommits]
	}
	out := make([]questions.SuspectCommit, 0, len(cands))
	for _, c := range cands {
		out = append(out, questions.SuspectCommit{
			SHA: c.Commit.SHA, Subject: c.Commit.Title, Reason: suspectReason(c, eventTS),
		})
	}
	return out
}

func suspectReason(c suspect.Candidate, eventTS int64) string {
	kinds := map[string]struct{}{}
	for _, r := range c.Reasons {
		kinds[r.Kind] = struct{}{}
	}
	names := make([]string, 0, len(kinds))
	for k := range kinds {
		names = append(names, strings.ReplaceAll(k, "_", " "))
	}
	sort.Strings(names)
	s := fmt.Sprintf("hit %d stack frame(s): %s", c.FramesHit, strings.Join(names, ", "))
	if !c.Commit.When.IsZero() && eventTS > 0 {
		days := time.UnixMilli(eventTS).Sub(c.Commit.When).Hours() / 24
		if days >= 0 {
			s += fmt.Sprintf("; committed %d day(s) before the event", int(days))
		}
	}
	return s
}
