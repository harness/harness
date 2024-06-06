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
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

const (
	oneMegabyte = 1048576
	// maxGetContentFileSize specifies the maximum number of bytes a file content response contains.
	// If a file is any larger, the content is truncated.
	maxGetContentFileSize = oneMegabyte * 4 // 4 MB
	// userGroupPrefixMarker is a prefix which will be used to identify if a given codeowner is usergroup.
	userGroupPrefixMarker = "@"
)

var (
	ErrNotFound = errors.New("file not found")

	// escapableControlCharactersInPattern are control characters that are used to
	// control the parsing of the codeowners file.
	escapableControlCharactersInPattern = []rune{' ', '\t', '#'}
	// escapableSpecialCharactersInPattern are special characters that are available in the pattern syntax
	// to allow for more complex pattern matching.
	escapableSpecialCharactersInPattern  = []rune{'*', '?', '[', ']', '{', '}', '-', '!', '^'}
	ErrFileParseInvalidEscapingInPattern = errors.New(
		"a pattern requires '\\' to be escaped with another '\\', or used to escape control characters " +
			"[space, tab, '#'] or any of the available special characters " +
			"['*', '?', '[', ']', '{', '}', '-', '!', '^']",
	)
	ErrFileParseTrailingBackslashInPattern = errors.New("a pattern can't end with a trailing '\\'")
)

// TooLargeError represents an error if codeowners file is too large.
type TooLargeError struct {
	FileSize int64
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

// FileParseError represents an error if codeowners file is not parsable.
type FileParseError struct {
	LineNumber int64
	Line       string
	Err        error
}

func (e *FileParseError) Error() string {
	return fmt.Sprintf(
		"The repository's CODEOWNERS file has an error at line %d: %s", e.LineNumber, e.Err,
	)
}

func (e *FileParseError) Unwrap() error {
	return e.Err
}

func (e *FileParseError) Is(target error) bool {
	_, ok := target.(*FileParseError)
	return ok
}

type Config struct {
	FilePaths []string
}

type Service struct {
	repoStore         store.RepoStore
	git               git.Interface
	principalStore    store.PrincipalStore
	config            Config
	userGroupResolver usergroup.Resolver
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
	// LineNumber is the line number of the code owners entry.
	LineNumber int64

	// Pattern is a glob star pattern used to match the entry against a given file path.
	Pattern string
	// Owners is the list of owners for the given pattern.
	// NOTE: Could be empty in case of an entry that clears previously defined ownerships.
	Owners []string
}

// IsOwnershipReset returns true iff the entry resets any previously defined ownerships.
func (e *Entry) IsOwnershipReset() bool {
	return len(e.Owners) == 0
}

type Evaluation struct {
	EvaluationEntries []EvaluationEntry
	FileSha           string
}

type EvaluationEntry struct {
	LineNumber                int64
	Pattern                   string
	OwnerEvaluations          []OwnerEvaluation
	UserGroupOwnerEvaluations []UserGroupOwnerEvaluation
}

type UserGroupOwnerEvaluation struct {
	Identifier  string
	Name        string
	Evaluations []OwnerEvaluation
}

type OwnerEvaluation struct {
	Owner          types.PrincipalInfo
	ReviewDecision enum.PullReqReviewDecision
	ReviewSHA      string
}

func New(
	repoStore store.RepoStore,
	git git.Interface,
	config Config,
	principalStore store.PrincipalStore,
	userGroupResolver usergroup.Resolver,
) *Service {
	service := &Service{
		repoStore:         repoStore,
		git:               git,
		config:            config,
		principalStore:    principalStore,
		userGroupResolver: userGroupResolver,
	}
	return service
}

func (s *Service) get(
	ctx context.Context,
	repo *types.Repository,
	ref string,
) (*CodeOwners, error) {
	codeOwnerFile, err := s.getCodeOwnerFile(ctx, repo, ref)
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
	var lineNumber int64
	var codeOwners []Entry
	scanner := bufio.NewScanner(strings.NewReader(codeOwnersContent))
	for scanner.Scan() {
		lineNumber++
		originalLine := scanner.Text()

		line := strings.TrimSpace(originalLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		isSeparator := func(r rune) bool { return r == ' ' || r == '\t' }
		lineAsRunes := []rune(line)
		pattern := strings.Builder{}

		// important to iterate over runes and not bytes to support utf-8 encoding.
		for len(lineAsRunes) > 0 {
			if isSeparator(lineAsRunes[0]) || lineAsRunes[0] == '#' {
				break
			}

			if lineAsRunes[0] == '\\' {
				// ensure pattern doesn't end with trailing backslash.
				if len(lineAsRunes) == 1 {
					return nil, &FileParseError{
						LineNumber: lineNumber,
						Line:       originalLine,
						Err:        ErrFileParseTrailingBackslashInPattern,
					}
				}

				switch {
				// escape character and special characters need to stay escaped ("\\", "\*", ...)
				case lineAsRunes[1] == '\\' || slices.Contains(escapableSpecialCharactersInPattern, lineAsRunes[1]):
					pattern.WriteRune('\\')
					lineAsRunes = lineAsRunes[1:]

				// control characters aren't special characters in pattern syntax, so escaping should be removed.
				case slices.Contains(escapableControlCharactersInPattern, lineAsRunes[1]):
					lineAsRunes = lineAsRunes[1:]

				default:
					return nil, &FileParseError{
						LineNumber: lineNumber,
						Line:       originalLine,
						Err:        ErrFileParseInvalidEscapingInPattern,
					}
				}
			}

			pattern.WriteRune(lineAsRunes[0])
			lineAsRunes = lineAsRunes[1:]
		}

		// remove inline comment (can't be escaped in owners, only pattern supports escaping)
		if i := slices.Index(lineAsRunes, '#'); i >= 0 {
			lineAsRunes = lineAsRunes[:i]
		}

		codeOwners = append(codeOwners, Entry{
			LineNumber: lineNumber,
			Pattern:    pattern.String(),
			// could be empty list in case of removing ownership
			Owners: strings.FieldsFunc(string(lineAsRunes), isSeparator),
		})
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
	params := git.CreateReadParams(repo)
	if ref == "" {
		ref = "refs/heads/" + repo.DefaultBranch
	}
	node, err := s.getCodeOwnerFileNode(ctx, params, ref)
	if err != nil {
		return nil, fmt.Errorf("cannot get codeowner file : %w", err)
	}
	if node.Node.Mode != git.TreeNodeModeFile {
		return nil, fmt.Errorf(
			"codeowner file is of format '%s' but expected to be of format '%s'",
			node.Node.Mode,
			git.TreeNodeModeFile,
		)
	}

	output, err := s.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: params,
		SHA:        node.Node.SHA,
		SizeLimit:  maxGetContentFileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	defer func() {
		if err := output.Content.Close(); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to close blob content reader.")
		}
	}()

	content, err := io.ReadAll(output.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}

	return &File{
		Content:   string(content),
		SHA:       output.SHA.String(),
		TotalSize: output.Size,
	}, nil
}

func (s *Service) getCodeOwnerFileNode(
	ctx context.Context,
	params git.ReadParams,
	ref string,
) (*git.GetTreeNodeOutput, error) {
	// iterating over multiple possible codeowner file path to get the file
	// todo: once we have api to get multi file we can simplify
	for _, path := range s.config.FilePaths {
		node, err := s.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
			ReadParams: params,
			GitREF:     ref,
			Path:       path,
		})

		if errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error encountered retrieving codeowner : %w", err)
		}
		log.Ctx(ctx).Debug().Msgf("using codeowner file from path %s", path)
		return node, nil
	}
	// get of codeowner file gives err at all the location then returning one of the error
	return nil, fmt.Errorf("no codeowner file found: %w", ErrNotFound)
}

