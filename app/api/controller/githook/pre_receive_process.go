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

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
)

func (c *Controller) processObjects(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	principal *types.Principal,
	refUpdates changedRefs,
	checks *protectionChecks,
	violationsInput *protection.PushViolationsInput,
	settingsViolations *settingsViolations,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) error {
	if refUpdates.hasOnlyDeletedBranches() {
		return nil
	}

	sizeLimit := checks.SettingsFileSizeLimit
	if checks.RulesFileSizeLimit != 0 {
		if sizeLimit == 0 || checks.RulesFileSizeLimit < sizeLimit {
			sizeLimit = checks.RulesFileSizeLimit
		}
	}

	principalCommitterMatch := checks.SettingsPrincipalCommitterMatch || checks.RulesPrincipalCommitterMatch

	if sizeLimit == 0 && !principalCommitterMatch && !checks.SettingsGitLFSEnabled {
		return nil
	}

	preReceiveObjsIn := git.ProcessPreReceiveObjectsParams{
		ReadParams: git.ReadParams{
			RepoUID:             repo.GitUID,
			AlternateObjectDirs: in.Environment.AlternateObjectDirs,
		},
	}

	if sizeLimit > 0 {
		preReceiveObjsIn.FindOversizeFilesParams = &git.FindOversizeFilesParams{
			SizeLimit: sizeLimit,
		}
	}

	if principalCommitterMatch && principal != nil && !in.Internal {
		preReceiveObjsIn.FindCommitterMismatchParams = &git.FindCommitterMismatchParams{
			PrincipalEmail: principal.Email,
		}
	}

	if checks.SettingsGitLFSEnabled {
		preReceiveObjsIn.FindLFSPointersParams = &git.FindLFSPointersParams{}
	}

	preReceiveObjsOut, err := rgit.ProcessPreReceiveObjects(
		ctx,
		preReceiveObjsIn,
	)
	if err != nil {
		return fmt.Errorf("failed to process pre-receive objects: %w", err)
	}

	if preReceiveObjsOut.FindOversizeFilesOutput != nil &&
		len(preReceiveObjsOut.FindOversizeFilesOutput.FileInfos) > 0 {
		printOversizeFiles(
			output,
			preReceiveObjsOut.FindOversizeFilesOutput.FileInfos,
			preReceiveObjsOut.FindOversizeFilesOutput.Total,
			sizeLimit,
		)
		if checks.SettingsFileSizeLimit > 0 {
			settingsViolations.FileSizeLimitExceeded = true
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
		if checks.SettingsPrincipalCommitterMatch {
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
			printLFSPointers(
				output,
				preReceiveObjsOut.FindLFSPointersOutput.LFSInfos,
				preReceiveObjsOut.FindLFSPointersOutput.Total,
			)

			if checks.SettingsGitLFSEnabled {
				settingsViolations.UnknownLFSObjectsFound = true
			}
		}
	}

	violationsInput.FindOversizeFilesOutput = preReceiveObjsOut.FindOversizeFilesOutput
	if preReceiveObjsOut.FindCommitterMismatchOutput != nil {
		violationsInput.CommitterMismatchCount = preReceiveObjsOut.FindCommitterMismatchOutput.Total
	}

	return nil
}
