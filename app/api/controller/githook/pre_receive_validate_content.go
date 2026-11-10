// Copyright 2026 Harness, Inc.
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

package githook

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// maxValidatedBlobSize is one byte more than the largest file a content validator accepts (4 MB), so an oversized
// file is read just far enough for the validator to see that it is too large without slurping it whole.
const maxValidatedBlobSize = 4*1024*1024 + 1

// validateChangedContent validates the registered files that a push adds or changes. It is advisory: it returns
// no error, and every failure is logged with that file left unvalidated, so it can never fail a push.
func (c *Controller) validateChangedContent(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	in types.GithookPreReceiveInput,
) []types.ContentValidation {
	if c.contentValidators == nil {
		return nil
	}
	// Only a direct git push gets content warnings. The commit API (with or without bypass_rules) validates its
	// own content and returns the result in its response, and system-initiated pushes - linked-repo sync, merge
	// queue, system refs, repo lifecycle - have no human pusher to read a warning, so validating them would
	// spend tree reads and owner lookups on messages nobody sees.
	if in.OperationType != enum.GitOpTypeGitPush {
		return nil
	}

	readParams := git.ReadParams{
		RepoUID:             repo.GitUID,
		AlternateObjectDirs: in.Environment.AlternateObjectDirs,
	}

	var result []types.ContentValidation
	validated := map[string]bool{}

	for _, refUpdate := range in.RefUpdates {
		if refUpdate.New.IsNil() || !isBranch(refUpdate.Ref) {
			continue
		}

		for _, path := range c.contentValidators.Paths() {
			// A new branch is compared against the default branch, so a branch cut from a repo whose CODEOWNERS
			// is already invalid does not warn about a file the pusher never touched.
			baseRef := repo.DefaultBranch
			if !refUpdate.Old.IsNil() {
				baseRef = refUpdate.Old.String()
			}

			blobSHA, ok := changedFileSHA(ctx, rgit, readParams, path, baseRef, refUpdate.New.String())
			if !ok {
				continue
			}

			// Dedupe on content, not on path: a push that updates the same file on several branches must
			// validate each distinct version, while a stack of branches carrying one identical version is
			// read and warned about once instead of repeating the same warning per branch.
			key := path + "\x00" + blobSHA
			if validated[key] {
				continue
			}
			validated[key] = true

			content, ok := fileContent(ctx, rgit, readParams, path, blobSHA)
			if !ok {
				continue
			}

			violations, err := c.contentValidators.Validate(ctx, repo, path, content)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Str("path", path).Msg("content validation failed, skipping")
				continue
			}
			if len(violations) > 0 {
				result = append(result, types.ContentValidation{Path: path, Violations: violations})
			}
		}
	}

	return result
}

// changedFileSHA returns the blob SHA of the file at path on newRef, provided it exists there and differs from its
// version on baseRef (skipped when baseRef is empty). ok is false if the file is absent, unchanged, or could not
// be resolved.
func changedFileSHA(
	ctx context.Context,
	rgit RestrictedGIT,
	readParams git.ReadParams,
	path, baseRef, newRef string,
) (string, bool) {
	newNode, err := rgit.GetTreeNode(ctx, &git.GetTreeNodeParams{ReadParams: readParams, GitREF: newRef, Path: path})
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Ctx(ctx).Warn().Err(err).Str("path", path).Msg("failed to read file for content validation")
		}
		return "", false
	}

	if baseRef != "" {
		oldNode, err := rgit.GetTreeNode(ctx, &git.GetTreeNodeParams{ReadParams: readParams, GitREF: baseRef, Path: path})
		switch {
		case err == nil && oldNode.Node.SHA == newNode.Node.SHA:
			return "", false
		case err != nil && !errors.IsNotFound(err):
			log.Ctx(ctx).Warn().Err(err).Str("path", path).Msg("failed to read previous file for content validation")
			return "", false
		}
	}

	return newNode.Node.SHA, true
}

// fileContent reads the blob, up to the largest size a validator accepts. ok is false if it could not be read.
func fileContent(
	ctx context.Context,
	rgit RestrictedGIT,
	readParams git.ReadParams,
	path, blobSHA string,
) ([]byte, bool) {
	blob, err := rgit.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: readParams,
		SHA:        blobSHA,
		SizeLimit:  maxValidatedBlobSize,
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("path", path).Msg("failed to read blob for content validation")
		return nil, false
	}
	defer func() { _ = blob.Content.Close() }()

	content, err := io.ReadAll(blob.Content)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("path", path).Msg("failed to read blob content for content validation")
		return nil, false
	}
	return content, true
}

// maxContentValidationMessages caps the warning lines a single push prints. A file whose owners were all
// deprovisioned can yield thousands of violations, and they would be echoed on every push until it is fixed.
const maxContentValidationMessages = 30

// contentValidationMessages renders validations as the lines printed to the pusher, capped at
// maxContentValidationMessages plus a line counting the remainder.
func contentValidationMessages(validations []types.ContentValidation) []string {
	var messages []string
	var omitted int
	for _, validation := range validations {
		for _, violation := range validation.Violations {
			if len(messages) == maxContentValidationMessages {
				omitted++
				continue
			}
			if violation.Line > 0 {
				messages = append(messages, fmt.Sprintf("Warning: %s line %d: %s",
					validation.Path, violation.Line, violation.Message))
				continue
			}
			messages = append(messages, fmt.Sprintf("Warning: %s: %s", validation.Path, violation.Message))
		}
	}
	if omitted > 0 {
		messages = append(messages, fmt.Sprintf("Warning: and %d more problem(s) not shown", omitted))
	}
	return messages
}
