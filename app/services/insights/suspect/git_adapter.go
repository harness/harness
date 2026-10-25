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
	"errors"
	"fmt"

	"github.com/harness/gitness/git"
)

// GitSource adapts git.Interface to Source.
type GitSource struct {
	git     git.Interface
	repoUID string
}

var _ Source = (*GitSource)(nil)

// NewGitSource creates a Source reading from the repository identified by repoUID (git UID).
func NewGitSource(g git.Interface, repoUID string) *GitSource {
	return &GitSource{git: g, repoUID: repoUID}
}

func (s *GitSource) read() git.ReadParams { return git.ReadParams{RepoUID: s.repoUID} }

func mapCommit(c *git.Commit) CommitRef {
	if c == nil {
		return CommitRef{}
	}
	ref := CommitRef{
		SHA:    c.SHA.String(),
		Title:  c.Title,
		Author: c.Author.Identity.Name,
		When:   c.Committer.When,
	}
	if len(c.ParentSHAs) > 0 {
		ref.Parent = c.ParentSHAs[0].String()
	}
	return ref
}

// ListTreePaths implements Source.
func (s *GitSource) ListTreePaths(ctx context.Context, ref string) ([]string, error) {
	out, err := s.git.ListPaths(ctx, &git.ListPathsParams{ReadParams: s.read(), GitREF: ref})
	if err != nil {
		return nil, err
	}
	return out.Files, nil
}

// BlameLines implements Source. Blame parts are contiguous and ordered, so line numbers
// are derived sequentially starting at from.
func (s *GitSource) BlameLines(ctx context.Context, ref, path string, from, to int) ([]BlameLine, error) {
	parts, errs := s.git.Blame(ctx, &git.BlameParams{
		ReadParams: s.read(), GitRef: ref, Path: path, LineFrom: from, LineTo: to,
	})
	var out []BlameLine
	line := from
	for p := range parts {
		c := mapCommit(p.Commit)
		for range p.Lines {
			out = append(out, BlameLine{Line: line, Commit: c})
			line++
		}
	}
	if err := <-errs; err != nil {
		return nil, err
	}
	return out, nil
}

// ListCommitsTouching implements Source.
func (s *GitSource) ListCommitsTouching(ctx context.Context, afterRef, toRef, path string) ([]CommitRef, error) {
	const pageSize = 100
	var out []CommitRef
	for page := int32(1); page <= 5; page++ {
		res, err := s.git.ListCommits(ctx, &git.ListCommitsParams{
			ReadParams: s.read(), GitREF: toRef, After: afterRef, Path: path, Page: page, Limit: pageSize,
		})
		if err != nil {
			return nil, err
		}
		for i := range res.Commits {
			out = append(out, mapCommit(&res.Commits[i]))
		}
		if len(res.Commits) < pageSize {
			break
		}
	}
	return out, nil
}

// CommitDiffHunks implements Source.
func (s *GitSource) CommitDiffHunks(ctx context.Context, c CommitRef) (map[string][]Hunk, error) {
	if c.Parent == "" {
		return nil, errors.New("commit has no parent")
	}
	res, err := s.git.GetDiffHunkHeaders(ctx, git.GetDiffHunkHeadersParams{
		ReadParams: s.read(), SourceCommitSHA: c.Parent, TargetCommitSHA: c.SHA,
	})
	if err != nil {
		return nil, fmt.Errorf("diff hunk headers: %w", err)
	}
	out := map[string][]Hunk{}
	for _, f := range res.Files {
		for _, h := range f.HunkHeaders {
			name := f.FileHeader.NewName
			out[name] = append(out[name], Hunk{NewStart: h.NewLine, NewLines: h.NewSpan})
		}
	}
	return out, nil
}
