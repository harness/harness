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
	"fmt"
	"sort"
)

type mappedFrame struct {
	depth int
	frame Frame
	path  string
}

func depthWeight(depth int) float64 { return 1 / (1 + float64(depth)) }

type acc struct {
	commit  CommitRef
	score   float64
	frames  map[int]struct{}
	reasons []Reason
}

// Find computes ranked suspect commits deterministically. An empty result means
// "none" and is up to the caller to handle.
//
// Scoring per (commit, frame): touched file 1, diff hunk within the line window
// 1..3 (closer is higher), blame within the blame window up to 3; each is multiplied
// by 1/(1+frameDepth). The sum is scaled by 1..1.5 by recency among candidates and
// gets +0.25 per additional distinct frame hit.
func Find(ctx context.Context, src Source, in Input, opts Options) ([]Candidate, error) {
	opts = opts.withDefaults()
	if in.ReleaseSHA == "" || len(in.Frames) == 0 {
		return nil, nil
	}
	tree, err := src.ListTreePaths(ctx, in.ReleaseSHA)
	if err != nil {
		return nil, fmt.Errorf("list tree paths: %w", err)
	}
	idx := newPathIndex(tree)

	var frames []mappedFrame
	for i, f := range in.Frames {
		p, ok := idx.mapFrame(f)
		if !ok {
			continue
		}
		frames = append(frames, mappedFrame{depth: i, frame: f, path: p})
		if len(frames) >= opts.MaxFrames {
			break
		}
	}
	if len(frames) == 0 {
		return nil, nil
	}

	cands := map[string]*acc{}
	add := func(c CommitRef, mf mappedFrame, points float64, r Reason) {
		a := cands[c.SHA]
		if a == nil {
			a = &acc{commit: c, frames: map[int]struct{}{}}
			cands[c.SHA] = a
		}
		a.score += points * depthWeight(mf.depth)
		a.frames[mf.depth] = struct{}{}
		r.Path, r.Line, r.Function, r.Depth = mf.path, mf.frame.Lineno, mf.frame.Function, mf.depth
		a.reasons = append(a.reasons, r)
	}

	if in.PreviousSHA != "" && in.PreviousSHA != in.ReleaseSHA {
		if err := collectRangeCommits(ctx, src, in, frames, opts, add); err != nil {
			return nil, err
		}
	}

	for _, mf := range frames {
		if mf.frame.Lineno <= 0 {
			continue
		}
		from := max(1, mf.frame.Lineno-opts.BlameWindow)
		bl, err := src.BlameLines(ctx, in.ReleaseSHA, mf.path, from, mf.frame.Lineno+opts.BlameWindow)
		if err != nil {
			continue // blame is best effort per frame
		}
		closest := map[string]int{}
		commits := map[string]CommitRef{}
		for _, l := range bl {
			d := abs(l.Line - mf.frame.Lineno)
			if cur, ok := closest[l.Commit.SHA]; !ok || d < cur {
				closest[l.Commit.SHA] = d
			}
			commits[l.Commit.SHA] = l.Commit
		}
		for sha, d := range closest {
			prox := 1 - float64(d)/float64(opts.BlameWindow+1)
			add(commits[sha], mf, 3*prox, Reason{Kind: "blame",
				Detail: fmt.Sprintf("last changed a line %d away from the frame line", d)})
		}
	}

	return rank(cands, opts.MaxCandidates), nil
}

func collectRangeCommits(
	ctx context.Context,
	src Source,
	in Input,
	frames []mappedFrame,
	opts Options,
	add func(CommitRef, mappedFrame, float64, Reason),
) error {
	perPath := map[string][]CommitRef{}
	for _, mf := range frames {
		if _, done := perPath[mf.path]; done {
			continue
		}
		cs, err := src.ListCommitsTouching(ctx, in.PreviousSHA, in.ReleaseSHA, mf.path)
		if err != nil {
			return fmt.Errorf("list commits touching %q: %w", mf.path, err)
		}
		perPath[mf.path] = cs
	}
	hunks := map[string]map[string][]Hunk{}
	for _, mf := range frames {
		for _, c := range perPath[mf.path] {
			add(c, mf, 1, Reason{Kind: "touched_file", Detail: "commit in the release range touched the file"})
			if mf.frame.Lineno <= 0 {
				continue
			}
			h, ok := hunks[c.SHA]
			if !ok {
				var err error
				if h, err = src.CommitDiffHunks(ctx, c); err != nil {
					h = nil // hunks are best effort
				}
				hunks[c.SHA] = h
			}
			for _, hk := range h[mf.path] {
				d := hunkDistance(hk, mf.frame.Lineno)
				if d <= opts.LineWindow {
					prox := 1 - float64(d)/float64(opts.LineWindow+1)
					add(c, mf, 1+2*prox, Reason{Kind: "diff_near_line",
						Detail: fmt.Sprintf("diff hunk +%d,%d is %d lines from the frame line", hk.NewStart, hk.NewLines, d)})
					break
				}
			}
		}
	}
	return nil
}

func rank(cands map[string]*acc, limit int) []Candidate {
	var minT, maxT int64
	first := true
	for _, a := range cands {
		t := a.commit.When.Unix()
		if first || t < minT {
			minT = t
		}
		if first || t > maxT {
			maxT = t
		}
		first = false
	}
	out := make([]Candidate, 0, len(cands))
	for _, a := range cands {
		recency := 1.0
		if maxT > minT {
			recency = float64(a.commit.When.Unix()-minT) / float64(maxT-minT)
		}
		score := a.score*(1+0.5*recency) + 0.25*float64(len(a.frames)-1)
		out = append(out, Candidate{Commit: a.commit, Score: score, FramesHit: len(a.frames), Reasons: a.reasons})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if !out[i].Commit.When.Equal(out[j].Commit.When) {
			return out[i].Commit.When.After(out[j].Commit.When)
		}
		return out[i].Commit.SHA < out[j].Commit.SHA
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func hunkDistance(h Hunk, line int) int {
	end := h.NewStart + max(h.NewLines, 1) - 1
	switch {
	case line < h.NewStart:
		return h.NewStart - line
	case line > end:
		return line - end
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
