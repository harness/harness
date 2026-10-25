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

// Package taskassign suggests the human assignee of a task from the CODEOWNERS of the affected files.
package taskassign

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// CandidateKind tells whether a candidate is a single user or a user group.
type CandidateKind string

const (
	CandidateKindUser  CandidateKind = "user"
	CandidateKindGroup CandidateKind = "group"
)

// Candidate is a resolved CODEOWNERS owner.
type Candidate struct {
	Kind CandidateKind `json:"kind"`
	// Owner is the owner as written in the CODEOWNERS file (email or @group).
	Owner       string `json:"owner"`
	DisplayName string `json:"display_name,omitempty"`
	PrincipalID *int64 `json:"principal_id,omitempty"`
	GroupID     *int64 `json:"group_id,omitempty"`
	// PathCount is the number of input paths owned by the candidate.
	PathCount int      `json:"path_count"`
	Paths     []string `json:"paths"`
}

// Rule is a CODEOWNERS rule that matched at least one of the paths. Meant for the task history.
type Rule struct {
	LineNumber int64    `json:"line_number"`
	Pattern    string   `json:"pattern"`
	Owners     []string `json:"owners"`
	Paths      []string `json:"paths"`
}

// Suggestion is the result of Suggest. The zero value means "no suggestion".
type Suggestion struct {
	// Top is the best candidate, nil if there is none.
	Top *Candidate `json:"top,omitempty"`
	// Candidates is the full ranked list (Top first).
	Candidates []Candidate `json:"candidates,omitempty"`
	Rules      []Rule      `json:"rules,omitempty"`
	FileSHA    string      `json:"file_sha,omitempty"`
}

// pathMatcher is the part of the codeowners service used by this package.
type pathMatcher interface {
	MatchPaths(
		ctx context.Context,
		repo *types.RepositoryCore,
		ref string,
		paths []string,
	) (*codeowners.PathMatches, error)
}

// Service suggests task assignees.
type Service struct {
	codeOwners        pathMatcher
	repoStore         store.RepoStore
	principalStore    store.PrincipalStore
	userGroupResolver usergroup.Resolver
}

// NewService creates the service.
func NewService(
	codeOwners pathMatcher,
	repoStore store.RepoStore,
	principalStore store.PrincipalStore,
	userGroupResolver usergroup.Resolver,
) *Service {
	return &Service{
		codeOwners:        codeOwners,
		repoStore:         repoStore,
		principalStore:    principalStore,
		userGroupResolver: userGroupResolver,
	}
}

type ownerStats struct {
	owner string
	// line and pos locate the first appearance of the owner in the CODEOWNERS file.
	line  int64
	pos   int
	paths []string
}

