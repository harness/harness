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

package suspect

import (
	"context"
	"testing"
	"time"
)

type fakeSource struct {
	tree    []string
	blame   map[string][]BlameLine // path -> all lines
	commits map[string][]CommitRef // path -> commits in range
	hunks   map[string]map[string][]Hunk
}

func (f *fakeSource) ListTreePaths(context.Context, string) ([]string, error) { return f.tree, nil }

func (f *fakeSource) BlameLines(_ context.Context, _, p string, from, to int) ([]BlameLine, error) {
	var out []BlameLine
	for _, l := range f.blame[p] {
		if l.Line >= from && l.Line <= to {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeSource) ListCommitsTouching(_ context.Context, _, _, p string) ([]CommitRef, error) {
	return f.commits[p], nil
}

func (f *fakeSource) CommitDiffHunks(_ context.Context, c CommitRef) (map[string][]Hunk, error) {
	return f.hunks[c.SHA], nil
}

func TestNormalizePath(t *testing.T) {
	tests := map[string]string{
		"/app/src/a.js":                    "src/a.js",
		"./src/a.js":                       "src/a.js",
		"webpack://myapp/./src/a.js":       "src/a.js",
		"file:///srv/www/src/a.js?v=3":     "srv/www/src/a.js",
		"app:///static/js/main.js#L2":      "static/js/main.js",
		"C:\\work\\src\\a.js":              "C:/work/src/a.js",
		"https://x.com/static/a%20b.js?q=": "https:/x.com/static/a b.js",
	}
	for in, want := range tests {
		if got := NormalizePath(in); got != want {
			t.Errorf("NormalizePath(%q)=%q want %q", in, got, want)
		}
	}
}

func TestMapFrame(t *testing.T) {
	idx := newPathIndex([]string{
		"web/src/app/service.ts", "api/service.ts", "pkg/util/db.py", "main.go", "a/x/util.go", "b/x/util.go",
	})
	tests := []struct {
		f    Frame
		want string
		ok   bool
	}{
		{Frame{Filename: "/app/web/src/app/service.ts"}, "web/src/app/service.ts", true},
		{Frame{Filename: "webpack://n/./api/service.ts?x"}, "api/service.ts", true},
		{Frame{Filename: "/srv/build/service.ts"}, "", false}, // ambiguous basename
		{Frame{Filename: "/home/u/proj/main.go"}, "main.go", true},
		{Frame{Filename: "/opt/x/util.go"}, "", false}, // ambiguous
		{Frame{Filename: "/opt/a/x/util.go"}, "a/x/util.go", true},
		{Frame{Module: "pkg.util.db"}, "pkg/util/db.py", true},
		{Frame{Filename: "/usr/lib/node_modules/nope.js"}, "", false},
	}
	for _, tc := range tests {
		got, ok := idx.mapFrame(tc.f)
		if got != tc.want || ok != tc.ok {
			t.Errorf("mapFrame(%+v)=%q,%v want %q,%v", tc.f, got, ok, tc.want, tc.ok)
		}
	}
}

func TestFind(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cA := CommitRef{SHA: "aaa", Parent: "p", Title: "touch far", When: base}
	cB := CommitRef{SHA: "bbb", Parent: "p", Title: "touch near", When: base.Add(time.Hour)}
	cC := CommitRef{SHA: "ccc", Parent: "p", Title: "other file", When: base.Add(2 * time.Hour)}
	cOld := CommitRef{SHA: "old", Title: "ancient blame", When: base.Add(-240 * time.Hour)}

	src := &fakeSource{
		tree: []string{"src/handler.go", "src/db.go", "README.md"},
		blame: map[string][]BlameLine{
			"src/handler.go": {{Line: 50, Commit: cOld}, {Line: 51, Commit: cOld}, {Line: 52, Commit: cB}, {Line: 53, Commit: cOld}, {Line: 54, Commit: cOld}},
			"src/db.go":      {{Line: 9, Commit: cOld}, {Line: 10, Commit: cOld}},
		},
		commits: map[string][]CommitRef{
			"src/handler.go": {cA, cB},
			"src/db.go":      {cC},
		},
		hunks: map[string]map[string][]Hunk{
			"aaa": {"src/handler.go": {{NewStart: 200, NewLines: 3}}},
			"bbb": {"src/handler.go": {{NewStart: 51, NewLines: 2}}},
			"ccc": {"src/db.go": {{NewStart: 1, NewLines: 1}}},
		},
	}
	in := Input{
		ReleaseSHA:  "rel",
		PreviousSHA: "prev",
		Frames: []Frame{
			{Filename: "/app/src/handler.go", Function: "Handle", Lineno: 52},
			{Filename: "/usr/lib/go/net/http/server.go", Lineno: 10}, // unmapped
			{Filename: "src/db.go", Function: "Query", Lineno: 10},
		},
	}
	got, err := Find(context.Background(), src, in, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 3 {
		t.Fatalf("expected >=3 candidates, got %d", len(got))
	}
	if got[0].Commit.SHA != "bbb" {
		t.Fatalf("top suspect = %s, want bbb: %+v", got[0].Commit.SHA, got)
	}
	kinds := map[string]bool{}
	for _, r := range got[0].Reasons {
		kinds[r.Kind] = true
		if r.Path != "src/handler.go" || r.Depth != 0 {
			t.Errorf("unexpected reason %+v", r)
		}
	}
	for _, k := range []string{"touched_file", "diff_near_line", "blame"} {
		if !kinds[k] {
			t.Errorf("missing reason kind %s", k)
		}
	}
	rank := map[string]int{}
	for i, c := range got {
		rank[c.Commit.SHA] = i
	}
	if rank["bbb"] >= rank["aaa"] {
		t.Errorf("near commit must outrank far commit: %v", rank)
	}
	if got[0].FramesHit != 1 {
		t.Errorf("FramesHit=%d", got[0].FramesHit)
	}
}

func TestFindNoPreviousUsesBlameOnly(t *testing.T) {
	c := CommitRef{SHA: "x", When: time.Now()}
	src := &fakeSource{
		tree:  []string{"a.go"},
		blame: map[string][]BlameLine{"a.go": {{Line: 5, Commit: c}}},
	}
	got, err := Find(context.Background(), src, Input{ReleaseSHA: "r", Frames: []Frame{{Filename: "a.go", Lineno: 5}}}, Options{})
	if err != nil || len(got) != 1 || got[0].Commit.SHA != "x" || got[0].Reasons[0].Kind != "blame" {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestFindNone(t *testing.T) {
	src := &fakeSource{tree: []string{"a.go"}}
	got, err := Find(context.Background(), src, Input{ReleaseSHA: "r", Frames: []Frame{{Filename: "zzz.go", Lineno: 1}}}, Options{})
	if err != nil || got != nil {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestFindMaxCandidates(t *testing.T) {
	src := &fakeSource{tree: []string{"a.go"}, commits: map[string][]CommitRef{}}
	for i := 0; i < 9; i++ {
		src.commits["a.go"] = append(src.commits["a.go"], CommitRef{SHA: string(rune('a' + i)), When: time.Unix(int64(i), 0)})
	}
	got, _ := Find(context.Background(), src, Input{ReleaseSHA: "r", PreviousSHA: "p", Frames: []Frame{{Filename: "a.go"}}}, Options{})
	if len(got) != 5 || got[0].Commit.SHA != "i" {
		t.Fatalf("got %d, top %s", len(got), got[0].Commit.SHA)
	}
}
