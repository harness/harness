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
	"math"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ScannerSAST identifies SAST results in types.ShieldFinding.Scanner.
const ScannerSAST = enum.ShieldScannerSAST

// ToShieldFinding maps a SAST finding to a new open Shield finding (kind sast) observed in scanID.
// The file path is stored in ManifestPath, the CWE ids in Aliases and the CVSS-like score scaled
// to 0-100 seeds RiskScore.
func ToShieldFinding(f Finding, repoID, scanID, now int64) *types.ShieldFinding {
	risk := max(0, min(100, int(math.Round(f.Score*10))))
	aliases := f.CWEs
	if aliases == nil {
		aliases = []string{}
	}
	return &types.ShieldFinding{
		RepoID:          repoID,
		Fingerprint:     f.Fingerprint,
		Scanner:         ScannerSAST,
		Kind:            enum.ShieldFindingKindSAST,
		ManifestPath:    f.Path,
		VulnID:          f.RuleID,
		Aliases:         aliases,
		Summary:         f.Title,
		CVSSScore:       f.Score,
		Reachable:       enum.ShieldReachabilityUnknown,
		RiskScore:       risk,
		FixedVersions:   []string{},
		State:           enum.ShieldFindingStateOpen,
		FirstSeenScanID: scanID,
		LastSeenScanID:  scanID,
		FirstSeen:       now,
		LastSeen:        now,
		Created:         now,
		Updated:         now,
	}
}
