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

package codecomments

import (
	"context"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// Migrator is a utility used to migrate code comments after update of the pull request's source branch.
type Migrator struct {
	hunkHeaderFetcher hunkHeaderFetcher
}

type hunkHeaderFetcher interface {
	GetDiffHunkHeaders(context.Context, git.GetDiffHunkHeadersParams) (git.GetDiffHunkHeadersOutput, error)
}

// MigrateNew updates the "+" (the added lines) part of code comments
// after a new commit on the pull request's source branch.
// The parameter newSHA should contain the latest commit SHA of the pull request's source branch.
func (migrator *Migrator) MigrateNew(
	ctx context.Context,
	repoGitUID string,
	newSHA string,
	comments []*types.CodeComment,
) {
	migrator.migrate(
		ctx,
		repoGitUID,
		newSHA,
		comments,
		func(codeComment *types.CodeComment) string {
			return codeComment.SourceSHA
		},
		func(codeComment *types.CodeComment, sha string) {
			codeComment.SourceSHA = sha
		},
		func(codeComment *types.CodeComment) (int, int) {
			return codeComment.LineNew, codeComment.LineNew + codeComment.SpanNew - 1
		},
		func(codeComment *types.CodeComment, line int) {
			codeComment.LineNew += line
		},
	)
}

// MigrateOld updates the "-" (the removes lines) part of code comments
// after the pull request's change of the merge base commit.
func (migrator *Migrator) MigrateOld(
	ctx context.Context,
	repoGitUID string,
	newSHA string,
	comments []*types.CodeComment,
) {
	migrator.migrate(
		ctx,
		repoGitUID,
		newSHA,
		comments,
		func(codeComment *types.CodeComment) string {
			return codeComment.MergeBaseSHA
		},
		func(codeComment *types.CodeComment, sha string) {
			codeComment.MergeBaseSHA = sha
		},
		func(codeComment *types.CodeComment) (int, int) {
			return codeComment.LineOld, codeComment.LineOld + codeComment.SpanOld - 1
		},
		func(codeComment *types.CodeComment, line int) {
			codeComment.LineOld += line
		},
	)
}

//nolint:gocognit,funlen // refactor if needed
func (migrator *Migrator) migrate(
	ctx context.Context,
	repoGitUID string,
	newSHA string,
	comments []*types.CodeComment,
	getSHA func(codeComment *types.CodeComment) string,
	setSHA func(codeComment *types.CodeComment, sha string),
	getCommentStartEnd func(codeComment *types.CodeComment) (int, int),
	updateCommentLine func(codeComment *types.CodeComment, line int),
) {
	if len(comments) == 0 {
		return
	}

	commitMap, initialValuesMap := mapCodeComments(comments, getSHA)

	for commentSHA, fileMap := range commitMap {
		// get all hunk headers for the diff between the SHA that's stored in the comment and the new SHA.
		diffSummary, errDiff := migrator.hunkHeaderFetcher.GetDiffHunkHeaders(ctx, git.GetDiffHunkHeadersParams{
			ReadParams: git.ReadParams{
				RepoUID: repoGitUID,
			},
			SourceCommitSHA: commentSHA,
			TargetCommitSHA: newSHA,
		})
		if errors.AsStatus(errDiff) == errors.StatusNotFound {
			// Handle the commit SHA not found error and mark all code comments as outdated.
			for _, codeComments := range fileMap {
				for _, codeComment := range codeComments {
					codeComment.Outdated = true
				}
			}
			continue
		}
		if errDiff != nil {
			log.Ctx(ctx).Err(errDiff).
				Msgf("failed to get git diff between comment's sha %s and the latest %s", commentSHA, newSHA)
			continue
		}

		// Traverse all the changed files
		for _, file := range diffSummary.Files {
			var codeComments []*types.CodeComment

			codeComments = fileMap[file.FileHeader.OldName]

			// Handle file renames
			if file.FileHeader.OldName != file.FileHeader.NewName {
				if len(codeComments) == 0 {
					// If the code comments are not found using the old name of the file, try with the new name.
					codeComments = fileMap[file.FileHeader.NewName]
				} else {
					// Update the code comment's path to the new file name
					for _, cc := range codeComments {
						cc.Path = file.FileHeader.NewName
					}
				}
			}

			// Handle file delete
			if _, isDeleted := file.FileHeader.Extensions[gitenum.DiffExtHeaderDeletedFileMode]; isDeleted {
				for _, codeComment := range codeComments {
					codeComment.Outdated = true
				}
				continue
			}

			// Handle new files - shouldn't happen because no code comments should exist for a non-existing file.
			if _, isAdded := file.FileHeader.Extensions[gitenum.DiffExtHeaderNewFileMode]; isAdded {
				for _, codeComment := range codeComments {
					codeComment.Outdated = true
				}
				continue
			}

			for hunkIdx := len(file.HunkHeaders) - 1; hunkIdx >= 0; hunkIdx-- {
				hunk := file.HunkHeaders[hunkIdx]

				for _, cc := range codeComments {
					if cc.Outdated {
						continue
					}

					ccStart, ccEnd := getCommentStartEnd(cc)
					outdated, moveDelta := processCodeComment(ccStart, ccEnd, hunk)
					if outdated {
						cc.CodeCommentFields = initialValuesMap[cc.ID] // revert the CC to the original values
						cc.Outdated = true
						continue
					}

					updateCommentLine(cc, moveDelta)
				}
			}
		}

		for _, codeComments := range fileMap {
			for _, codeComment := range codeComments {
				if codeComment.Outdated {
					continue
				}
				setSHA(codeComment, newSHA)
			}
		}
	}
}

// mapCodeComments groups code comments to maps, first by commit SHA and then by file name.
// It assumes the incoming list is already sorted.
func mapCodeComments(
	comments []*types.CodeComment,
	extractSHA func(*types.CodeComment) string,
) (map[string]map[string][]*types.CodeComment, map[int64]types.CodeCommentFields) {
	commitMap := map[string]map[string][]*types.CodeComment{}
	originalComments := make(map[int64]types.CodeCommentFields, len(comments))

	for _, comment := range comments {
		commitSHA := extractSHA(comment)

		fileMap := commitMap[commitSHA]
		if fileMap == nil {
			fileMap = map[string][]*types.CodeComment{}
		}

		fileComments := fileMap[comment.Path]
		fileComments = append(fileComments, comment)
		fileMap[comment.Path] = fileComments

		commitMap[commitSHA] = fileMap

		originalComments[comment.ID] = comment.CodeCommentFields
	}

	return commitMap, originalComments
}

func processCodeComment(ccStart, ccEnd int, h git.HunkHeader) (outdated bool, moveDelta int) {
	// A code comment is marked as outdated if:
	// * The code lines covered by the code comment are changed
	//   (the range given by the OldLine/OldSpan overlaps the code comment's code range)
	// * There are new lines inside the line range covered by the code comment, don't care about how many
	//   (the NewLine is between the CC start and CC end; the value of the NewSpan is unimportant).
	outdated =
		(h.OldSpan > 0 && ccEnd >= h.OldLine && ccStart <= h.OldLine+h.OldSpan-1) || // code comment's code is changed
			(h.NewSpan > 0 && h.NewLine > ccStart && h.NewLine <= ccEnd) // lines are added inside the code comment

	if outdated {
		return // outdated comments aren't moved
	}

	if ccEnd <= h.OldLine {
		return // the change described by the hunk header is below the code comment, so it doesn't affect it
	}

	moveDelta = h.NewSpan - h.OldSpan

	return
}
