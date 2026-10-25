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

package questions

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/app/services/aigateway/decisions"
)

// Question ids of the Insights error group. Candidate questions are suffixed _1.._3.
const (
	QDuplicatePrefix = "is_duplicate_of_candidate_"
	QSeverity        = "severity"
	QAction          = "action"
	QSuspectCommit   = "suspect_commit"

	MaxDuplicateCandidates = 3
	MaxSuspectCommits      = 5

	// SuspectNone is the suspect_commit option meaning no commit is to blame.
	SuspectNone = "none"

	shortSHALen = 8
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Action string

const (
	ActionIgnore  Action = "ignore"
	ActionNotify  Action = "notify"
	ActionSuggest Action = "suggest"
	ActionFix     Action = "fix"
)

// ErrorInfo is the new error being triaged.
type ErrorInfo struct {
	IssueID     string `json:"issue_id,omitempty"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	Platform    string `json:"platform,omitempty"`
	Release     string `json:"release,omitempty"`
	Environment string `json:"environment,omitempty"`
	// Frames are the top stack frames, innermost first.
	Frames        []string `json:"frames,omitempty"`
	EventCount    int      `json:"event_count"`
	UsersAffected int      `json:"users_affected"`
	// Regression is true when the error was resolved before and came back.
	Regression bool `json:"regression,omitempty"`
	// Code is a short excerpt, kept only when Limits.IncludeCode is true.
	Code string `json:"code,omitempty"`
}

// Candidate is an existing issue with a close fingerprint.
type Candidate struct {
	IssueID string `json:"issue_id"`
	Type    string `json:"type"`
	Message string `json:"message"`
	// FingerprintDistance is 0 for identical and grows with the difference.
	FingerprintDistance float64  `json:"fingerprint_distance"`
	Frames              []string `json:"frames,omitempty"`
	EventCount          int      `json:"event_count"`
	Resolved            bool     `json:"resolved,omitempty"`
}

// SuspectCommit is a commit from git blame over the top frames.
type SuspectCommit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject,omitempty"`
	// Reason says why the commit is suspect, e.g. "touched 3 of the top 5 frames, 2 days before the first event".
	Reason string `json:"reason,omitempty"`
}

// InsightsState is the state of an Insights error triage request.
type InsightsState struct {
	Error          ErrorInfo       `json:"error"`
	Candidates     []Candidate     `json:"candidates,omitempty"`
	SuspectCommits []SuspectCommit `json:"suspect_commits,omitempty"`
}

// ShortSHA lowercases and shortens a commit id to the form used as choice key.
func ShortSHA(sha string) string {
	sha = strings.ToLower(strings.TrimSpace(sha))
	if len(sha) > shortSHALen {
		sha = sha[:shortSHALen]
	}
	return sha
}

// Fit compacts the state in place: it keeps at most 3 candidates and 5 commits, shortens SHAs,
// caps frames and messages and drops code unless l.IncludeCode is set.
func (s *InsightsState) Fit(l Limits) error {
	l = l.normalized()
	if len(s.Candidates) > MaxDuplicateCandidates {
		s.Candidates = s.Candidates[:MaxDuplicateCandidates]
	}
	if len(s.SuspectCommits) > MaxSuspectCommits {
		s.SuspectCommits = s.SuspectCommits[:MaxSuspectCommits]
	}
	s.Error.Message = trunc(s.Error.Message, l.MaxMessageChars)
	s.Error.Frames = truncList(s.Error.Frames, l.MaxFrames, maxFrameChars)
	s.Error.Code = l.code(s.Error.Code)
	for i := range s.Candidates {
		c := &s.Candidates[i]
		c.Message = trunc(c.Message, l.MaxMessageChars/2)
		c.Frames = truncList(c.Frames, min(l.MaxFrames, 4), maxFrameChars)
	}
	for i := range s.SuspectCommits {
		c := &s.SuspectCommits[i]
		c.SHA = ShortSHA(c.SHA)
		c.Subject = trunc(c.Subject, 100)
		c.Reason = trunc(c.Reason, 160)
	}
	each := func(f func(c *Candidate) bool) func() bool {
		return func() bool {
			changed := false
			for i := range s.Candidates {
				changed = f(&s.Candidates[i]) || changed
			}
			return changed
		}
	}
	return fit(l.MaxStateBytes, s,
		func() bool { return shrinkStr(&s.Error.Code, 0) },
		func() bool { return halve(&s.Error.Frames, 3) },
		each(func(c *Candidate) bool { return halve(&c.Frames, 2) }),
		func() bool { return shrinkStr(&s.Error.Message, 200) },
		each(func(c *Candidate) bool { return shrinkStr(&c.Message, 100) }),
		each(func(c *Candidate) bool { return halve(&c.Frames, 0) }),
		func() bool {
			changed := false
			for i := range s.SuspectCommits {
				changed = shrinkStr(&s.SuspectCommits[i].Subject, 40) || changed
				changed = shrinkStr(&s.SuspectCommits[i].Reason, 60) || changed
			}
			return changed
		},
		func() bool { return shrinkStr(&s.Error.Message, 60) },
		each(func(c *Candidate) bool { return shrinkStr(&c.Message, 0) }),
	)
}

// DuplicateQuestionID returns the question id of candidate n (1-based).
func DuplicateQuestionID(n int) string {
	return fmt.Sprintf("%s%d", QDuplicatePrefix, n)
}

// InsightsQuestions builds one batched request for the state: one duplicate question per
// candidate, plus severity, action and, when there are commits, suspect_commit.
// Candidate n of the questions is state.candidates[n-1]; callers must pass the same state to
// ParseInsights.
func InsightsQuestions(s *InsightsState) (map[string]decisions.Question, error) {
	if len(s.Candidates) > MaxDuplicateCandidates {
		return nil, fmt.Errorf("%w: %d candidates, max %d", ErrInvalidInput, len(s.Candidates), MaxDuplicateCandidates)
	}
	if len(s.SuspectCommits) > MaxSuspectCommits {
		return nil, fmt.Errorf("%w: %d commits, max %d", ErrInvalidInput, len(s.SuspectCommits), MaxSuspectCommits)
	}
	qs := map[string]decisions.Question{}
	for i := range s.Candidates {
		qs[DuplicateQuestionID(i+1)] = decisions.Noul(instr(fmt.Sprintf(
			"Is the new error in state.error the same underlying problem as state.candidates[%d]? "+
				"Compare type, message and stack frames; a different root cause is not a duplicate.", i)),
			"Same root cause: the candidate issue already tracks this error.",
			"Different root cause: this error needs its own issue.")
	}
	qs[QSeverity] = decisions.Score(instr(
		"Rate how severe the new error is for the application's users. Weigh event_count, users_affected, "+
			"regression and whether the failing path looks central or optional. "+
			"Levels from lowest: low is cosmetic or rare, medium degrades a feature, "+
			"high breaks a main flow for some users, critical is an outage, data loss or security impact."),
		string(SeverityLow), string(SeverityMedium), string(SeverityHigh), string(SeverityCritical))
	qs[QAction] = decisions.Choice(instr(
		"Pick the right response to the new error."),
		map[string]string{
			string(ActionIgnore):  "Noise: expected, harmless or from a bot, nobody needs to see it.",
			string(ActionNotify):  "A person should know about it, but the cause or fix is unclear.",
			string(ActionSuggest): "A fix idea can be proposed for a human to review.",
			string(ActionFix):     "The cause is clear and local enough for an agent to open a fix pull request.",
		})
	if len(s.SuspectCommits) > 0 {
		opts := map[string]string{SuspectNone: "No listed commit is likely to have caused the error."}
		for _, c := range s.SuspectCommits {
			key := ShortSHA(c.SHA)
			if key == "" || key == SuspectNone {
				return nil, fmt.Errorf("%w: invalid suspect commit sha %q", ErrInvalidInput, c.SHA)
			}
			if _, dup := opts[key]; dup {
				return nil, fmt.Errorf("%w: duplicate suspect commit %q", ErrInvalidInput, key)
			}
			opts[key] = suspectDescription(c)
		}
		qs[QSuspectCommit] = decisions.Choice(instr(
			"Pick the commit that most likely introduced the new error, using the reasons given, "+
				"or none when no commit is likely."), opts)
	}
	return qs, nil
}

func suspectDescription(c SuspectCommit) string {
	d := strings.TrimSpace(c.Reason)
	if d == "" {
		d = strings.TrimSpace(c.Subject)
	}
	if d == "" {
		d = "Commit " + ShortSHA(c.SHA)
	}
	return "Commit " + ShortSHA(c.SHA) + ": " + trunc(d, 200)
}

// DuplicateMatch is the answer for one candidate.
type DuplicateMatch struct {
	Candidate   int // 1-based
	IssueID     string
	IsDuplicate bool
	Probability float64
	Confidence  float64
}

// InsightsTriage is the typed result of InsightsQuestions.
type InsightsTriage struct {
	Duplicates []DuplicateMatch
	// BestDuplicate is the duplicate with the highest probability, nil when none is at 0.5 or more.
	BestDuplicate *DuplicateMatch

	Severity           Severity
	SeverityScore      float64 // expected level index, 0 (low) to 3 (critical)
	SeverityConfidence float64

	Action           Action
	ActionConfidence float64

	// SuspectCommit is the chosen short SHA, empty when none or when no commits were offered.
	SuspectCommit           string
	SuspectCommitConfidence float64
}

// ParseInsights maps decisions to an InsightsTriage for the state used to build the questions.
func ParseInsights(s *InsightsState, ds map[string]decisions.Decision) (InsightsTriage, error) {
	var t InsightsTriage
	for i, c := range s.Candidates {
		n := i + 1
		d, err := need(ds, DuplicateQuestionID(n), decisions.TypeNoul)
		if err != nil {
			return InsightsTriage{}, err
		}
		m := DuplicateMatch{
			Candidate: n, IssueID: c.IssueID, IsDuplicate: d.Value == "true",
			Probability: d.Probability, Confidence: d.Confidence,
		}
		t.Duplicates = append(t.Duplicates, m)
		if m.IsDuplicate && (t.BestDuplicate == nil || m.Probability > t.BestDuplicate.Probability) {
			best := m
			t.BestDuplicate = &best
		}
	}

	sev, err := needOneOf(ds, QSeverity, decisions.TypeScore,
		string(SeverityLow), string(SeverityMedium), string(SeverityHigh), string(SeverityCritical))
	if err != nil {
		return InsightsTriage{}, err
	}
	t.Severity, t.SeverityScore, t.SeverityConfidence = Severity(sev.Value), sev.Score, sev.Confidence

	act, err := needOneOf(ds, QAction, decisions.TypeChoice,
		string(ActionIgnore), string(ActionNotify), string(ActionSuggest), string(ActionFix))
	if err != nil {
		return InsightsTriage{}, err
	}
	t.Action, t.ActionConfidence = Action(act.Value), act.Confidence

	if len(s.SuspectCommits) > 0 {
		allowed := []string{SuspectNone}
		for _, c := range s.SuspectCommits {
			allowed = append(allowed, ShortSHA(c.SHA))
		}
		sc, err := needOneOf(ds, QSuspectCommit, decisions.TypeChoice, allowed...)
		if err != nil {
			return InsightsTriage{}, err
		}
		if sc.Value != SuspectNone {
			t.SuspectCommit = sc.Value
		}
		t.SuspectCommitConfidence = sc.Confidence
	}
	return t, nil
}