func (s *Service) getApplicableCodeOwnersForPR(
	ctx context.Context,
	repo *types.Repository,
	pr *types.PullReq,
) (*CodeOwners, error) {
	codeOwners, err := s.get(ctx, repo, pr.TargetBranch)
	if err != nil {
		return nil, err
	}

	diffFileStats, err := s.git.DiffFileNames(ctx, &git.DiffParams{
		ReadParams: git.CreateReadParams(repo),
		BaseRef:    pr.MergeBaseSHA,
		HeadRef:    pr.SourceSHA,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get diff file stat: %w", err)
	}

	entryIDs := map[int]struct{}{}
	for _, file := range diffFileStats.Files {
		// last rule that matches wins (hence simply go in reverse order)
		for i := len(codeOwners.Entries) - 1; i >= 0; i-- {
			pattern := codeOwners.Entries[i].Pattern
			if ok, err := match(pattern, file); err != nil {
				return nil, fmt.Errorf("failed to match pattern %q for file %q: %w", pattern, file, err)
			} else if ok {
				entryIDs[i] = struct{}{}
				break
			}
		}
	}

	filteredEntries := make([]Entry, 0, len(entryIDs))
	for i := range entryIDs {
		if !codeOwners.Entries[i].IsOwnershipReset() {
			filteredEntries = append(filteredEntries, codeOwners.Entries[i])
		}
	}

	// sort output to match order of occurrence in source CODEOWNERS file
	sort.Slice(
		filteredEntries,
		func(i, j int) bool { return filteredEntries[i].LineNumber <= filteredEntries[j].LineNumber },
	)

	return &CodeOwners{
		FileSHA: codeOwners.FileSHA,
		Entries: filteredEntries,
	}, err
}

//nolint:gocognit
func (s *Service) Evaluate(
	ctx context.Context,
	repo *types.Repository,
	pr *types.PullReq,
	reviewers []*types.PullReqReviewer,
) (*Evaluation, error) {
	owners, err := s.getApplicableCodeOwnersForPR(ctx, repo, pr)
	if err != nil {
		return &Evaluation{}, fmt.Errorf("failed to get codeOwners: %w", err)
	}

	if owners == nil || len(owners.Entries) == 0 {
		return &Evaluation{}, nil
	}

	evaluationEntries := make([]EvaluationEntry, 0, len(owners.Entries))

	for _, entry := range owners.Entries {
		ownerEvaluations := make([]OwnerEvaluation, 0, len(owners.Entries))
		userGroupOwnerEvaluations := make([]UserGroupOwnerEvaluation, 0, len(owners.Entries))
		for _, owner := range entry.Owners {
			// check for usrgrp
			if strings.HasPrefix(owner, userGroupPrefixMarker) {
				userGroupCodeOwner, err := s.resolveUserGroupCodeOwner(ctx, owner[1:], reviewers)
				if errors.Is(err, usergroup.ErrNotFound) {
					log.Ctx(ctx).Debug().Msgf("usergroup %q not found hence skipping for code owner", owner)
					continue
				}
				if err != nil {
					return nil, fmt.Errorf("error resolving usergroup :%w", err)
				}
				userGroupOwnerEvaluations = append(userGroupOwnerEvaluations, *userGroupCodeOwner)
				continue
			}
			// user email based codeowner
			userCodeOwner, err := s.resolveUserCodeOwnerByEmail(ctx, owner, reviewers)
			if errors.Is(err, gitness_store.ErrResourceNotFound) {
				log.Ctx(ctx).Debug().Msgf("user %q not found in database hence skipping for code owner", owner)
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("error resolving user by email : %w", err)
			}
			if pr.CreatedBy == userCodeOwner.Owner.ID {
				continue
			}

			ownerEvaluations = append(ownerEvaluations, *userCodeOwner)
		}
		if len(ownerEvaluations) != 0 || len(userGroupOwnerEvaluations) != 0 {
			evaluationEntries = append(evaluationEntries, EvaluationEntry{
				LineNumber:                entry.LineNumber,
				Pattern:                   entry.Pattern,
				OwnerEvaluations:          ownerEvaluations,
				UserGroupOwnerEvaluations: userGroupOwnerEvaluations,
			})
		}
	}

	return &Evaluation{
		EvaluationEntries: evaluationEntries,
		FileSha:           owners.FileSHA,
	}, nil
}

func (s *Service) resolveUserGroupCodeOwner(
	ctx context.Context,
	owner string,
	reviewers []*types.PullReqReviewer,
) (*UserGroupOwnerEvaluation, error) {
	usrgrp, err := s.userGroupResolver.Resolve(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("not able to resolve usergroup : %w", err)
	}
	userGroupEvaluation := &UserGroupOwnerEvaluation{
		Identifier: usrgrp.Identifier,
		Name:       usrgrp.Name,
	}
	ownersEvaluations := make([]OwnerEvaluation, 0, len(usrgrp.Users))
	for _, uid := range usrgrp.Users {
		pullreqReviewer := findReviewerInList("", uid, reviewers)
		// we don't append all the user of the user group in the owner evaluations and
		// append it only if it is reviewed by a user.
		if pullreqReviewer != nil {
			ownersEvaluations = append(ownersEvaluations,
				OwnerEvaluation{
					Owner:          pullreqReviewer.Reviewer,
					ReviewDecision: pullreqReviewer.ReviewDecision,
					ReviewSHA:      pullreqReviewer.SHA,
				},
			)
			continue
		}
	}
	userGroupEvaluation.Evaluations = ownersEvaluations

	return userGroupEvaluation, nil
}

func (s *Service) resolveUserCodeOwnerByEmail(
	ctx context.Context,
	owner string,
	reviewers []*types.PullReqReviewer,
) (*OwnerEvaluation, error) {
	pullreqReviewer := findReviewerInList(owner, "", reviewers)
	if pullreqReviewer != nil {
		return &OwnerEvaluation{
			Owner:          pullreqReviewer.Reviewer,
			ReviewDecision: pullreqReviewer.ReviewDecision,
			ReviewSHA:      pullreqReviewer.SHA,
		}, nil
	}
	principal, err := s.principalStore.FindByEmail(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("error finding user by email: %w", err)
	}
	return &OwnerEvaluation{
		Owner: *principal.ToPrincipalInfo(),
	}, nil
}

func (s *Service) Validate(
	ctx context.Context,
	repo *types.Repository,
	branch string,
) (*types.CodeOwnersValidation, error) {
	var codeOwnerValidation types.CodeOwnersValidation
	// check file parsing, existence and size
	codeowners, err := s.get(ctx, repo, branch)
	if err != nil {
		return nil, err
	}

	for _, entry := range codeowners.Entries {
		// check for users in file
		for _, owner := range entry.Owners {
			// todo: handle usergroup better
			if strings.HasPrefix(owner, userGroupPrefixMarker) {
				continue
			}
			_, err := s.principalStore.FindByEmail(ctx, owner)
			if errors.Is(err, gitness_store.ErrResourceNotFound) {
				codeOwnerValidation.Addf(enum.CodeOwnerViolationCodeUserNotFound,
					"user %q not found", owner)
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("error encountered fetching user %q by email: %w", owner, err)
			}
		}

		// check for pattern
		if entry.Pattern == "" {
			codeOwnerValidation.Add(enum.CodeOwnerViolationCodePatternEmpty,
				"empty pattern")
			continue
		}

		ok := doublestar.ValidatePathPattern(entry.Pattern)
		if !ok {
			codeOwnerValidation.Addf(enum.CodeOwnerViolationCodePatternInvalid, "pattern %q is invalid",
				entry.Pattern)
		}
	}

	return &codeOwnerValidation, nil
}

func findReviewerInList(email string, uid string, reviewers []*types.PullReqReviewer) *types.PullReqReviewer {
	for _, reviewer := range reviewers {
		if uid == reviewer.Reviewer.UID || email == reviewer.Reviewer.Email {
			return reviewer
		}
	}

	return nil
}

// Match matches a file path against the provided CODEOWNERS pattern.
// The code follows the .gitignore syntax closely (similar to github):
// https://git-scm.com/docs/gitignore#_pattern_format
//
// IMPORTANT: It seems that doublestar has a bug, as `*k/**` matches `k` but `k*/**` doesnt (incorrect)'.
// Because of that, we currently match patterns like `test*` only partially:
// - `test2`, `test/abc`, `test2/abc` are matching
// - `test` is not matching
// As a workaround, the user will have to add the same rule without a trailing `*` for now.
func match(pattern string, path string) (bool, error) {
	if pattern == "" {
		return false, fmt.Errorf("empty pattern not allowed")
	}
	if path == "" {
		return false, fmt.Errorf("empty path not allowed")
	}

	// catch easy cases immediately to simplify code
	if pattern == "/" || pattern == "*" || pattern == "**" {
		return true, nil
	}

	// cleanup path to simplify matching (always start with "/" and remove trailing "/")
	if path[0] != '/' {
		path = "/" + path
	}
	if path[len(path)-1] == '/' {
		path = path[0 : len(path)-1]
	}

	// if the pattern contains a slash anywhere but at the end, it's treated as an absolute path.
	// Otherwise, the pattern can match on any level.
	if !strings.Contains(pattern[:len(pattern)-1], "/") {
		pattern = "**/" + pattern
	} else if pattern[0] != '/' {
		pattern = "/" + pattern
	}

	// if the pattern ends with "/**", then it matches everything inside.
	// Since doublestar matches pattern "x/**" with target "x", we replace it with "x/*/**".
	if strings.HasSuffix(pattern, "/**") {
		pattern = pattern[:len(pattern)-3] + "/*/**"
	}

	// If CODEOWNERS matches a file, it also matches a folder with the same name, and anything inside that folder.
	// Special case is a rule ending with "/", it only matches files inside the folder, not the folder itself.
	// Since doublestar matches pattern "x/**" with target "x", we extend the pattern with "*/**" in such a case.
	// Another special case is "/*", where the user explicitly stops nested matching.
	if pattern[len(pattern)-1] == '/' {
		pattern += "*/**"
	} else if !strings.HasSuffix(pattern, "/**") && !strings.HasSuffix(pattern, "/*") {
		pattern += "/**"
	}

	match, err := doublestar.PathMatch(pattern, path)
	if err != nil {
		return false, fmt.Errorf("failed doublestar path match: %w", err)
	}

	return match, nil
}
