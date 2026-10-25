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
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types/enum"
)

// refGit serves a lockfile per commit.
type refGit struct{ lock map[string]string }

func (g *refGit) ListPaths(_ context.Context, p *git.ListPathsParams) (*git.ListPathsOutput, error) {
	return &git.ListPathsOutput{Files: []string{"package-lock.json"}}, nil
}

func (g *refGit) GetTreeNode(_ context.Context, p *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	return &git.GetTreeNodeOutput{Node: git.TreeNode{SHA: p.GitREF, Path: p.Path}}, nil
}

func (g *refGit) GetBlob(_ context.Context, p *git.GetBlobParams) (*git.GetBlobOutput, error) {
	c := g.lock[p.SHA]
	return &git.GetBlobOutput{Size: int64(len(c)), ContentSize: int64(len(c)), Content: io.NopCloser(strings.NewReader(c))}, nil
}

func diffEnv(t *testing.T, locks map[string]string) *env {
	e := newEnv(t, 1, true)
	e.svc.git = &refGit{lock: locks}
	return e
}

func TestScanDiff_NewExistingFixed(t *testing.T) {
	vuln := readFixture(t, lockFixture)
	ctx := context.Background()

	// new: clean base, vulnerable head
	e := diffEnv(t, map[string]string{"base": cleanLock, "head": vuln})
	res, err := e.svc.ScanDiff(ctx, e.repo, "base", "head")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.New) == 0 || res.Existing != 0 || res.Fixed != 0 {
		t.Fatalf("new case: %+v", res)
	}
	for i := 1; i < len(res.New); i++ {
		if res.New[i-1].RiskScore < res.New[i].RiskScore {
			t.Error("not sorted by risk")
		}
	}
	if len(e.findings.rows) != 0 || len(e.tasks.tasks) != 0 {
		t.Error("diff scan must not persist findings or create tasks")
	}
	if len(e.scans.rows) != 1 || e.scans.rows[0].Trigger != enum.ShieldScanTriggerPullReq ||
		e.scans.rows[0].Status != enum.ShieldScanStatusDone {
		t.Errorf("bad scan row: %+v", e.scans.rows)
	}
	total := len(res.New)

	// existing: same at both
	e = diffEnv(t, map[string]string{"base": vuln, "head": vuln})
	res, err = e.svc.ScanDiff(ctx, e.repo, "base", "head")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.New) != 0 || res.Existing != total || res.Fixed != 0 {
		t.Fatalf("existing case: new=%d existing=%d fixed=%d", len(res.New), res.Existing, res.Fixed)
	}

	// fixed: vulnerable base, clean head
	e = diffEnv(t, map[string]string{"base": vuln, "head": cleanLock})
	res, err = e.svc.ScanDiff(ctx, e.repo, "base", "head")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.New) != 0 || res.Existing != 0 || res.Fixed != total {
		t.Fatalf("fixed case: %+v", res)
	}
}

func TestScanDiff_BaseCached(t *testing.T) {
	e := diffEnv(t, map[string]string{"base": cleanLock, "h1": cleanLock, "h2": cleanLock})
	for _, h := range []string{"h1", "h2"} {
		if _, err := e.svc.ScanDiff(context.Background(), e.repo, "base", h); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := e.svc.baseCache.get("7|base|public"); !ok {
		t.Error("base findings not cached")
	}
}

func TestScanDiff_Errors(t *testing.T) {
	e := diffEnv(t, nil)
	if _, err := e.svc.ScanDiff(context.Background(), e.repo, "", "head"); err == nil {
		t.Error("expected error for empty base")
	}
	e.svc.config.Enabled = false
	if _, err := e.svc.ScanDiff(context.Background(), e.repo, "b", "h"); err != ErrNotEnabled {
		t.Errorf("err = %v", err)
	}
}
