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
	"github.com/harness/gitness/app/services/aigateway/decisions"
)

// Question ids of the Shield finding group.
const (
	QReachable     = "reachable"
	QFalsePositive = "false_positive"
	QFixStrategy   = "fix_strategy"
)

type Reachability string

const (
	ReachabilityReachable    Reachability = "reachable"
	ReachabilityNotReachable Reachability = "not_reachable"
	ReachabilityUnknown      Reachability = "unknown"
)

type FixStrategy string

const (
	FixDeterministicBump FixStrategy = "deterministic_bump"
	FixAgent             FixStrategy = "agent"
	FixManual            FixStrategy = "manual"
	FixAcceptRisk        FixStrategy = "accept_risk"
)

// ShieldState describes one Shield finding. It carries metadata and counts only.
type ShieldState struct {
	FindingID string `json:"finding_id,omitempty"`
	// Kind is the finding class, e.g. "vulnerability", "secret" or "license".
	Kind      string  `json:"kind"`
	Package   string  `json:"package,omitempty"`
	Ecosystem string  `json:"ecosystem,omitempty"`
	Version   string  `json:"version,omitempty"`
	FixedIn   string  `json:"fixed_in,omitempty"`
	CVE       string  `json:"cve,omitempty"`
	CVSS      float64 `json:"cvss,omitempty"`
	EPSS      float64 `json:"epss,omitempty"`
	KEV       bool    `json:"kev,omitempty"`
	Severity  string  `json:"severity,omitempty"`
	Summary   string  `json:"summary,omitempty"`
	// Manifest is the path of the file that declares the dependency.
	Manifest         string `json:"manifest,omitempty"`
	DirectDependency bool   `json:"direct_dependency"`
	// BumpIsMajor tells whether the fixed version is a major bump.
	BumpIsMajor bool `json:"bump_is_major,omitempty"`
	// ImportCount is how many source files import the package; -1 means unknown.
	ImportCount int `json:"import_count"`
	// CallSites are short "file:line symbol" strings of affected API usage (top N).
	CallSites []string `json:"call_sites,omitempty"`
	// DiffStats describes the expected fix size, e.g. "1 file, +1 -1".
	DiffStats string `json:"diff_stats,omitempty"`
	// Code is a short excerpt, kept only when Limits.IncludeCode is true.
	Code string `json:"code,omitempty"`
}

// Fit compacts the state in place to fit the byte budget of l.
func (s *ShieldState) Fit(l Limits) error {
	l = l.normalized()
	s.Summary = trunc(s.Summary, l.MaxMessageChars)
	s.CallSites = truncList(s.CallSites, l.MaxFrames, maxFrameChars)
	s.Code = l.code(s.Code)
	return fit(l.MaxStateBytes, s,
		func() bool { return shrinkStr(&s.Code, 0) },
		func() bool { return halve(&s.CallSites, 1) },
		func() bool { return shrinkStr(&s.Summary, 200) },
		func() bool { return shrinkStr(&s.Summary, 60) },
		func() bool { return shrinkStr(&s.DiffStats, 80) },
		func() bool { return shrinkStr(&s.Manifest, 80) },
		func() bool { return shrinkStr(&s.Summary, 0) },
	)
}

// ShieldQuestions returns the three Shield questions for one batched request.
func ShieldQuestions() map[string]decisions.Question {
	return map[string]decisions.Question{
		QReachable: decisions.Choice(instr(
			"Decide whether the vulnerable code of this finding can be reached by the application. "+
				"Use import_count, call_sites and direct_dependency. Pick unknown when the state does not show it."),
			map[string]string{
				string(ReachabilityReachable): "The application imports the package and calls the affected API, " +
					"or the finding is a secret or other issue that is always exposed.",
				string(ReachabilityNotReachable): "The package is unused, only a dev or test dependency, " +
					"or the affected API is not called.",
				string(ReachabilityUnknown): "The state does not allow telling whether the affected code is reachable.",
			}),
		QFalsePositive: decisions.Noul(instr(
			"Is this finding a false positive, meaning it does not describe a real problem in this repository? "+
				"A finding with a known fix version, a high CVSS or a KEV flag is rarely a false positive."),
			"The finding is a false positive: the version is not affected, the match is wrong or the data is a placeholder.",
			"The finding is real: the affected version or secret is actually present."),
		QFixStrategy: decisions.Choice(instr(
			"Pick the cheapest safe way to resolve this finding. Prefer a deterministic bump when fixed_in exists "+
				"and bump_is_major is false."),
			map[string]string{
				string(FixDeterministicBump): "A version bump to fixed_in is enough and is not a major bump, " +
					"so no code change is expected.",
				string(FixAgent): "The fix needs code changes of limited size, such as a major bump with small API changes.",
				string(FixManual): "The fix is risky, large or needs a human decision, " +
					"for example a leaked secret to rotate.",
				string(FixAcceptRisk): "The risk is low or unreachable and the fix is not worth the cost.",
			}),
	}
}

// ShieldTriage is the typed result of ShieldQuestions.
type ShieldTriage struct {
	Reachable           Reachability
	ReachableConfidence float64

	FalsePositive            bool
	FalsePositiveProbability float64
	FalsePositiveConfidence  float64

	FixStrategy           FixStrategy
	FixStrategyConfidence float64
}

// ParseShield maps decisions to a ShieldTriage; every key is required.
func ParseShield(ds map[string]decisions.Decision) (ShieldTriage, error) {
	r, err := needOneOf(ds, QReachable, decisions.TypeChoice,
		string(ReachabilityReachable), string(ReachabilityNotReachable), string(ReachabilityUnknown))
	if err != nil {
		return ShieldTriage{}, err
	}
	fp, err := need(ds, QFalsePositive, decisions.TypeNoul)
	if err != nil {
		return ShieldTriage{}, err
	}
	fs, err := needOneOf(ds, QFixStrategy, decisions.TypeChoice,
		string(FixDeterministicBump), string(FixAgent), string(FixManual), string(FixAcceptRisk))
	if err != nil {
		return ShieldTriage{}, err
	}
	return ShieldTriage{
		Reachable:                Reachability(r.Value),
		ReachableConfidence:      r.Confidence,
		FalsePositive:            fp.Value == "true",
		FalsePositiveProbability: fp.Probability,
		FalsePositiveConfidence:  fp.Confidence,
		FixStrategy:              FixStrategy(fs.Value),
		FixStrategyConfidence:    fs.Confidence,
	}, nil
}