// Suggest evaluates CODEOWNERS at the default branch for the given repo paths and ranks the owners by the
// number of paths they own (ties: users before groups, then order of appearance in CODEOWNERS).
// A missing CODEOWNERS file or no matching rule gives an empty suggestion and no error.
func (s *Service) Suggest(ctx context.Context, repo *types.RepositoryCore, paths []string) (Suggestion, error) {
	if repo == nil || len(paths) == 0 {
		return Suggestion{}, nil
	}

	res, err := s.codeOwners.MatchPaths(ctx, repo, "", paths)
	if errors.Is(err, codeowners.ErrNotFound) {
		return Suggestion{}, nil
	}
	if err != nil {
		return Suggestion{}, fmt.Errorf("failed to match code owners: %w", err)
	}
	if len(res.Matches) == 0 {
		return Suggestion{}, nil
	}

	var (
		rules   []Rule
		ruleIdx = map[int64]int{}
		stats   = map[string]*ownerStats{}
		ordered []*ownerStats
	)
	for _, m := range res.Matches {
		i, ok := ruleIdx[m.Entry.LineNumber]
		if !ok {
			i = len(rules)
			ruleIdx[m.Entry.LineNumber] = i
			rules = append(rules, Rule{
				LineNumber: m.Entry.LineNumber,
				Pattern:    m.Entry.Pattern,
				Owners:     m.Entry.Owners,
			})
		}
		rules[i].Paths = append(rules[i].Paths, m.Path)

		for pos, owner := range m.Entry.Owners {
			st, found := stats[owner]
			if !found {
				st = &ownerStats{owner: owner, line: m.Entry.LineNumber, pos: pos}
				stats[owner] = st
				ordered = append(ordered, st)
			}
			if m.Entry.LineNumber < st.line || (m.Entry.LineNumber == st.line && pos < st.pos) {
				st.line, st.pos = m.Entry.LineNumber, pos
			}
			st.paths = append(st.paths, m.Path)
		}
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].LineNumber < rules[j].LineNumber })

	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].line != ordered[j].line {
			return ordered[i].line < ordered[j].line
		}
		return ordered[i].pos < ordered[j].pos
	})

	candidates := make([]Candidate, 0, len(ordered))
	for _, st := range ordered {
		c, resolved, resolveErr := s.resolve(ctx, st.owner)
		if resolveErr != nil {
			return Suggestion{}, resolveErr
		}
		if !resolved {
			continue
		}
		c.PathCount = len(st.paths)
		c.Paths = st.paths
		candidates = append(candidates, c)
	}
	if len(candidates) == 0 {
		return Suggestion{Rules: rules, FileSHA: res.FileSHA}, nil
	}

	// candidates are in CODEOWNERS order, so a stable sort keeps it for equal keys.
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.PathCount != b.PathCount {
			return a.PathCount > b.PathCount
		}
		return a.Kind == CandidateKindUser && b.Kind == CandidateKindGroup
	})

	top := candidates[0]
	return Suggestion{
		Top:        &top,
		Candidates: candidates,
		Rules:      rules,
		FileSHA:    res.FileSHA,
	}, nil
}

// SuggestForTask is Suggest for the repository of the task. It also returns the assignee input to pass to
// task.Service.Update; the input is nil if there is no suggestion. It does not apply anything.
func (s *Service) SuggestForTask(
	ctx context.Context,
	t *types.Task,
	paths []string,
) (Suggestion, *task.AssigneeInput, error) {
	if t == nil {
		return Suggestion{}, nil, nil
	}
	repo, err := s.repoStore.Find(ctx, t.RepoID)
	if err != nil {
		return Suggestion{}, nil, fmt.Errorf("failed to find repo %d of task: %w", t.RepoID, err)
	}

	sug, err := s.Suggest(ctx, repo.Core(), paths)
	if err != nil {
		return Suggestion{}, nil, err
	}
	if sug.Top == nil {
		return sug, nil, nil
	}
	return sug, &task.AssigneeInput{
		PrincipalID: sug.Top.PrincipalID,
		GroupID:     sug.Top.GroupID,
	}, nil
}

// resolve maps a CODEOWNERS owner to a user or user group. Unknown owners are skipped (resolved=false).
func (s *Service) resolve(ctx context.Context, owner string) (Candidate, bool, error) {
	if groupID, ok := codeowners.ParseUserGroupOwner(owner); ok {
		group, err := s.userGroupResolver.Resolve(ctx, groupID)
		if errors.Is(err, usergroup.ErrNotFound) {
			log.Ctx(ctx).Debug().Msgf("user group %q not found", groupID)
			return Candidate{}, false, nil
		}
		if err != nil {
			return Candidate{}, false, fmt.Errorf("failed to resolve user group %q: %w", groupID, err)
		}
		id := group.ID
		return Candidate{Kind: CandidateKindGroup, Owner: owner, DisplayName: group.Name, GroupID: &id}, true, nil
	}

	principal, err := s.principalStore.FindByEmail(ctx, owner)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		log.Ctx(ctx).Debug().Msgf("user %q not found", owner)
		return Candidate{}, false, nil
	}
	if err != nil {
		return Candidate{}, false, fmt.Errorf("failed to find user by email %q: %w", owner, err)
	}
	id := principal.ID
	return Candidate{
		Kind: CandidateKindUser, Owner: owner, DisplayName: principal.DisplayName, PrincipalID: &id,
	}, true, nil
}
