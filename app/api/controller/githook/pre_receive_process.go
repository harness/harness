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

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
)

func (c *Controller) processObjects(
	ctx context.Context,
	repo *types.RepositoryCore,
	principal *types.Principal,
	refUpdates changedRefs,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) error {
	if refUpdates.hasOnlyDeletedBranches() {
		return nil
	}

	var sizeLimit int64
	var err error
	sizeLimit, err = settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyFileSizeLimit,
		settings.DefaultFileSizeLimit,
	)
	if err != nil {
		return fmt.Errorf("failed to check settings for file size limit: %w", err)
	}

	principalCommitterMatch, err := settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyPrincipalCommitterMatch,
		settings.DefaultPrincipalCommitterMatch,
	)
	if err != nil {
		return fmt.Errorf("failed to check settings for principal committer match: %w", err)
	}

	gitLFSEnabled, err := settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyGitLFSEnabled,
		settings.DefaultGitLFSEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to check settings for Git LFS enabled: %w", err)
	}

	if sizeLimit == 0 && !principalCommitterMatch && !gitLFSEnabled {
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

	if principalCommitterMatch && principal != nil {
		preReceiveObjsIn.FindCommitterMismatchParams = &git.FindCommitterMismatchParams{
			PrincipalEmail: principal.Email,
		}
	}

	if gitLFSEnabled {
		preReceiveObjsIn.FindLFSPointersParams = &git.FindLFSPointersParams{}
	}

	preReceiveObjsOut, err := c.git.ProcessPreReceiveObjects(
		ctx,
		preReceiveObjsIn,
	)
	if err != nil {
		return fmt.Errorf("failed to process pre-receive objects: %w", err)
	}

	if preReceiveObjsOut.FindOversizeFilesOutput != nil &&
		len(preReceiveObjsOut.FindOversizeFilesOutput.FileInfos) > 0 {
		output.Error = ptr.String("Changes blocked by files exceeding the file size limit")
		printOversizeFiles(
			output,
			preReceiveObjsOut.FindOversizeFilesOutput.FileInfos,
			preReceiveObjsOut.FindOversizeFilesOutput.Total,
			sizeLimit,
		)
	}

	if preReceiveObjsOut.FindCommitterMismatchOutput != nil &&
		len(preReceiveObjsOut.FindCommitterMismatchOutput.CommitInfos) > 0 {
		output.Error = ptr.String("Committer verification failed: authenticated user and committer must match")
		printCommitterMismatch(
			output,
			preReceiveObjsOut.FindCommitterMismatchOutput.CommitInfos,
			preReceiveObjsIn.FindCommitterMismatchParams.PrincipalEmail,
			preReceiveObjsOut.FindCommitterMismatchOutput.Total,
		)
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
		}
	}

	return nil
}
