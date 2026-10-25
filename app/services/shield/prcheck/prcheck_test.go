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

package prcheck

import (
	"context"
	"errors"
	"strings"
	"testing"

	checkevents "github.com/harness/gitness/app/events/check"
	"github.com/harness/gitness/app/services/shield"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeChecks struct {
	rows  map[string]*types.Check
	trail []enum.CheckStatus
}

func (f *fakeChecks) FindByIdentifier(_ context.Context, _ int64, sha, id string) (types.Check, error) {
	if c, ok := f.rows[sha+"|"+id]; ok {
		return *c, nil
	}
	return types.Check{}, gitness_store.ErrResourceNotFound
}

func (f *fakeChecks) Upsert(_ context.Context, c *types.Check) error {
	cc := *c
	f.rows[c.CommitSHA+"|"+c.Identifier] = &cc
	f.trail = append(f.trail, c.Status)
	return nil
}

type fakeDiffer struct {
	enabled bool
	res     *shield.DiffResult
	err     error
	calls   int
}

func (d *fakeDiffer) Enabled() bool       { return d.enabled }
func (d *fakeDiffer) PatchThreshold() int { return 70 }
func (d *fakeDiffer) ScanDiff(context.Context, *types.RepositoryCore, string, string) (*shield.DiffResult, error) {
	d.calls++
	return d.res, d.err
}

type fakePRs struct{ pr *types.PullReq }

func (f fakePRs) Find(context.Context, int64) (*types.PullReq, error) { return f.pr, nil }

type fakeRepos struct{}

func (fakeRepos) FindByID(context.Context, int64) (*types.RepositoryCore, error) {
	return &types.RepositoryCore{ID: 1}, nil
}

type nopReporter struct{ n int }

func (r *nopReporter) Reported(context.Context, *checkevents.ReportedPayload) { r.n++ }

func finding(pkg, id string, score int, fix string) *types.ShieldFinding {
	return &types.ShieldFinding{
		Package: pkg, InstalledVersion: "1.0.0", VulnID: id, RiskScore: score, MinFixedVersion: fix,
	}
}

func TestDecide(t *testing.T) {
	cases := []struct {
		name string
		res  *shield.DiffResult
		err  error
		want enum.CheckStatus
	}{
		{"error", nil, errors.New("boom"), enum.CheckStatusError},
		{"none", &shield.DiffResult{}, nil, enum.CheckStatusSuccess},
		{"below", &shield.DiffResult{New: []*types.ShieldFinding{finding("a", "V1", 69, "")}}, nil,
			enum.CheckStatusSuccess},
		{"at", &shield.DiffResult{New: []*types.ShieldFinding{finding("a", "V1", 70, "2.0")}}, nil,
			enum.CheckStatusFailure},
	}
	for _, c := range cases {
		got, summary := Decide(c.res, c.err, 70)
		if got != c.want || summary == "" {
			t.Errorf("%s: status=%s summary=%q", c.name, got, summary)
		}
	}
	if _, s := Decide(&shield.DiffResult{}, nil, 70); s != "no new vulnerabilities above 70" {
		t.Errorf("summary = %q", s)
	}
}

func TestDecide_TopFive(t *testing.T) {
	var fs []*types.ShieldFinding
	for i := 0; i < 8; i++ {
		fs = append(fs, finding("pkg", "V"+string(rune('A'+i)), 70+i, "9.9"))
	}
	st, s := Decide(&shield.DiffResult{New: fs}, nil, 70)
	if st != enum.CheckStatusFailure {
		t.Fatal(st)
	}
	if strings.Count(s, "\n- ") != 5 || !strings.Contains(s, "and 3 more") ||
		!strings.Contains(s, "pkg 1.0.0: VH (risk 77, fix 9.9)") {
		t.Errorf("summary:\n%s", s)
	}
}

func newSvc(d *fakeDiffer) (*Service, *fakeChecks) {
	src := int64(1)
	checks := &fakeChecks{rows: map[string]*types.Check{}}
	s := New(d, fakePRs{pr: &types.PullReq{SourceRepoID: &src, TargetRepoID: 1, SourceSHA: "head", MergeBaseSHA: "base"}},
		fakeRepos{}, checks, &nopReporter{})
	s.principalID = func() int64 { return 1 }
	s.now = func() int64 { return 1 }
	return s, checks
}

func TestCheck_IdempotentPerHead(t *testing.T) {
	d := &fakeDiffer{enabled: true, res: &shield.DiffResult{
		New: []*types.ShieldFinding{finding("a", "V1", 90, "2.0")}}}
	s, checks := newSvc(d)
	ctx := context.Background()
	if err := s.Check(ctx, 1, "head"); err != nil {
		t.Fatal(err)
	}
	if err := s.Check(ctx, 1, "head"); err != nil {
		t.Fatal(err)
	}
	if d.calls != 1 {
		t.Errorf("scans = %d, want 1", d.calls)
	}
	if len(checks.trail) != 2 || checks.trail[0] != enum.CheckStatusPending || checks.trail[1] != enum.CheckStatusFailure {
		t.Errorf("trail = %v", checks.trail)
	}
}

func TestCheck_ErrorRetriedAndStaleSkipped(t *testing.T) {
	d := &fakeDiffer{enabled: true, err: errors.New("boom")}
	s, checks := newSvc(d)
	ctx := context.Background()
	if err := s.Check(ctx, 1, "head"); err != nil {
		t.Fatal(err)
	}
	if got := checks.rows["head|"+Identifier].Status; got != enum.CheckStatusError {
		t.Fatalf("status = %s", got)
	}
	d.err, d.res = nil, &shield.DiffResult{}
	if err := s.Check(ctx, 1, "head"); err != nil {
		t.Fatal(err)
	}
	if got := checks.rows["head|"+Identifier].Status; got != enum.CheckStatusSuccess {
		t.Fatalf("status after retry = %s", got)
	}
	calls := d.calls
	if err := s.Check(ctx, 1, "old-sha"); err != nil || d.calls != calls {
		t.Error("stale head must be skipped")
	}
	d.enabled = false
	if err := s.Check(ctx, 1, "head"); err != nil || d.calls != calls {
		t.Error("disabled must be skipped")
	}
}
