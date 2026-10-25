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

package shield

import (
	"math"

	"github.com/harness/gitness/app/services/shield/osv"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Fingerprint returns the stable identity of a dependency finding:
// ecosystem + package + vulnerability id + manifest path.
func Fingerprint(f osv.Finding) string {
	return f.Package.Ecosystem + "|" + f.Package.Name + "|" + f.VulnID + "|" + f.Package.Path
}

// FromOSVFinding maps an OSV match to a new open Shield finding observed in the given scan.
// The risk score is the CVSS base score scaled to 0-100; enrichment (EPSS, KEV, reachability)
// is expected to refine it afterwards.
func FromOSVFinding(f osv.Finding, repoID, scanID, now int64) *types.ShieldFinding {
	var strategy *enum.ShieldFixStrategy
	s := enum.ShieldFixStrategyNoFix
	if f.MinFixedVersion != "" {
		s = enum.ShieldFixStrategyBump
	}
	strategy = &s

	risk := 0
	if f.HasScore {
		risk = int(math.Round(f.CVSSScore * 10))
		risk = max(0, min(100, risk))
	}

	return &types.ShieldFinding{
		RepoID:           repoID,
		Fingerprint:      Fingerprint(f),
		Scanner:          enum.ShieldScannerOSV,
		Kind:             enum.ShieldFindingKindDependency,
		Ecosystem:        f.Package.Ecosystem,
		Package:          f.Package.Name,
		InstalledVersion: f.Package.Version,
		ManifestPath:     f.Package.Path,
		VulnID:           f.VulnID,
		Aliases:          f.Aliases,
		Summary:          f.Summary,
		CVSSVector:       f.CVSSVector,
		CVSSScore:        f.CVSSScore,
		Reachable:        enum.ShieldReachabilityUnknown,
		RiskScore:        risk,
		FixStrategy:      strategy,
		MinFixedVersion:  f.MinFixedVersion,
		FixedVersions:    f.FixedVersions,
		State:            enum.ShieldFindingStateOpen,
		FirstSeenScanID:  scanID,
		LastSeenScanID:   scanID,
		FirstSeen:        now,
		LastSeen:         now,
		Created:          now,
		Updated:          now,
	}
}
