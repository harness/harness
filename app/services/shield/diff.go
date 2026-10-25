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
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/harness/gitness/app/services/shield/osv"
	"github.com/harness/gitness/app/services/shield/risk"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// maxBaseCacheEntries bounds the cache of base commit findings.
const maxBaseCacheEntries = 64

// DiffResult is the outcome of a diff scan: the findings a change introduces compared to its base.
type DiffResult struct {
	// Scan is the persisted scan row of the head commit (trigger pullreq).
	Scan *types.ShieldScan
	// New are the findings present at head but not at base, by risk descending.
	New []*types.ShieldFinding
	// Existing is the number of findings present at both commits.
	Existing int
	// Fixed is the number of findings present at base but not at head.
	Fixed int
}

// ScanDiff scans the head commit and the base commit and returns the findings only present at head.
// Findings are never persisted (the repository finding list and tasks are not touched); only a
// shield_scans row with trigger pullreq records the run. Findings of base commits are cached. A
// scan failure is recorded on the scan row and returned as error.
func (s *Service) ScanDiff(
	ctx context.Context,
	repo *types.RepositoryCore,
	baseSHA, headSHA string,
) (*DiffResult, error) {
	baseSHA = strings.TrimSpace(baseSHA)
	if baseSHA == "" {
		return nil, errors.New("base sha is required")
	}
	scan, err := s.createScan(ctx, repo, headSHA, enum.ShieldScanTriggerPullReq)
	if err != nil {
		return nil, err
	}
	scan.Status = enum.ShieldScanStatusRunning
	if err = s.scans.Update(ctx, scan); err != nil {
		return nil, fmt.Errorf("failed to mark shield scan running: %w", err)
	}

	res, derr := s.diff(ctx, repo, scan, baseSHA)

	finished := s.now()
	scan.Finished = &finished
	if res != nil {
		scan.PackagesCount = res.packages
		scan.FindingsCount = len(res.out.New) + res.out.Existing
	}
	if derr != nil {
		log.Ctx(ctx).Warn().Err(derr).Int64("repo_id", repo.ID).Int64("scan_id", scan.ID).
			Msg("shield diff scan failed")
		scan.Status = enum.ShieldScanStatusFailed
		scan.Error = derr.Error()
	} else {
		scan.Status = enum.ShieldScanStatusDone
	}
	if uerr := s.scans.Update(ctx, scan); uerr != nil {
		return nil, fmt.Errorf("failed to update shield scan: %w", uerr)
	}
	if derr != nil {
		return nil, derr
	}
	res.out.Scan = scan
	return res.out, nil
}

type diffRun struct {
	out      *DiffResult
	packages int
}

func (s *Service) diff(
	ctx context.Context,
	repo *types.RepositoryCore,
	scan *types.ShieldScan,
	baseSHA string,
) (*diffRun, error) {
	exposure, err := s.exposure(ctx, repo)
	if err != nil {
		return nil, err
	}
	head, headPkgs, err := s.findingsAt(ctx, repo, scan.CommitSHA, exposure)
	if err != nil {
		return nil, err
	}
	base, err := s.baseFindings(ctx, repo, baseSHA, exposure)
	if err != nil {
		return nil, err
	}
	out := &DiffResult{}
	for fp, f := range head {
		f.FirstSeenScanID, f.LastSeenScanID, f.RepoID = scan.ID, scan.ID, repo.ID
		if _, ok := base[fp]; ok {
			out.Existing++
			continue
		}
		out.New = append(out.New, f)
	}
	for fp := range base {
		if _, ok := head[fp]; !ok {
			out.Fixed++
		}
	}
	SortByRisk(out.New)
	return &diffRun{out: out, packages: headPkgs}, nil
}

// SortByRisk sorts findings by risk descending, then by package and vulnerability id.
func SortByRisk(fs []*types.ShieldFinding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.RiskScore != b.RiskScore {
			return a.RiskScore > b.RiskScore
		}
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		return a.VulnID < b.VulnID
	})
}

type baseCache struct {
	mu      sync.Mutex
	entries map[string]map[string]*types.ShieldFinding
}

func (c *baseCache) get(key string) (map[string]*types.ShieldFinding, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[key]
	return v, ok
}

func (c *baseCache) put(key string, v map[string]*types.ShieldFinding) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil || len(c.entries) >= maxBaseCacheEntries {
		c.entries = make(map[string]map[string]*types.ShieldFinding, maxBaseCacheEntries)
	}
	c.entries[key] = v
}

func (s *Service) baseFindings(
	ctx context.Context,
	repo *types.RepositoryCore,
	sha string,
	exposure risk.Exposure,
) (map[string]*types.ShieldFinding, error) {
	// the key includes the exposure as it feeds the risk score.
	key := fmt.Sprintf("%d|%s|%s", repo.ID, sha, exposure)
	if v, ok := s.baseCache.get(key); ok {
		return v, nil
	}
	v, _, err := s.findingsAt(ctx, repo, sha, exposure)
	if err != nil {
		return nil, err
	}
	s.baseCache.put(key, v)
	return v, nil
}

// findingsAt computes the scored findings of a commit by fingerprint without persisting anything.
// It fails if a manifest could not be read, as a partial result would misreport the diff.
func (s *Service) findingsAt(
	ctx context.Context,
	repo *types.RepositoryCore,
	sha string,
	exposure risk.Exposure,
) (map[string]*types.ShieldFinding, int, error) {
	pkgs, complete, err := s.readPackages(ctx, repo, sha)
	if err != nil {
		return nil, 0, err
	}
	if !complete {
		return nil, len(pkgs), fmt.Errorf("could not read every manifest at %s", sha)
	}
	out := map[string]*types.ShieldFinding{}
	if len(pkgs) == 0 {
		return out, 0, nil
	}
	db, err := s.data.LoadDB(pkgs)
	if err != nil {
		return nil, len(pkgs), fmt.Errorf("failed to load vulnerability database: %w", err)
	}
	scorer := s.data.Scorer()
	now := s.now()
	for _, of := range osv.Match(db, pkgs).Findings {
		f := FromOSVFinding(of, repo.ID, 0, now)
		if _, dup := out[f.Fingerprint]; dup {
			continue
		}
		score, bd := scorer.Score(risk.Input{
			CVSSScore:    of.CVSSScore,
			HasCVSS:      of.HasScore,
			Severity:     of.Severity,
			CVEs:         of.CVEs,
			Reachability: risk.ReachabilityUnknown,
			Exposure:     exposure,
		})
		f.RiskScore, f.EPSS, f.InKEV = score, bd.EPSSScore, bd.InKEV
		out[f.Fingerprint] = f
	}
	return out, len(pkgs), nil
}

// PatchThreshold returns the configured risk threshold.
func (s *Service) PatchThreshold() int { return s.config.PatchThreshold }
