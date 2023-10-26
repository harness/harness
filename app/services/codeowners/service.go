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

package codeowners

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/gitrpc"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog/log"
)

const (
	oneMegabyte = 1048576
	// maxGetContentFileSize specifies the maximum number of bytes a file content response contains.
	// If a file is any larger, the content is truncated.
	maxGetContentFileSize = oneMegabyte * 4 // 4 MB
)

var (
	ErrNotFound = errors.New("file not found")
)

// TooLargeError represents an error if codeowners file is too large.
type TooLargeError struct {
	FileSize int64
}

func IsTooLargeError(err error) bool {
	return errors.Is(err, &TooLargeError{})
}

func (e *TooLargeError) Error() string {
	return fmt.Sprintf(
		"The repository's CODEOWNERS file size %.2fMB exceeds the maximum supported size of %dMB",
		float32(e.FileSize)/oneMegabyte,
		maxGetContentFileSize/oneMegabyte,
	)
}

//nolint:errorlint // the purpose of this method is to check whether the target itself if of this type.
func (e *TooLargeError) Is(target error) bool {
	_, ok := target.(*TooLargeError)
	return ok
}

type Config struct {
	FilePath string
}

type Service struct {
	repoStore      store.RepoStore
	git            gitrpc.Interface
	principalStore store.PrincipalStore
	config         Config
}

type File struct {
	Content   string
	SHA       string
	TotalSize int64
}

type CodeOwners struct {
	FileSHA string
	Entries []Entry
}

type Entry struct {
	Pattern string
	Owners  []string
}

type Evaluation struct {
	EvaluationEntries []EvaluationEntry
	FileSha           string
}

type EvaluationEntry struct {
	Pattern          string
	OwnerEvaluations []OwnerEvaluation
}

type OwnerEvaluation struct {
	Owner          types.PrincipalInfo
	ReviewDecision enum.PullReqReviewDecision
	ReviewSHA      string
}

func New(
	repoStore store.RepoStore,
	git gitrpc.Interface,
	config Config,
	principalStore store.PrincipalStore,
) *Service {
	service := &Service{
		repoStore:      repoStore,
		git:            git,
		config:         config,
		principalStore: principalStore,
	}
	return service
}

func (s *Service) get(
	ctx context.Context,
	repo *types.Repository,
	ref string,
) (*CodeOwners, error) {
	codeOwnerFile, err := s.getCodeOwnerFile(ctx, repo, ref)
	// no codeowner file
	if gitrpc.ErrorStatus(err) == gitrpc.StatusPathNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get codeowner file: %w", err)
	}
	if codeOwnerFile.TotalSize > maxGetContentFileSize {
		return nil, &TooLargeError{FileSize: codeOwnerFile.TotalSize}
	}

	owner, err := s.parseCodeOwner(codeOwnerFile.Content)
	if err != nil {
		return nil, fmt.Errorf("unable to parse codeowner %w", err)
	}

	return &CodeOwners{
		FileSHA: codeOwnerFile.SHA,
		Entries: owner,
	}, nil
}

func (s *Service) parseCodeOwner(codeOwnersContent string) ([]Entry, error) {
	var codeOwners []Entry
	scanner := bufio.NewScanner(strings.NewReader(codeOwnersContent))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			return nil, fmt.Errorf("line has invalid format: '%s'", line)
		}

		pattern := parts[0]
		owners := parts[1:]

		codeOwner := Entry{
			Pattern: pattern,
			Owners:  owners,
		}

		codeOwners = append(codeOwners, codeOwner)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return codeOwners, nil
}

