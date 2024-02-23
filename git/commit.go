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

package git

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	"github.com/rs/zerolog/log"
)

type GetCommitParams struct {
	ReadParams
	// SHA is the git commit sha
	SHA string
}

type Commit struct {
	SHA        string          `json:"sha"`
	ParentSHAs []string        `json:"parent_shas,omitempty"`
	Title      string          `json:"title"`
	Message    string          `json:"message,omitempty"`
	Author     Signature       `json:"author"`
	Committer  Signature       `json:"committer"`
	FileStats  CommitFileStats `json:"file_stats,omitempty"`
	DiffStats  CommitDiffStats `json:"diff_stats,omitempty"`
}

type GetCommitOutput struct {
	Commit Commit `json:"commit"`
}

type Signature struct {
	Identity Identity  `json:"identity"`
	When     time.Time `json:"when"`
}

type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (i *Identity) Validate() error {
	if i.Name == "" {
		return errors.InvalidArgument("identity name is mandatory")
	}

	if i.Email == "" {
		return errors.InvalidArgument("identity email is mandatory")
	}

	return nil
}

func (s *Service) GetCommit(ctx context.Context, params *GetCommitParams) (*GetCommitOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	if !isValidGitSHA(params.SHA) {
		return nil, errors.InvalidArgument("the provided commit sha '%s' is of invalid format.", params.SHA)
	}
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	result, err := s.adapter.GetCommit(ctx, repoPath, params.SHA)
	if err != nil {
		return nil, err
	}

	commit, err := mapCommit(result)
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc commit: %w", err)
	}

	return &GetCommitOutput{
		Commit: *commit,
	}, nil
}

type ListCommitsParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF string
	// After is a git reference (branch / tag / commit SHA)
	// If provided, commits only up to that reference will be returned (exlusive)
	After string
	Page  int32
	Limit int32
	Path  string

	// Since allows to filter for commits since the provided UNIX timestamp - Optional, ignored if value is 0.
	Since int64

	// Until allows to filter for commits until the provided UNIX timestamp - Optional, ignored if value is 0.
	Until int64

	// Committer allows to filter for commits based on the committer - Optional, ignored if string is empty.
	Committer string

	// IncludeFileStats allows you to include information about files changed, added and modified.
	IncludeFileStats bool
}

type RenameDetails struct {
	OldPath         string
	NewPath         string
	CommitShaBefore string
	CommitShaAfter  string
}

type ListCommitsOutput struct {
	Commits       []Commit
	RenameDetails []*RenameDetails
	TotalCommits  int
}

type CommitFileStats struct {
	Added    []string
	Modified []string
	Removed  []string
}

func (s *Service) ListCommits(ctx context.Context, params *ListCommitsParams) (*ListCommitsOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	gitCommits, renameDetails, err := s.adapter.ListCommits(
		ctx,
		repoPath,
		params.GitREF,
		int(params.Page),
		int(params.Limit),
		params.IncludeFileStats,
		types.CommitFilter{
			AfterRef:  params.After,
			Path:      params.Path,
			Since:     params.Since,
			Until:     params.Until,
			Committer: params.Committer,
		},
	)
	if err != nil {
		return nil, err
	}

	// try to get total commits between gitref and After refs
	totalCommits := 0
	if params.Page == 1 && len(gitCommits) < int(params.Limit) {
		totalCommits = len(gitCommits)
	} else if params.After != "" && params.GitREF != params.After {
		div, err := s.adapter.GetCommitDivergences(ctx, repoPath, []types.CommitDivergenceRequest{
			{From: params.GitREF, To: params.After},
		}, 0)
		if err != nil {
			return nil, err
		}
		if len(div) > 0 {
			totalCommits = int(div[0].Ahead)
		}
	}

	commits := make([]Commit, len(gitCommits))
	for i := range gitCommits {
		commit, err := mapCommit(&gitCommits[i])
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc commit: %w", err)
		}

		stat, err := s.CommitShortStat(ctx, &CommitShortStatParams{
			Path: repoPath,
			Ref:  commit.SHA,
		})
		if err != nil {
			log.Warn().Msgf("failed to get diff stats: %s", err)
		}
		commit.DiffStats = CommitDiffStats{
			Additions: stat.Additions,
			Deletions: stat.Deletions,
			Total:     stat.Additions + stat.Deletions,
		}

		commits[i] = *commit
	}

	return &ListCommitsOutput{
		Commits:       commits,
		RenameDetails: mapRenameDetails(renameDetails),
		TotalCommits:  totalCommits,
	}, nil
}

type GetCommitDivergencesParams struct {
	ReadParams
	MaxCount int32
	Requests []CommitDivergenceRequest
}

type GetCommitDivergencesOutput struct {
	Divergences []types.CommitDivergence
}

// CommitDivergenceRequest contains the refs for which the converging commits should be counted.
type CommitDivergenceRequest struct {
	// From is the ref from which the counting of the diverging commits starts.
	From string
	// To is the ref at which the counting of the diverging commits ends.
	To string
}

// CommitDivergence contains the information of the count of converging commits between two refs.
type CommitDivergence struct {
	// Ahead is the count of commits the 'From' ref is ahead of the 'To' ref.
	Ahead int32
	// Behind is the count of commits the 'From' ref is behind the 'To' ref.
	Behind int32
}

func (s *Service) GetCommitDivergences(
	ctx context.Context,
	params *GetCommitDivergencesParams,
) (*GetCommitDivergencesOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	requests := make([]types.CommitDivergenceRequest, len(params.Requests))
	for i, req := range params.Requests {
		requests[i] = types.CommitDivergenceRequest{
			From: req.From,
			To:   req.To,
		}
	}

	// call gitea
	divergences, err := s.adapter.GetCommitDivergences(
		ctx,
		repoPath,
		requests,
		params.MaxCount,
	)
	if err != nil {
		return nil, err
	}

	return &GetCommitDivergencesOutput{
		Divergences: divergences,
	}, nil
}
