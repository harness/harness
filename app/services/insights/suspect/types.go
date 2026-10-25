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
	"time"
)

// Frame is an in-app stack frame of an error event, as reported by an SDK.
type Frame struct {
	Filename string
	AbsPath  string
	Module   string
	Function string
	Lineno   int
}

// CommitRef is the minimal commit information needed for ranking.
type CommitRef struct {
	SHA    string
	Parent string // first parent, empty for root commits
	Title  string
	Author string
	When   time.Time
}

// BlameLine attributes a single line (1-based) to the commit that last changed it.
type BlameLine struct {
	Line   int
	Commit CommitRef
}

// Hunk is a changed range on the new side of a diff (1-based start, NewLines may be 0 for pure deletions).
type Hunk struct {
	NewStart int
	NewLines int
}

// Source abstracts the git access required by the algorithm.
type Source interface {
	// ListTreePaths returns all file paths in the tree at ref.
	ListTreePaths(ctx context.Context, ref string) ([]string, error)
	// BlameLines returns blame for lines [from, to] (inclusive) of path at ref.
	BlameLines(ctx context.Context, ref, path string, from, to int) ([]BlameLine, error)
	// ListCommitsTouching returns commits reachable from toRef and not from afterRef that touched path.
	ListCommitsTouching(ctx context.Context, afterRef, toRef, path string) ([]CommitRef, error)
	// CommitDiffHunks returns the changed ranges (new side) per path of the commit against its parent.
	CommitDiffHunks(ctx context.Context, commit CommitRef) (map[string][]Hunk, error)
}

// Input describes the lookup. Frames must be ordered innermost (crash site) first.
type Input struct {
	ReleaseSHA  string
	PreviousSHA string // optional
	Frames      []Frame
}

// Reason explains one piece of evidence for a candidate.
type Reason struct {
	Kind     string // "touched_file", "diff_near_line", "blame"
	Path     string
	Line     int // frame line
	Function string
	Depth    int // frame index, 0 = top
	Detail   string
}

// Candidate is a suspect commit with its score.
type Candidate struct {
	Commit    CommitRef
	Score     float64
	FramesHit int
	Reasons   []Reason
}

// Options tunes the algorithm; zero values pick defaults.
type Options struct {
	LineWindow    int // +-N lines for diff proximity (default 10)
	BlameWindow   int // +-N lines for blame (default 2)
	MaxCandidates int // default 5
	MaxFrames     int // mapped frames considered (default 10)
}

func (o Options) withDefaults() Options {
	if o.LineWindow <= 0 {
		o.LineWindow = 10
	}
	if o.BlameWindow <= 0 {
		o.BlameWindow = 2
	}
	if o.MaxCandidates <= 0 {
		o.MaxCandidates = 5
	}
	if o.MaxFrames <= 0 {
		o.MaxFrames = 10
	}
	return o
}