func (s *Service) getCodeOwnerFile(
	ctx context.Context,
	repo *types.Repository,
	ref string,
) (*File, error) {
	params := gitrpc.CreateRPCReadParams(repo)
	if ref == "" {
		ref = "refs/heads/" + repo.DefaultBranch
	}
	node, err := s.git.GetTreeNode(ctx, &gitrpc.GetTreeNodeParams{
		ReadParams: params,
		GitREF:     ref,
		Path:       s.config.FilePath,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve codeowner file %w", err)
	}

	if node.Node.Mode != gitrpc.TreeNodeModeFile {
		return nil, fmt.Errorf(
			"codeowner file is of format '%s' but expected to be of format '%s'",
			node.Node.Mode,
			gitrpc.TreeNodeModeFile,
		)
	}

	output, err := s.git.GetBlob(ctx, &gitrpc.GetBlobParams{
		ReadParams: params,
		SHA:        node.Node.SHA,
		SizeLimit:  maxGetContentFileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	content, err := io.ReadAll(output.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}

	return &File{
		Content:   string(content),
		SHA:       output.SHA,
		TotalSize: output.Size,
	}, nil
}

func (s *Service) GetApplicableCodeOwnersForPR(
	ctx context.Context,
	repo *types.Repository,
	pr *types.PullReq,
) (*CodeOwners, error) {
	codeOwners, err := s.get(ctx, repo, pr.TargetBranch)
	if err != nil {
		return nil, err
	}

	var filteredEntries []Entry
	diffFileStats, err := s.git.DiffFileNames(ctx, &gitrpc.DiffParams{
		ReadParams: gitrpc.CreateRPCReadParams(repo),
		BaseRef:    pr.MergeBaseSHA,
		HeadRef:    pr.SourceSHA,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get diff file stat: %w", err)
	}

	for _, entry := range codeOwners.Entries {
		ok, err := contains(diffFileStats.Files, entry.Pattern)
		if err != nil {
			return nil, err
		}
		if ok {
			filteredEntries = append(filteredEntries, entry)
		}
	}
	return &CodeOwners{
		FileSHA: codeOwners.FileSHA,
		Entries: filteredEntries,
	}, err
}

func (s *Service) Evaluate(
	ctx context.Context,
	owners *CodeOwners,
	reviewer []*types.PullReqReviewer,
) (*Evaluation, error) {
	flattenedReviewers := flattenReviewers(reviewer)
	evaluationEntries := make([]EvaluationEntry, len(owners.Entries))

	for i, entry := range owners.Entries {
		ownerEvaluations := make([]OwnerEvaluation, len(entry.Owners))
		for j, owner := range entry.Owners {
			if pullreqReviewer, ok := flattenedReviewers[owner]; ok {
				ownerEvaluations[j] = OwnerEvaluation{
					Owner:          pullreqReviewer.Reviewer,
					ReviewDecision: pullreqReviewer.ReviewDecision,
					ReviewSHA:      pullreqReviewer.SHA,
				}
				continue
			}
			principal, err := s.principalStore.FindByEmail(ctx, owner)
			if errors.Is(err, gitness_store.ErrResourceNotFound) {
				log.Ctx(ctx).Info().Msgf("user %s not found in database hence skipping for code owner: %v",
					owner, err)
				continue
			}
			if err != nil {
				return &Evaluation{}, fmt.Errorf("error finding user by email: %w", err)
			}
			ownerEvaluations[j] = OwnerEvaluation{
				Owner: *principal.ToPrincipalInfo(),
			}
		}
		evaluationEntries[i] = EvaluationEntry{
			Pattern:          entry.Pattern,
			OwnerEvaluations: ownerEvaluations,
		}
	}

	return &Evaluation{
		EvaluationEntries: evaluationEntries,
		FileSha:           owners.FileSHA,
	}, nil
}

func flattenReviewers(reviewers []*types.PullReqReviewer) map[string]*types.PullReqReviewer {
	r := make(map[string]*types.PullReqReviewer)
	for _, reviewer := range reviewers {
		r[reviewer.Reviewer.Email] = reviewer
	}
	return r
}

// We match a pattern list against a target
// doubleStar match allows to match / separated path wisely.
// A path foo/bar will match against pattern ** or foo/*
// Also, for a directory ending with / we have to return true for all files in that directory,
// hence we append ** for it.
func contains(patterns []string, target string) (bool, error) {
	for _, pattern := range patterns {
		// in case of / ending rule, owner owns the whole directory hence append **
		if strings.HasSuffix(pattern, "/") {
			pattern += "**"
		}
		match, err := doublestar.PathMatch(pattern, target)
		if err != nil {
			return false, fmt.Errorf("failed to match pattern due to error: %w", err)
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
