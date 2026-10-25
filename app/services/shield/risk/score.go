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

// Package risk computes the Shield risk score of a vulnerability finding:
// CVSS x EPSS factor x KEV factor x reachability factor x exposure factor,
// normalized to 0..100. It has no DB or network dependency except the
// optional EPSS/KEV fetchers.
package risk

import (
	"math"
	"strings"
)

// Reachability is the outcome of the reachability analysis of a finding.
type Reachability string

const (
	ReachabilityReachable    Reachability = "reachable"
	ReachabilityUnknown      Reachability = "unknown"
	ReachabilityNotReachable Reachability = "not_reachable"
)

// Exposure describes how exposed the affected component is.
type Exposure string

const (
	// ExposurePublic is an internet-facing service or a public repository
	// (also the default for an empty value).
	ExposurePublic Exposure = "public"
	// ExposureInternal is a component only reachable internally.
	ExposureInternal Exposure = "internal"
	// ExposureDevOnly is a dev/test-only dependency.
	ExposureDevOnly Exposure = "dev_only"
)

// Factor values.
const (
	reachableFactor    = 1.3
	unknownFactor      = 1.0
	notReachableFactor = 0.2
	kevFactor          = 1.5

	exposurePublicFactor   = 1.0
	exposureInternalFactor = 0.8
	exposureDevOnlyFactor  = 0.5

	// EPSSNeutralFactor is used when no alias has an EPSS entry.
	EPSSNeutralFactor = 1.0

	// DefaultHighConfidence is the default confidence threshold from which a
	// not_reachable verdict is trusted.
	DefaultHighConfidence = 0.8

	// MaxRaw is the highest possible raw risk:
	// 10 (CVSS) x 1.5 (EPSS) x 1.5 (KEV) x 1.3 (reachable) x 1.0 (exposure).
	MaxRaw = 10 * 1.5 * 1.5 * 1.3 * 1.0

	// NormRaw is the raw risk mapped to a score of 100: a CVSS 10 finding with every other
	// factor neutral. Aggravating factors (KEV, high EPSS, reachable) push findings with a
	// lower CVSS towards 100 and mitigating ones pull them down; the result is clamped.
	// Normalizing by MaxRaw instead would cap a critical CVE with no extra signal at 34,
	// far below the default patch threshold of 70.
	NormRaw = 10.0
)

// Severity fallbacks used when no CVSS base score is available.
var severityFallback = map[string]float64{
	"critical": 9.0,
	"high":     7.5,
	"medium":   5.0,
	"low":      2.5,
}

const unknownSeverityScore = 5.0

// Input is what Score needs.
type Input struct {
	// CVSSScore is the base score (0..10); used only if HasCVSS.
	CVSSScore float64
	HasCVSS   bool
	// Severity is the qualitative label used when HasCVSS is false.
	Severity string
	// CVEs are the CVE aliases of the finding.
	CVEs []string

	Reachability Reachability
	// Confidence (0..1) of the reachability decision.
	Confidence float64
	// HighConfidence is the threshold from which not_reachable is trusted;
	// zero means DefaultHighConfidence.
	HighConfidence float64

	Exposure Exposure
}

// Breakdown lists every factor of a score, for the Task history and the UI.
type Breakdown struct {
	CVSS       float64 `json:"cvss"`
	CVSSSource string  `json:"cvss_source"` // "cvss" or "severity:<label>"

	EPSSScore      float64 `json:"epss_score"`
	EPSSPercentile float64 `json:"epss_percentile"`
	EPSSCVE        string  `json:"epss_cve,omitempty"`
	EPSSFactor     float64 `json:"epss_factor"`

	InKEV     bool    `json:"in_kev"`
	KEVCVE    string  `json:"kev_cve,omitempty"`
	KEVFactor float64 `json:"kev_factor"`

	Reachability       Reachability `json:"reachability"`
	ReachabilityFactor float64      `json:"reachability_factor"`

	Exposure       Exposure `json:"exposure"`
	ExposureFactor float64  `json:"exposure_factor"`

	Raw   float64 `json:"raw"`
	Max   float64 `json:"max"`
	Score int     `json:"score"`
}

