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

package sast

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Severity is the normalized severity of a SAST finding.
type Severity string

// Severity values.
const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Finding is a normalized SAST result.
type Finding struct {
	RuleID      string
	Title       string
	Description string
	Severity    Severity
	// Score is a CVSS-like base score in [0, 10].
	Score       float64
	CWEs        []string // e.g. "CWE-79"
	Path        string
	StartLine   int
	EndLine     int
	Snippet     string
	Fingerprint string
	Tool        string
	ToolVersion string
}

var cweRe = regexp.MustCompile(`(?i)\bCWE-(\d+)`)

// Findings maps all results of the report to normalized findings, in report order.
// Results without a rule id are skipped.
func (r *Report) Findings() []Finding {
	var out []Finding
	for _, run := range r.Runs {
		rules := make(map[string]*Rule, len(run.Tool.Driver.Rules))
		for i := range run.Tool.Driver.Rules {
			rules[run.Tool.Driver.Rules[i].ID] = &run.Tool.Driver.Rules[i]
		}
		for i := range run.Results {
			res := &run.Results[i]
			rule := lookupRule(run.Tool.Driver.Rules, rules, res)
			if f, ok := buildFinding(run.Tool.Driver, rule, res); ok {
				out = append(out, f)
			}
		}
	}
	return out
}

func lookupRule(list []Rule, byID map[string]*Rule, res *Result) *Rule {
	if res.RuleID != "" {
		if r, ok := byID[res.RuleID]; ok {
			return r
		}
	}
	if res.RuleIndex != nil && *res.RuleIndex >= 0 && *res.RuleIndex < len(list) {
		return &list[*res.RuleIndex]
	}
	return nil
}

func buildFinding(d Driver, rule *Rule, res *Result) (Finding, bool) {
	ruleID := res.RuleID
	if ruleID == "" && rule != nil {
		ruleID = rule.ID
	}
	if ruleID == "" {
		return Finding{}, false
	}

	f := Finding{RuleID: ruleID, Tool: d.Name, ToolVersion: d.Version}

	f.Title = res.Message.Text
	if rule != nil {
		f.Title = firstNonEmpty(rule.ShortDescription.Text, rule.Name, res.Message.Text)
		f.Description = firstNonEmpty(rule.FullDescription.Text, rule.Help.Text, res.Message.Text)
	}
	f.Title = firstNonEmpty(f.Title, ruleID)
	f.Description = firstNonEmpty(f.Description, res.Message.Text)

	sev := res.Properties.SecuritySeverity
	if sev == nil && rule != nil {
		sev = rule.Properties.SecuritySeverity
	}
	f.Severity, f.Score = severityAndScore(sev, res.Level)

	f.CWEs = collectCWEs(res, rule)

	if len(res.Locations) > 0 {
		pl := res.Locations[0].PhysicalLocation
		f.Path = normalizePath(pl.ArtifactLocation.URI)
		f.StartLine = pl.Region.StartLine
		f.EndLine = max(pl.Region.EndLine, pl.Region.StartLine)
		f.Snippet = pl.Region.Snippet.Text
	}

	f.Fingerprint = fingerprint(f, res)
	return f, true
}

// severityAndScore maps security-severity (0-10) to a severity bucket and score;
// without it the SARIF level is used with representative scores.
func severityAndScore(sev *float64, level string) (Severity, float64) {
	if sev != nil {
		score := math.Max(0, math.Min(10, *sev))
		switch {
		case score >= 9:
			return SeverityCritical, score
		case score >= 7:
			return SeverityHigh, score
		case score >= 4:
			return SeverityMedium, score
		default:
			return SeverityLow, score
		}
	}
	switch strings.ToLower(level) {
	case "error":
		return SeverityHigh, 8.0
	case "warning":
		return SeverityMedium, 5.0
	default: // note, none, empty
		return SeverityLow, 2.0
	}
}

func collectCWEs(res *Result, rule *Rule) []string {
	var src []string
	src = append(src, res.Properties.CWE...)
	src = append(src, res.Properties.Tags...)
	if rule != nil {
		src = append(src, rule.Properties.CWE...)
		src = append(src, rule.Properties.Tags...)
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range src {
		for _, m := range cweRe.FindAllStringSubmatch(s, -1) {
			id := "CWE-" + m[1]
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}

// normalizePath turns a SARIF artifact URI into a repo-relative slash path.
func normalizePath(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.Path != "" && (u.Scheme == "" || u.Scheme == "file") {
		uri = u.Path
	}
	uri = strings.TrimPrefix(uri, "./")
	return strings.TrimPrefix(uri, "/")
}

// fingerprint is rule id + path + a hash of the whitespace-normalized snippet; line numbers are
// deliberately excluded so that moving code keeps the identity. Without a snippet it falls back
// to the tool-provided partialFingerprints, then fingerprints, then the message.
func fingerprint(f Finding, res *Result) string {
	var disc string
	switch {
	case strings.TrimSpace(f.Snippet) != "":
		disc = hashString(normalizeSnippet(f.Snippet))
	case len(res.PartialFingerprints) > 0:
		disc = hashMap(res.PartialFingerprints)
	case len(res.Fingerprints) > 0:
		disc = hashMap(res.Fingerprints)
	default:
		disc = hashString(normalizeSnippet(res.Message.Text))
	}
	return f.RuleID + "|" + f.Path + "|" + disc
}

func normalizeSnippet(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

func hashMap(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k + "=" + m[k] + ";")
	}
	return hashString(b.String())
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
