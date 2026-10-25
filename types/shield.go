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

package types

import "github.com/harness/gitness/types/enum"

// ShieldScan is one execution of a Shield scanner over a repository.
type ShieldScan struct {
	ID            int64                  `json:"id"`
	RepoID        int64                  `json:"repo_id"`
	CommitSHA     string                 `json:"commit_sha"`
	Ref           string                 `json:"ref"`
	Trigger       enum.ShieldScanTrigger `json:"trigger"`
	Scanner       enum.ShieldScanner     `json:"scanner"`
	Status        enum.ShieldScanStatus  `json:"status"`
	Error         string                 `json:"error,omitempty"`
	PackagesCount int                    `json:"packages_count"`
	FindingsCount int                    `json:"findings_count"`
	Started       int64                  `json:"started"`
	Finished      *int64                 `json:"finished,omitempty"`
	Created       int64                  `json:"created"`
}

// ShieldFinding is a deduplicated security finding of a repository.
// Its identity is (RepoID, Fingerprint).
type ShieldFinding struct {
	ID          int64  `json:"id"`
	Version     int64  `json:"-"`
	RepoID      int64  `json:"repo_id"`
	Fingerprint string `json:"fingerprint"`

	Scanner          enum.ShieldScanner     `json:"scanner"`
	Kind             enum.ShieldFindingKind `json:"kind"`
	Ecosystem        string                 `json:"ecosystem,omitempty"`
	Package          string                 `json:"package,omitempty"`
	InstalledVersion string                 `json:"installed_version,omitempty"`
	ManifestPath     string                 `json:"manifest_path,omitempty"`
	VulnID           string                 `json:"vuln_id"`
	Aliases          []string               `json:"aliases"`
	Summary          string                 `json:"summary,omitempty"`

	CVSSVector string  `json:"cvss_vector,omitempty"`
	CVSSScore  float64 `json:"cvss_score"`
	EPSS       float64 `json:"epss"`
	InKEV      bool    `json:"in_kev"`

	Reachable enum.ShieldReachability `json:"reachable"`
	// RiskScore is the prioritization score in [0, 100].
	RiskScore int `json:"risk_score"`

	FixStrategy     *enum.ShieldFixStrategy `json:"fix_strategy,omitempty"`
	MinFixedVersion string                  `json:"min_fixed_version,omitempty"`
	FixedVersions   []string                `json:"fixed_versions"`

	State       enum.ShieldFindingState `json:"state"`
	StateReason string                  `json:"state_reason,omitempty"`

	FirstSeenScanID int64  `json:"first_seen_scan_id"`
	LastSeenScanID  int64  `json:"last_seen_scan_id"`
	FirstSeen       int64  `json:"first_seen"`
	LastSeen        int64  `json:"last_seen"`
	FixedAt         *int64 `json:"fixed_at,omitempty"`

	TaskID        *int64  `json:"task_id,omitempty"`
	AIDecisionIDs []int64 `json:"ai_decision_ids"`

	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

// ShieldFindingFilter stores Shield finding list query parameters.
type ShieldFindingFilter struct {
	Page         int                       `json:"page"`
	Size         int                       `json:"size"`
	States       []enum.ShieldFindingState `json:"state"`
	Kinds        []enum.ShieldFindingKind  `json:"kind"`
	MinRiskScore int                       `json:"min_risk_score"`
}