// Scorer computes risk scores with the loaded EPSS and KEV data (either may
// be nil, meaning "no data"; a nil Scorer is valid too).
type Scorer struct {
	EPSS *EPSS
	KEV  *KEV
}

// EPSSFactor maps an EPSS percentile (0..1) to a multiplier in 0.5..1.5:
// 0.5 + percentile.
func EPSSFactor(percentile float64) float64 {
	return 0.5 + clamp(percentile, 0, 1)
}

// Score computes the risk (0..100) of a finding:
//
//	raw   = CVSS x EPSS factor x KEV factor x reachability factor x exposure factor
//	score = round(100 x raw / NormRaw), clamped to 0..100
//
// EPSS factor: 0.5 + percentile of the alias with the highest EPSS score
// (1.0 when no alias is scored). KEV factor: 1.5 if any alias is in KEV.
// Reachability: reachable 1.3, unknown 1.0, not_reachable 0.2 only when
// Confidence >= HighConfidence (else 1.0). Exposure: public 1.0, internal
// 0.8, dev_only 0.5.
func (s *Scorer) Score(in Input) (int, Breakdown) {
	b := Breakdown{Max: NormRaw, EPSSFactor: EPSSNeutralFactor, KEVFactor: 1.0}

	if in.HasCVSS {
		b.CVSS = clamp(in.CVSSScore, 0, 10)
		b.CVSSSource = "cvss"
	} else {
		label := strings.ToLower(strings.TrimSpace(in.Severity))
		v, ok := severityFallback[label]
		if !ok {
			v = unknownSeverityScore
			label = "unknown"
		}
		b.CVSS = v
		b.CVSSSource = "severity:" + label
	}

	var best EPSSScore
	found := false
	if s != nil {
		for _, cve := range in.CVEs {
			id := strings.ToUpper(strings.TrimSpace(cve))
			if e, ok := s.EPSS.Lookup(id); ok && (!found || e.Score > best.Score) {
				best, found = e, true
				b.EPSSCVE = id
			}
			if !b.InKEV && s.KEV.Contains(id) {
				b.InKEV = true
				b.KEVCVE = id
			}
		}
	}
	if found {
		b.EPSSScore, b.EPSSPercentile = best.Score, best.Percentile
		b.EPSSFactor = EPSSFactor(best.Percentile)
	}
	if b.InKEV {
		b.KEVFactor = kevFactor
	}

	threshold := in.HighConfidence
	if threshold <= 0 {
		threshold = DefaultHighConfidence
	}
	b.Reachability = in.Reachability
	switch in.Reachability {
	case ReachabilityReachable:
		b.ReachabilityFactor = reachableFactor
	case ReachabilityNotReachable:
		if in.Confidence >= threshold {
			b.ReachabilityFactor = notReachableFactor
		} else {
			b.ReachabilityFactor = unknownFactor
		}
	default:
		b.Reachability = ReachabilityUnknown
		b.ReachabilityFactor = unknownFactor
	}

	switch in.Exposure {
	case ExposureInternal:
		b.Exposure, b.ExposureFactor = ExposureInternal, exposureInternalFactor
	case ExposureDevOnly:
		b.Exposure, b.ExposureFactor = ExposureDevOnly, exposureDevOnlyFactor
	default:
		b.Exposure, b.ExposureFactor = ExposurePublic, exposurePublicFactor
	}

	b.Raw = b.CVSS * b.EPSSFactor * b.KEVFactor * b.ReachabilityFactor * b.ExposureFactor
	b.Score = int(clamp(math.Round(100*b.Raw/NormRaw), 0, 100))
	return b.Score, b
}

func clamp(v, lo, hi float64) float64 {
	if math.IsNaN(v) || v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
