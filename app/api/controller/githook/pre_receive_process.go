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

package githook

import (
	"context"
	"fmt"
	"slices"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
)

func (c *Controller) processObjects(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	principal *types.Principal,
	refUpdates changedRefs,
	violationsInput *protection.PushViolationsInput,
	settingsChecks repoSettings,
	settingsViolations *repoSettingsViolations,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) error {
	if refUpdates.hasOnlyDeletedBranches() {
		return nil
	}

	var sizeLimits []int64

	for _, limit := range violationsInput.FileSizeLimits {
		if limit > 0 {
			sizeLimits = append(sizeLimits, limit)
		}
	}

	if limit := settingsChecks.fileSizeLimit(len(sizeLimits) > 0); limit > 0 {
		sizeLimits = append(sizeLimits, limit)
	}

	if len(sizeLimits) > 0 {
		slices.Sort(sizeLimits)
		sizeLimits = slices.Compact(sizeLimits)
	}

	preReceiveObjsIn := git.ProcessPreReceiveObjectsParams{
		ReadParams: git.ReadParams{
			RepoUID:             repo.GitUID,
			AlternateObjectDirs: in.Environment.AlternateObjectDirs,
		},
	}

	if len(sizeLimits) > 0 {
		preReceiveObjsIn.FindOversizeFilesParams = &git.FindOversizeFilesParams{
			SizeLimit:  sizeLimits[0], // deprecated: min of all limits for backward compatibility
			SizeLimits: sizeLimits,
		}
	}

	if (settingsChecks.PrincipalCommitterMatch || violationsInput.PrincipalCommitterMatch) &&
		!in.Internal {
		preReceiveObjsIn.FindCommitterMismatchParams = &git.FindCommitterMismatchParams{
			PrincipalEmail: principal.Email,
		}
	}

	if settingsChecks.GitLFSEnabled {
		preReceiveObjsIn.FindLFSPointersParams = &git.FindLFSPointersParams{}
	}

	preReceiveObjsOut, err := rgit.ProcessPreReceiveObjects(
		ctx,
		preReceiveObjsIn,
	)
	if err != nil {
		return fmt.Errorf("failed to process pre-receive objects: %w", err)
	}

	if out := preReceiveObjsOut.FindOversizeFilesOutput; out != nil && len(out.TotalsPerLimit) > 0 {
		printOversizeFiles(output, out)

		violationsInput.FindOversizeFilesOutput = out

		if limit := settingsChecks.fileSizeLimit(len(violationsInput.FileSizeLimits) > 0); limit > 0 {
			if out.AccumulatedTotal(limit) > 0 {
				settingsViolations.ExceededFileSizeLimit = limit
			}
		}
	}

	if preReceiveObjsOut.FindCommitterMismatchOutput != nil &&
		len(preReceiveObjsOut.FindCommitterMismatchOutput.CommitInfos) > 0 {
		printCommitterMismatch(
			output,
			preReceiveObjsOut.FindCommitterMismatchOutput.CommitInfos,
			preReceiveObjsIn.FindCommitterMismatchParams.PrincipalEmail,
			preReceiveObjsOut.FindCommitterMismatchOutput.Total,
		)

		violationsInput.CommitterMismatchCount = preReceiveObjsOut.FindCommitterMismatchOutput.Total

		if settingsChecks.PrincipalCommitterMatch {
			settingsViolations.CommitterMismatchFound = true
		}
	}

	if preReceiveObjsOut.FindLFSPointersOutput != nil &&
		len(preReceiveObjsOut.FindLFSPointersOutput.LFSInfos) > 0 {
		objIDs := make([]string, len(preReceiveObjsOut.FindLFSPointersOutput.LFSInfos))
		for i, info := range preReceiveObjsOut.FindLFSPointersOutput.LFSInfos {
			objIDs[i] = info.ObjID
		}

		existingObjs, err := c.lfsStore.FindMany(ctx, in.RepoID, objIDs)
		if err != nil {
			return fmt.Errorf("failed to find lfs objects: %w", err)
		}

		//nolint:lll
		if len(existingObjs) != len(objIDs) {
			output.Error = ptr.String(
				"Changes blocked by unknown Git LFS objects. Please try `git lfs push --all` or check if LFS is setup properly.")
			printLFSPointers(
				output,
				preReceiveObjsOut.FindLFSPointersOutput.LFSInfos,
				preReceiveObjsOut.FindLFSPointersOutput.Total,
			)

			if settingsChecks.GitLFSEnabled {
				settingsViolations.UnknownLFSObjectsFound = true
			}
		}
	}

	violationsInput.FindOversizeFilesOutput = preReceiveObjsOut.FindOversizeFilesOutput

	return nil
}
