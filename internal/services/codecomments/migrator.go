// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package codecomments

import (
	"context"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// Migrator is a utility used to migrate code comments after update of the pull request's source branch.
type Migrator struct {
	gitRPCClient gitrpc.Interface
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

//nolint:gocognit // refactor if needed
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

	commitMap := mapCodeComments(comments, getSHA)

	for commentSHA, fileMap := range commitMap {
		// get all hunk headers for the diff between the SHA that's stored in the comment and the new SHA.
		diffSummary, errDiff := migrator.gitRPCClient.GetDiffHunkHeaders(ctx, gitrpc.GetDiffHunkHeadersParams{
			ReadParams: gitrpc.ReadParams{
				RepoUID: repoGitUID,
			},
			SourceCommitSHA: commentSHA,
			TargetCommitSHA: newSHA,
		})
		if gitrpc.ErrorStatus(errDiff) == gitrpc.StatusNotFound {
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
			if len(file.HunkHeaders) == 1 && file.HunkHeaders[0].NewLine == 0 && file.HunkHeaders[0].NewSpan == 0 {
				for _, codeComment := range codeComments {
					codeComment.Outdated = true
				}
				continue
			}

			for _, hunk := range file.HunkHeaders {
				hunkStart := hunk.NewLine
				hunkEnd := hunk.NewLine + hunk.NewSpan - 1
				for _, cc := range codeComments {
					commentStart, commentEnd := getCommentStartEnd(cc)
					if commentEnd < hunkStart {
						continue
					}

					outdated := commentStart <= hunkEnd
					cc.Outdated = outdated

					if outdated {
						continue
					}

					updateCommentLine(cc, hunk.NewSpan-hunk.OldSpan)
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
) map[string]map[string][]*types.CodeComment {
	commitMap := map[string]map[string][]*types.CodeComment{}
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
	}

	return commitMap
}
